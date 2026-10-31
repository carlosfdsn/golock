package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// Senhas "difíceis": vírgula, ponto e vírgula, aspas, dois-pontos, acento,
// espaços nas pontas e emoji — tudo precisa sobreviver exportar -> importar.
func sampleApp() *App {
	a := NewApp()
	a.unlocked = true
	a.entries = []Entry{
		{ID: 1, Name: "Gmail, pessoal", Username: "eu@example.com", Password: `a,b;c"d'e:f`, Category: "email"},
		{ID: 2, Name: "Banco \"Azul\"", Username: "", Password: "  espaços  ", Category: "banco"},
		{ID: 3, Name: "Jogo", Username: "jogador", Password: "çãõ-é-ü-123-🔒", Category: "jogos"},
		{ID: 4, Name: "Outro site", Username: "u", Password: "Senha: com dois-pontos", Category: "outros"},
	}
	a.nextID = 5
	return a
}

// Compara o que foi lido de volta com o original (por nome, que é único aqui).
func checkRoundTrip(t *testing.T, a *App, got []Entry) {
	t.Helper()
	want := map[string]Entry{}
	for _, e := range a.entries {
		want[e.Name] = e
	}
	if len(got) != len(want) {
		t.Fatalf("esperava %d entradas; obtive %d: %+v", len(want), len(got), got)
	}
	for _, g := range got {
		w, ok := want[g.Name]
		if !ok {
			t.Fatalf("entrada inesperada: %+v", g)
		}
		if g.Username != w.Username || g.Password != w.Password {
			t.Errorf("%q: usuário/senha mudaram: %q/%q -> %q/%q", w.Name, w.Username, w.Password, g.Username, g.Password)
		}
		if cat := a.resolveCategory(g.Category, g.Name); cat != w.Category {
			t.Errorf("%q: categoria %q -> %q", w.Name, w.Category, cat)
		}
	}
}

func TestExportImportCSV(t *testing.T) {
	a := sampleApp()
	data, err := a.buildCSV()
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseCSV(data)
	if err != nil {
		t.Fatal(err)
	}
	checkRoundTrip(t, a, got)
}

func TestExportImportTXT(t *testing.T) {
	a := sampleApp()
	checkRoundTrip(t, a, parseTXT(a.buildTXT()))
}

func TestExportImportPDF(t *testing.T) {
	a := sampleApp()
	data, err := a.buildPDF()
	if err != nil {
		t.Fatal(err)
	}
	got, err := parsePDF(data)
	if err != nil {
		t.Fatal(err)
	}
	checkRoundTrip(t, a, got)
}

// O PDF tem de ser estruturalmente válido: tabela xref apontando para os
// objetos certos e uma página nova quando a lista não cabe em uma.
func TestPDFStructure(t *testing.T) {
	a := NewApp()
	a.unlocked = true
	for i := 1; i <= 80; i++ {
		a.entries = append(a.entries, Entry{
			ID: i, Name: fmt.Sprintf("Site %03d", i), Username: "usuario",
			Password: strings.Repeat("x", 90), Category: "outros",
		})
	}
	data, err := a.buildPDF()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-1.4")) || !bytes.HasSuffix(data, []byte("%%EOF\n")) {
		t.Fatal("cabeçalho ou final do PDF inválido")
	}

	// startxref -> posição da tabela
	i := bytes.LastIndex(data, []byte("startxref\n"))
	if i < 0 {
		t.Fatal("faltou startxref")
	}
	var xrefPos int
	if _, err := fmt.Sscanf(string(data[i+len("startxref\n"):]), "%d", &xrefPos); err != nil {
		t.Fatal(err)
	}
	table := string(data[xrefPos:])
	if !strings.HasPrefix(table, "xref\n0 ") {
		t.Fatalf("o startxref não aponta para a tabela xref: %q", table[:20])
	}
	lines := strings.Split(table, "\n")
	n, err := strconv.Atoi(strings.TrimPrefix(lines[1], "0 "))
	if err != nil || n < 9 {
		t.Fatalf("tabela xref estranha: %q", lines[1])
	}
	// lines[2] é o objeto 0; os demais apontam para "N 0 obj"
	for obj := 1; obj < n; obj++ {
		var off int
		if _, err := fmt.Sscanf(lines[2+obj], "%d", &off); err != nil {
			t.Fatal(err)
		}
		want := fmt.Sprintf("%d 0 obj\n", obj)
		if !bytes.HasPrefix(data[off:], []byte(want)) {
			t.Fatalf("xref do objeto %d aponta para o lugar errado (offset %d)", obj, off)
		}
	}

	// 80 itens não cabem em uma página
	count := strings.Count(string(data), "/Type /Page /Parent")
	if count < 2 {
		t.Fatalf("esperava várias páginas; obtive %d", count)
	}
	if !strings.Contains(string(data), fmt.Sprintf("/Count %d", count)) {
		t.Fatal("o /Count do PDF não bate com o número de páginas")
	}
}

// Entrada no tamanho máximo ainda cabe numa página.
func TestPDFMaxSizeEntryFitsOnePage(t *testing.T) {
	a := NewApp()
	a.unlocked = true
	a.entries = []Entry{{
		ID: 1, Name: strings.Repeat("n", maxNameLen), Username: strings.Repeat("u", maxUserLen),
		Password: strings.Repeat("p", maxPasswordLen), Category: "outros",
	}}
	data, err := a.buildPDF()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), "/Type /Page /Parent"); got != 1 {
		t.Fatalf("esperava 1 página; obtive %d", got)
	}
}

func TestParseCSVFormats(t *testing.T) {
	cases := []struct {
		name, csv        string
		wantName, wantU  string
		wantPass, wantCt string
	}{
		{"Chrome", "name,url,username,password,note\nGitHub,https://github.com,dev,pw123,\n",
			"GitHub", "dev", "pw123", ""},
		{"Bitwarden",
			"folder,favorite,type,name,notes,fields,reported_ids,login_uri,login_username,login_password,login_totp\nTrabalho,,login,GitHub,,,,https://github.com,dev,pw123,\n",
			"GitHub", "dev", "pw123", "Trabalho"},
		{"Excel pt-BR (;)", "nome;usuario;senha;categoria\nNetflix;eu;pw;Streaming\n",
			"Netflix", "eu", "pw", "Streaming"},
		{"com BOM", "\xEF\xBB\xBFnome,usuario,senha\nSite,u,pw\n", "Site", "u", "pw", ""},
	}
	for _, c := range cases {
		got, err := parseCSV([]byte(c.csv))
		if err != nil || len(got) != 1 {
			t.Fatalf("%s: %v %+v", c.name, err, got)
		}
		e := got[0]
		if e.Name != c.wantName || e.Username != c.wantU || e.Password != c.wantPass || e.Category != c.wantCt {
			t.Errorf("%s: obtive %+v", c.name, e)
		}
	}

	if _, err := parseCSV([]byte("coluna1,coluna2\na,b\n")); err == nil {
		t.Fatal("CSV sem as colunas nome/senha deveria dar erro")
	}
}

func TestToUTF8(t *testing.T) {
	cases := map[string][]byte{
		"UTF-8":        []byte("ação"),
		"UTF-8 BOM":    []byte("\xEF\xBB\xBFação"),
		"UTF-16 LE":    {0xFF, 0xFE, 'a', 0, 0xE7, 0, 0xE3, 0, 'o', 0},
		"UTF-16 BE":    {0xFE, 0xFF, 0, 'a', 0, 0xE7, 0, 0xE3, 0, 'o'},
		"Latin-1/ANSI": {'a', 0xE7, 0xE3, 'o'},
	}
	for name, raw := range cases {
		if got := toUTF8(raw); got != "ação" {
			t.Errorf("%s: obtive %q", name, got)
		}
	}
}

func TestWrapText(t *testing.T) {
	if got := wrapText("", 4); len(got) != 1 || got[0] != "" {
		t.Fatalf("texto vazio: %q", got)
	}
	got := wrapText("abcdefghij", 4)
	if strings.Join(got, "|") != "abcd|efgh|ij" {
		t.Fatalf("quebra errada: %q", got)
	}
	// não corta no meio de um caractere de vários bytes
	got = wrapText("ççççç", 2)
	if strings.Join(got, "|") != "çç|çç|ç" {
		t.Fatalf("quebra errada com acentos: %q", got)
	}
}

func TestPDFTextEscaping(t *testing.T) {
	if got := string(pdfEscape([]byte(`a(b)\`))); got != `a\(b\)\\` {
		t.Fatalf("pdfEscape: %q", got)
	}
	if got := winAnsi("é€"); !bytes.Equal(got, []byte{0xE9, 0x80}) {
		t.Fatalf("winAnsi: % x", got)
	}
	if got := string(winAnsi("🔒")); got != "?" {
		t.Fatalf("caractere fora do WinAnsi deveria virar '?': %q", got)
	}
}

func TestEntryKeyIgnoresCaseButNotPassword(t *testing.T) {
	a := Entry{Name: "Gmail", Username: "Eu", Password: "Abc"}
	b := Entry{Name: "gmail", Username: "eu", Password: "Abc"}
	c := Entry{Name: "gmail", Username: "eu", Password: "abc"}
	if entryKey(a) != entryKey(b) {
		t.Fatal("nome e usuário deveriam ignorar maiúsculas")
	}
	if entryKey(a) == entryKey(c) {
		t.Fatal("a senha deveria diferenciar maiúsculas")
	}
}
