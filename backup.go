package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ImportResult struct {
	Format  string `json:"format"`
	Added   int    `json:"added"`
	Skipped int    `json:"skipped"`
	Invalid int    `json:"invalid"`
}

const maxImportSize = 20 << 20 // 20 MB

// ====================== Utilidades ======================

var accentFix = strings.NewReplacer(
	"á", "a", "à", "a", "ã", "a", "â", "a",
	"é", "e", "ê", "e", "í", "i",
	"ó", "o", "ô", "o", "õ", "o",
	"ú", "u", "ç", "c",
)

func normHeader(s string) string {
	return accentFix.Replace(strings.ToLower(strings.TrimSpace(s)))
}

func categoryLabel(key string) string {
	for _, c := range categories {
		if c.Key == key {
			return c.Label
		}
	}
	return "Outros"
}

// Aceita a chave ("banco") ou o nome ("Bancos e finanças"); senão detecta pelo nome
func (a *App) resolveCategory(raw, name string) string {
	r := strings.ToLower(strings.TrimSpace(raw))
	if r != "" {
		for _, c := range categories {
			if r == c.Key || r == strings.ToLower(c.Label) {
				return c.Key
			}
		}
	}
	return a.DetectCategory(name)
}

func (a *App) sortedEntries() []Entry {
	out := make([]Entry, len(a.entries))
	copy(out, a.entries)
	order := map[string]int{}
	for i, c := range categories {
		order[c.Key] = i
	}
	sort.SliceStable(out, func(i, j int) bool {
		oi, oj := order[out[i].Category], order[out[j].Category]
		if oi != oj {
			return oi < oj
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func entryKey(e Entry) string {
	return strings.ToLower(e.Name) + "\x00" + strings.ToLower(e.Username) + "\x00" + e.Password
}

// Lê UTF-8, UTF-8 com BOM, UTF-16 (Bloco de Notas) e ANSI/Latin-1 (Excel antigo)
func toUTF8(raw []byte) string {
	if len(raw) >= 2 && ((raw[0] == 0xFF && raw[1] == 0xFE) || (raw[0] == 0xFE && raw[1] == 0xFF)) {
		le := raw[0] == 0xFF
		u := make([]uint16, 0, len(raw)/2)
		for i := 2; i+1 < len(raw); i += 2 {
			if le {
				u = append(u, uint16(raw[i])|uint16(raw[i+1])<<8)
			} else {
				u = append(u, uint16(raw[i])<<8|uint16(raw[i+1]))
			}
		}
		return string(utf16.Decode(u))
	}
	raw = bytes.TrimPrefix(raw, []byte("\xEF\xBB\xBF"))
	if utf8.Valid(raw) {
		return string(raw)
	}
	rs := make([]rune, len(raw))
	for i, b := range raw {
		rs[i] = rune(b)
	}
	return string(rs)
}

// ====================== EXPORTAR ======================

// Devolve o caminho salvo, ou "" se o usuário cancelou.
func (a *App) ExportEntries(format, master string) (string, error) {
	if !a.unlocked {
		return "", errors.New("cofre travado")
	}
	if subtle.ConstantTimeCompare([]byte(master), []byte(a.masterPass)) != 1 {
		return "", errors.New("senha mestra incorreta")
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "pdf" && format != "csv" && format != "txt" {
		return "", errors.New("formato inválido")
	}
	if len(a.entries) == 0 {
		return "", errors.New("não há senhas para exportar")
	}

	var data []byte
	var err error
	switch format {
	case "csv":
		data, err = a.buildCSV()
	case "txt":
		data = a.buildTXT()
	case "pdf":
		data, err = a.buildPDF()
	}
	if err != nil {
		return "", err
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Salvar exportação do golock",
		DefaultFilename: "golock-senhas-" + time.Now().Format("2006-01-02") + "." + format,
		Filters: []runtime.FileFilter{
			{DisplayName: strings.ToUpper(format) + " (*." + format + ")", Pattern: "*." + format},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	if strings.ToLower(filepath.Ext(path)) != "."+format {
		path += "." + format
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("não foi possível salvar: %w", err)
	}
	return path, nil
}

func (a *App) buildCSV() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // BOM: o Excel abre os acentos certos
	w := csv.NewWriter(&buf)
	w.UseCRLF = true
	if err := w.Write([]string{"nome", "usuario", "senha", "categoria"}); err != nil {
		return nil, err
	}
	for _, e := range a.sortedEntries() {
		if err := w.Write([]string{e.Name, e.Username, e.Password, categoryLabel(e.Category)}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func (a *App) buildTXT() []byte {
	var b strings.Builder
	sep := strings.Repeat("=", 48) + "\r\n"
	b.WriteString("golock - exportação de senhas\r\n")
	fmt.Fprintf(&b, "Gerado em: %s\r\n", time.Now().Format("02/01/2006 15:04"))
	fmt.Fprintf(&b, "Total: %d\r\n", len(a.entries))
	b.WriteString("ATENÇÃO: este arquivo NÃO é criptografado. Guarde com cuidado.\r\n\r\n")
	for _, e := range a.sortedEntries() {
		b.WriteString(sep)
		fmt.Fprintf(&b, "Nome: %s\r\n", e.Name)
		fmt.Fprintf(&b, "Usuário: %s\r\n", e.Username)
		fmt.Fprintf(&b, "Senha: %s\r\n", e.Password)
		fmt.Fprintf(&b, "Categoria: %s\r\n", categoryLabel(e.Category))
	}
	b.WriteString(sep)
	return append([]byte("\xEF\xBB\xBF"), b.String()...)
}

// ---------- PDF (gerado à mão, sem bibliotecas) ----------
// O PDF tem a lista bonita para ler E uma cópia dos dados escondida dentro dele,
// que o golock usa para importar de volta sem perder nenhum caractere.

const (
	pdfW      = 595.0
	pdfH      = 842.0
	pdfMargin = 48.0
)

type pdfLayout struct {
	pages []*bytes.Buffer
	cur   *bytes.Buffer
	y     float64
}

func (l *pdfLayout) newPage() {
	l.cur = &bytes.Buffer{}
	l.pages = append(l.pages, l.cur)
	l.y = pdfH - 56
}

func (l *pdfLayout) text(font string, size, x, y float64, color, s string) {
	fmt.Fprintf(l.cur, "BT /%s %.1f Tf %s rg %.2f %.2f Td (", font, size, color, x, y)
	l.cur.Write(pdfEscape(winAnsi(s)))
	l.cur.WriteString(") Tj ET\n")
}

func (l *pdfLayout) rect(x, y, w, h float64, color string) {
	fmt.Fprintf(l.cur, "%s rg %.2f %.2f %.2f %.2f re f\n", color, x, y, w, h)
}

// Converte para a codificação das fontes padrão do PDF (WinAnsi)
func winAnsi(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 32 && r < 127:
			out = append(out, byte(r))
		case r >= 160 && r < 256:
			out = append(out, byte(r))
		case r == '€':
			out = append(out, 0x80)
		case r == '‘':
			out = append(out, 0x91)
		case r == '’':
			out = append(out, 0x92)
		case r == '“':
			out = append(out, 0x93)
		case r == '”':
			out = append(out, 0x94)
		case r == '•':
			out = append(out, 0x95)
		case r == '–':
			out = append(out, 0x96)
		case r == '—':
			out = append(out, 0x97)
		default:
			out = append(out, '?')
		}
	}
	return out
}

func pdfEscape(b []byte) []byte {
	out := make([]byte, 0, len(b)+8)
	for _, c := range b {
		switch {
		case c == '(' || c == ')' || c == '\\':
			out = append(out, '\\', c)
		case c < 32 || c > 126:
			out = fmt.Appendf(out, "\\%03o", c)
		default:
			out = append(out, c)
		}
	}
	return out
}

func wrapText(s string, n int) []string {
	r := []rune(s)
	if len(r) == 0 {
		return []string{""}
	}
	var out []string
	for len(r) > n {
		out = append(out, string(r[:n]))
		r = r[n:]
	}
	return append(out, string(r))
}

func streamObj(b []byte) []byte {
	out := fmt.Appendf(nil, "<< /Length %d >>\nstream\n", len(b))
	out = append(out, b...)
	return append(out, "\nendstream"...)
}

func (a *App) buildPDF() ([]byte, error) {
	entries := a.sortedEntries()
	l := &pdfLayout{}
	l.newPage()

	const (
		purple = "0.43 0.16 0.85"
		gold   = "0.72 0.56 0.12"
		grey   = "0.42 0.39 0.52"
		dark   = "0.11 0.07 0.19"
		warn   = "0.6 0.35 0"
	)
	boxW := pdfW - 2*pdfMargin

	// Cabeçalho
	l.text("F2", 26, pdfMargin, l.y, purple, "golock")
	l.y -= 20
	l.text("F1", 10, pdfMargin, l.y, grey,
		fmt.Sprintf("Cofre de senhas · %d itens · gerado em %s", len(entries), time.Now().Format("02/01/2006 15:04")))
	l.y -= 18
	l.rect(pdfMargin, l.y-32, boxW, 36, "1 0.96 0.84")
	l.text("F2", 9, pdfMargin+10, l.y-12, warn, "ATENÇÃO: este documento NÃO é criptografado e mostra todas as senhas em texto puro.")
	l.text("F1", 9, pdfMargin+10, l.y-25, warn, "Guarde em local seguro e apague quando não precisar mais.")
	l.y -= 56

	// Lista agrupada por categoria
	lastCat := ""
	for _, e := range entries {
		nameLines := wrapText(e.Name, 58)
		userText := e.Username
		if userText == "" {
			userText = "-"
		}
		userLines := wrapText(userText, 70)
		passLines := wrapText(e.Password, 70)

		rows := len(nameLines) + len(userLines) + len(passLines)
		h := float64(rows)*13 + 14
		need := h + 10
		newCat := e.Category != lastCat
		if newCat {
			need += 30
		}
		if l.y-need < pdfMargin+20 {
			l.newPage()
		}
		if newCat {
			l.text("F2", 10, pdfMargin, l.y-10, gold, strings.ToUpper(categoryLabel(e.Category)))
			l.y -= 28
			lastCat = e.Category
		}

		top := l.y
		l.rect(pdfMargin, top-h, boxW, h, "0.965 0.955 1")
		ty := top - 15
		for _, s := range nameLines {
			l.text("F2", 11, pdfMargin+10, ty, dark, s)
			ty -= 13
		}
		for i, s := range userLines {
			if i == 0 {
				l.text("F1", 8.5, pdfMargin+10, ty, grey, "Usuário")
			}
			l.text("F3", 9.5, pdfMargin+70, ty, dark, s)
			ty -= 13
		}
		for i, s := range passLines {
			if i == 0 {
				l.text("F1", 8.5, pdfMargin+10, ty, grey, "Senha")
			}
			l.text("F3", 9.5, pdfMargin+70, ty, dark, s)
			ty -= 13
		}
		l.y = top - h - 8
	}

	// Rodapé com número da página
	total := len(l.pages)
	for i, p := range l.pages {
		l.cur = p
		l.text("F1", 8, pdfMargin, 26, grey, fmt.Sprintf("golock · página %d de %d", i+1, total))
	}

	// Cópia dos dados, usada na importação
	payload, err := json.Marshal(struct {
		V       int     `json:"v"`
		Entries []Entry `json:"entries"`
	}{1, entries})
	if err != nil {
		return nil, err
	}
	hidden := []byte("GOLOCK-BEGIN:" + base64.StdEncoding.EncodeToString(payload) + ":GOLOCK-END")

	// Montagem dos objetos do PDF
	kids := make([]string, 0, total)
	for i := 0; i < total; i++ {
		kids = append(kids, fmt.Sprintf("%d 0 R", 8+2*i))
	}
	objs := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R /GolockData 6 0 R >>"),
		fmt.Appendf(nil, "<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), total),
		[]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>"),
		[]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>"),
		[]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Courier /Encoding /WinAnsiEncoding >>"),
		streamObj(hidden),
		[]byte("<< /Title (golock - senhas) /Producer (golock) >>"),
	}
	for i, p := range l.pages {
		objs = append(objs,
			fmt.Appendf(nil, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 3 0 R /F2 4 0 R /F3 5 0 R >> >> /Contents %d 0 R >>", 9+2*i),
			streamObj(p.Bytes()),
		)
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n", i+1)
		out.Write(o)
		out.WriteString("\nendobj\n")
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R /Info 7 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return out.Bytes(), nil
}

// ====================== IMPORTAR ======================

func (a *App) ImportEntries() (ImportResult, error) {
	var res ImportResult
	if !a.unlocked {
		return res, errors.New("cofre travado")
	}

	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Escolha o arquivo para importar",
		Filters: []runtime.FileFilter{
			{DisplayName: "Senhas (*.csv, *.txt, *.pdf)", Pattern: "*.csv;*.txt;*.pdf"},
		},
	})
	if err != nil {
		return res, err
	}
	if path == "" {
		return res, nil // cancelou
	}

	info, err := os.Stat(path)
	if err != nil {
		return res, err
	}
	if info.Size() > maxImportSize {
		return res, errors.New("arquivo grande demais (máximo 20 MB)")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return res, err
	}

	var list []Entry
	switch strings.ToLower(filepath.Ext(path)) {
	case ".csv":
		res.Format = "CSV"
		list, err = parseCSV(raw)
	case ".txt":
		res.Format = "TXT"
		list = parseTXT(raw)
	case ".pdf":
		res.Format = "PDF"
		list, err = parsePDF(raw)
	default:
		return ImportResult{}, errors.New("formato não suportado: use .csv, .txt ou .pdf")
	}
	if err != nil {
		return ImportResult{}, err
	}
	if len(list) == 0 {
		return ImportResult{}, errors.New("não encontrei nenhuma senha nesse arquivo")
	}

	// Nada é apagado nem sobrescrito: só entra o que ainda não existe
	seen := map[string]bool{}
	for _, e := range a.entries {
		seen[entryKey(e)] = true
	}
	for _, e := range list {
		e.Name = strings.TrimSpace(e.Name)
		e.Username = strings.TrimSpace(e.Username)
		if e.Name == "" || e.Password == "" {
			res.Invalid++
			continue
		}
		e.Category = a.resolveCategory(e.Category, e.Name)
		k := entryKey(e)
		if seen[k] {
			res.Skipped++
			continue
		}
		seen[k] = true
		e.ID = a.nextID
		a.nextID++
		a.entries = append(a.entries, e)
		res.Added++
	}
	if res.Added > 0 {
		if err := a.save(); err != nil {
			return res, errors.New("importou, mas não consegui salvar no disco: " + err.Error())
		}
	}
	return res, nil
}

func parseCSV(raw []byte) ([]Entry, error) {
	text := toUTF8(raw)

	// Descobre o separador (o Excel em português salva com ponto e vírgula)
	first := text
	if i := strings.IndexAny(text, "\r\n"); i >= 0 {
		first = text[:i]
	}
	delim := ','
	best := strings.Count(first, ",")
	for _, d := range []rune{';', '\t'} {
		if c := strings.Count(first, string(d)); c > best {
			best = c
			delim = d
		}
	}

	r := csv.NewReader(strings.NewReader(text))
	r.Comma = delim
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV inválido: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}

	// Reconhece colunas do golock e de outros gerenciadores (Chrome, Bitwarden, 1Password...)
	head := rows[0]
	find := func(aliases ...string) int {
		for _, al := range aliases {
			for i, h := range head {
				if normHeader(h) == al {
					return i
				}
			}
		}
		return -1
	}
	iPass := find("senha", "password", "pass", "login_password")
	iName := find("nome", "name", "title", "titulo", "site", "login_uri", "url")
	iUser := find("usuario", "username", "user", "login", "login_username", "email", "e-mail")
	iCat := find("categoria", "category", "folder", "grouping", "pasta")
	if iPass < 0 || iName < 0 {
		return nil, errors.New("não encontrei as colunas 'nome' e 'senha' na primeira linha do CSV")
	}

	get := func(row []string, i int) string {
		if i >= 0 && i < len(row) {
			return row[i]
		}
		return ""
	}
	var list []Entry
	for _, row := range rows[1:] {
		list = append(list, Entry{
			Name:     get(row, iName),
			Username: get(row, iUser),
			Password: get(row, iPass),
			Category: get(row, iCat),
		})
	}
	return list, nil
}

func splitField(line string) (string, string, bool) {
	i := strings.Index(line, ":")
	if i < 0 {
		return "", "", false
	}
	var key string
	switch normHeader(line[:i]) {
	case "nome", "name":
		key = "nome"
	case "usuario", "user", "username", "login":
		key = "usuario"
	case "senha", "password":
		key = "senha"
	case "categoria", "category":
		key = "categoria"
	default:
		return "", "", false
	}
	return key, strings.TrimPrefix(line[i+1:], " "), true
}

func parseTXT(raw []byte) []Entry {
	var list []Entry
	var cur *Entry
	flush := func() {
		if cur != nil {
			list = append(list, *cur)
			cur = nil
		}
	}
	for _, line := range strings.Split(toUTF8(raw), "\n") {
		line = strings.TrimRight(line, "\r")
		key, val, ok := splitField(line)
		if !ok {
			continue
		}
		switch key {
		case "nome":
			flush()
			cur = &Entry{Name: val}
		case "usuario":
			if cur != nil {
				cur.Username = val
			}
		case "senha":
			if cur != nil {
				cur.Password = val
			}
		case "categoria":
			if cur != nil {
				cur.Category = val
			}
		}
	}
	flush()
	return list
}

var pdfDataRe = regexp.MustCompile(`GOLOCK-BEGIN:([A-Za-z0-9+/=]+):GOLOCK-END`)

func parsePDF(raw []byte) ([]Entry, error) {
	m := pdfDataRe.FindSubmatch(raw)
	if m == nil {
		return nil, errors.New("este PDF não foi gerado pelo golock. Para PDFs de outros programas, exporte em CSV por lá")
	}
	dec, err := base64.StdEncoding.DecodeString(string(m[1]))
	if err != nil {
		return nil, errors.New("os dados dentro do PDF estão corrompidos")
	}
	var p struct {
		V       int     `json:"v"`
		Entries []Entry `json:"entries"`
	}
	if err := json.Unmarshal(dec, &p); err != nil {
		return nil, errors.New("os dados dentro do PDF estão corrompidos")
	}
	return p.Entries, nil
}