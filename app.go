package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Category struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type Entry struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
	Category string `json:"category"`
}

// Limites dos campos. Evitam entradas gigantes (e, de quebra, garantem que
// qualquer senha cabe numa página do PDF exportado).
const (
	minMasterLen   = 8
	maxNameLen     = 200
	maxUserLen     = 200
	maxPasswordLen = 1024

	clipboardTTL = 15 * time.Second
)

type App struct {
	ctx context.Context

	// O Wails chama os métodos em goroutines diferentes (um clique duplo, o
	// bloqueio automático e uma importação podem acontecer ao mesmo tempo).
	// O mutex protege TODO o estado abaixo. Funções internas (save, resetState,
	// checkMaster...) assumem que ele já está preso.
	mu       sync.Mutex
	unlocked bool
	entries  []Entry
	nextID   int

	header vaultFile // cabeçalho (sal e parâmetros) do cofre aberto
	key    []byte    // chave de dados, só na memória enquanto o cofre está aberto

	// A senha mestra NÃO fica guardada: quando o app precisa conferi-la
	// (exportar, gerar chave de recuperação) ela é verificada contra o cofre.

	clipMu    sync.Mutex
	clipTimer *time.Timer
	clipGen   int
}

func NewApp() *App { return &App{nextID: 1} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// Ao fechar o app, apaga a chave da memória.
func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.resetState()
}

// ---------- Categorias ----------

var categories = []Category{
	{"email", "E-mails"},
	{"banco", "Bancos e finanças"},
	{"social", "Redes sociais"},
	{"streaming", "Streaming"},
	{"jogos", "Jogos"},
	{"trabalho", "Trabalho"},
	{"compras", "Compras"},
	{"outros", "Outros"},
}

// Palavras que o app reconhece no nome para descobrir a categoria
var rules = []struct {
	key   string
	words []string
}{
	{"email", []string{"gmail", "outlook", "hotmail", "yahoo", "proton", "icloud", "e-mail", "email", "mail"}},
	{"banco", []string{"banco", "bank", "nubank", "itau", "bradesco", "santander", "caixa", "c6", "picpay", "paypal", "mercado pago", "pix", "cartao", "cartão", "corretora", "xp", "binance"}},
	{"social", []string{"instagram", "facebook", "twitter", "tiktok", "linkedin", "whatsapp", "telegram", "discord", "reddit", "pinterest", "snapchat", "threads"}},
	{"streaming", []string{"netflix", "spotify", "disney", "prime", "hbo", "max", "youtube", "globoplay", "deezer", "twitch", "crunchyroll"}},
	{"jogos", []string{"steam", "epic", "xbox", "playstation", "psn", "riot", "valorant", "nintendo", "roblox", "minecraft", "battle.net"}},
	{"trabalho", []string{"slack", "github", "gitlab", "notion", "trello", "teams", "zoom", "jira", "figma", "google drive", "empresa", "trabalho"}},
	{"compras", []string{"amazon", "mercado livre", "mercadolivre", "shopee", "aliexpress", "magalu", "americanas", "shein", "ifood", "loja"}},
}

func (a *App) Categories() []Category { return categories }

func (a *App) DetectCategory(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return "outros"
	}
	tokens := strings.FieldsFunc(n, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, r := range rules {
		for _, w := range r.words {
			if matchWord(n, tokens, w) {
				return r.key
			}
		}
	}
	return "outros"
}

// Palavras curtas (até 3 letras: "xp", "c6", "max", "pix") só valem como palavra
// inteira. Antes, qualquer nome que as contivesse caía na categoria errada
// ("Expedia" virava banco, "Maxi Atacado" virava streaming).
func matchWord(name string, tokens []string, w string) bool {
	if utf8.RuneCountInString(w) > 3 {
		return strings.Contains(name, w)
	}
	for _, t := range tokens {
		if t == w {
			return true
		}
	}
	return false
}

func validCategory(k string) bool {
	for _, c := range categories {
		if c.Key == k {
			return true
		}
	}
	return false
}

// ---------- Validação das senhas guardadas ----------

// Limpa e valida os campos de uma senha.
func cleanEntry(name, user, pass string) (string, string, string, error) {
	name = strings.TrimSpace(name)
	user = strings.TrimSpace(user)
	if name == "" || pass == "" {
		return "", "", "", errors.New("nome e senha são obrigatórios")
	}
	e := Entry{Name: name, Username: user, Password: pass}
	if !entryFits(e) {
		return "", "", "", fmt.Errorf("limite de tamanho excedido (nome e usuário: %d caracteres; senha: %d)",
			maxNameLen, maxPasswordLen)
	}
	return name, user, pass, nil
}

func entryFits(e Entry) bool {
	return utf8.RuneCountInString(e.Name) <= maxNameLen &&
		utf8.RuneCountInString(e.Username) <= maxUserLen &&
		utf8.RuneCountInString(e.Password) <= maxPasswordLen
}

// ---------- Senha mestra ----------

// Existe cofre salvo no computador?
func (a *App) HasVault() bool {
	path, err := vaultPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	// Só "não existe" significa "sem cofre". Qualquer outro erro (permissão,
	// disco com problema...) NÃO pode levar a criar um cofre novo por cima do
	// arquivo que está lá.
	return err == nil || !errors.Is(err, fs.ErrNotExist)
}

// Cria o cofre e devolve a CHAVE DE RECUPERAÇÃO (mostrada uma única vez).
// Quem perder a senha mestra e essa chave perde os dados: não há outro caminho.
func (a *App) CreateVault(master string) (string, error) {
	if utf8.RuneCountInString(master) < minMasterLen {
		return "", fmt.Errorf("a senha mestra precisa ter pelo menos %d caracteres", minMasterLen)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.HasVault() {
		// nunca sobrescreve um cofre que já existe
		return "", errors.New("já existe um cofre neste computador")
	}

	h, dek, err := newVaultHeader(master)
	if err != nil {
		return "", err
	}
	recovery, err := setRecovery(&h, dek)
	if err != nil {
		zero(dek)
		return "", err
	}

	a.resetState()
	a.header = h
	a.key = dek
	a.entries = []Entry{}
	a.nextID = 1
	if err := a.save(); err != nil {
		a.resetState()
		return "", err
	}
	a.unlocked = true
	return recovery, nil
}

func (a *App) Unlock(master string) error {
	path, err := vaultPath()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return errors.New("não consegui ler o arquivo do cofre")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	h, key, p, err := openVault(raw, master)
	if err != nil {
		return err
	}

	a.resetState() // descarta qualquer estado anterior (ex.: dois desbloqueios seguidos)
	a.header = h
	a.key = key
	a.setPayload(p)
	a.unlocked = true

	// Cofre da versão 1: migra para o formato com chave de dados.
	// A cópia .bak guarda o arquivo antigo; se a migração falhar, segue funcionando.
	// (A chave de recuperação é criada depois, pelo menu do app.)
	if h.Version == legacyVersion {
		if nh, dek, err := newVaultHeader(master); err == nil {
			oldH, oldKey := a.header, a.key
			a.header, a.key = nh, dek
			if err := a.save(); err != nil {
				zero(dek)
				a.header, a.key = oldH, oldKey
			} else {
				zero(oldKey)
			}
		}
	}
	return nil
}

func (a *App) setPayload(p vaultPayload) {
	a.entries = p.Entries
	a.nextID = p.NextID
	if a.nextID < 1 {
		a.nextID = 1
	}
	for _, e := range a.entries {
		if e.ID >= a.nextID {
			a.nextID = e.ID + 1
		}
	}
}

// Confere a senha mestra contra o cofre aberto, sem precisar guardá-la na memória.
func (a *App) checkMaster(master string) error {
	h := a.header
	kek := deriveKey(master, h.Salt, h.Time, h.Memory, h.Threads)
	defer zero(kek)

	if h.Version == legacyVersion {
		// versão 1: a chave derivada é a própria chave de dados
		if subtle.ConstantTimeCompare(kek, a.key) != 1 {
			return errWrongPassword
		}
		return nil
	}
	dek, err := unwrapKey(kek, h.KeyNonce, h.KeyWrap, h.aad("master"))
	if err != nil {
		return errWrongPassword
	}
	zero(dek)
	return nil
}

// O cofre tem chave de recuperação? (lê só o cabeçalho; funciona travado)
func (a *App) HasRecoveryKey() bool {
	path, err := vaultPath()
	if err != nil {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	h, err := parseHeader(raw)
	return err == nil && h.Version == vaultVersion && h.hasRecovery()
}

// Gera uma chave de recuperação nova (a antiga deixa de valer). Exige a senha mestra.
func (a *App) GenerateRecoveryKey(master string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.unlocked {
		return "", errors.New("cofre travado")
	}
	if a.header.Version != vaultVersion {
		return "", errors.New("este cofre ainda está no formato antigo: trave e destrave de novo para atualizá-lo")
	}
	if err := a.checkMaster(master); err != nil {
		return "", err
	}
	old := a.header
	recovery, err := setRecovery(&a.header, a.key)
	if err != nil {
		a.header = old
		return "", err
	}
	if err := a.save(); err != nil {
		a.header = old
		return "", errors.New("não consegui salvar no disco: " + err.Error())
	}
	return recovery, nil
}

// "Esqueci a senha mestra": abre o cofre com a chave de recuperação, define uma
// senha mestra nova e devolve uma chave de recuperação nova (a usada deixa de valer).
// Os dados são preservados.
func (a *App) RecoverVault(recoveryKey, newMaster string) (string, error) {
	if utf8.RuneCountInString(newMaster) < minMasterLen {
		return "", fmt.Errorf("a nova senha mestra precisa ter pelo menos %d caracteres", minMasterLen)
	}
	path, err := vaultPath()
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", errors.New("não consegui ler o arquivo do cofre")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	_, oldKey, p, err := openVaultWithRecovery(raw, recoveryKey)
	if err != nil {
		return "", err
	}
	zero(oldKey)

	// chave de dados nova: a senha antiga e a chave usada deixam de servir para o cofre novo
	h, dek, err := newVaultHeader(newMaster)
	if err != nil {
		return "", err
	}
	recovery, err := setRecovery(&h, dek)
	if err != nil {
		zero(dek)
		return "", err
	}

	a.resetState()
	a.header = h
	a.key = dek
	a.setPayload(p)
	if err := a.save(); err != nil {
		a.resetState()
		return "", errors.New("não consegui salvar no disco: " + err.Error())
	}
	a.unlocked = true
	return recovery, nil
}

// Criptografa e grava o cofre no disco. Chamado a cada alteração.
// O mutex precisa estar preso.
func (a *App) save() error {
	path, err := vaultPath()
	if err != nil {
		return err
	}
	raw, err := sealVault(a.header, a.key, vaultPayload{NextID: a.nextID, Entries: a.entries})
	if err != nil {
		return err
	}
	return writeFileAtomic(path, raw)
}

// Apaga da memória a chave e os dados. O mutex precisa estar preso.
func (a *App) resetState() {
	zero(a.key)
	a.key = nil
	a.header = vaultFile{}
	a.entries = nil
	a.nextID = 1
	a.unlocked = false
}

// Trava o cofre de verdade: além de esconder a lista, tira a chave e as senhas
// da memória (antes, só uma flag era desligada e a chave continuava lá).
func (a *App) Lock() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.resetState()
}

func (a *App) isUnlocked() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.unlocked
}

// ---------- Senhas ----------

// category vazio = detectar automaticamente
func (a *App) AddEntry(name, user, pass, category string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.unlocked {
		return errors.New("cofre travado")
	}
	name, user, pass, err := cleanEntry(name, user, pass)
	if err != nil {
		return err
	}
	if !validCategory(category) {
		category = a.DetectCategory(name)
	}
	a.entries = append(a.entries, Entry{ID: a.nextID, Name: name, Username: user, Password: pass, Category: category})
	a.nextID++
	if err := a.save(); err != nil {
		// não conseguiu gravar: desfaz para a tela não mostrar algo que não foi salvo
		a.entries = a.entries[:len(a.entries)-1]
		a.nextID--
		return errors.New("não consegui salvar no disco: " + err.Error())
	}
	return nil
}

func (a *App) ListEntries() []Entry {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.unlocked {
		return []Entry{}
	}
	out := make([]Entry, len(a.entries))
	copy(out, a.entries)
	return out
}

// Edita uma senha que já existe. category vazio = detectar automaticamente.
// Se não conseguir gravar no disco, desfaz a alteração.
func (a *App) UpdateEntry(id int, name, user, pass, category string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.unlocked {
		return errors.New("cofre travado")
	}
	name, user, pass, err := cleanEntry(name, user, pass)
	if err != nil {
		return err
	}
	if !validCategory(category) {
		category = a.DetectCategory(name)
	}
	for i, e := range a.entries {
		if e.ID != id {
			continue
		}
		old := e
		a.entries[i] = Entry{ID: id, Name: name, Username: user, Password: pass, Category: category}
		if err := a.save(); err != nil {
			a.entries[i] = old // não gravou: mantém os dados antigos
			return errors.New("não consegui salvar no disco: " + err.Error())
		}
		return nil
	}
	return errors.New("senha não encontrada")
}

// Apaga uma senha. Se não conseguir gravar no disco, mantém a senha e avisa
// (antes o erro era engolido e a tela dizia "Senha apagada" mesmo sem apagar).
func (a *App) DeleteEntry(id int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.unlocked {
		return errors.New("cofre travado")
	}
	for i, e := range a.entries {
		if e.ID != id {
			continue
		}
		old := append([]Entry(nil), a.entries...)
		a.entries = append(a.entries[:i], a.entries[i+1:]...)
		if err := a.save(); err != nil {
			a.entries = old // não gravou: mantém a senha
			return errors.New("não consegui salvar no disco: " + err.Error())
		}
		return nil
	}
	return errors.New("senha não encontrada")
}

// Sem 'l', 'I', 'O', '0' e '1': evita confusão ao ler a senha.
const (
	lowerChars  = "abcdefghijkmnopqrstuvwxyz"
	upperChars  = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	digitChars  = "23456789"
	symbolChars = "!@#$%&*?"
)

func randIndex(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		// Nunca devolver uma senha previsível: sem aleatoriedade, melhor parar.
		panic("crypto/rand indisponível: " + err.Error())
	}
	return int(v.Int64())
}

// Gera uma senha com pelo menos uma minúscula, uma maiúscula, um número e um
// símbolo (muitos sites exigem), com as posições embaralhadas.
func (a *App) GeneratePassword(n int) string {
	if n < 8 {
		n = 8
	}
	if n > 64 {
		n = 64
	}
	groups := []string{lowerChars, upperChars, digitChars, symbolChars}
	all := strings.Join(groups, "")

	out := make([]byte, 0, n)
	for _, g := range groups {
		out = append(out, g[randIndex(len(g))])
	}
	for len(out) < n {
		out = append(out, all[randIndex(len(all))])
	}
	// embaralha (Fisher-Yates) para os caracteres garantidos não ficarem sempre no começo
	for i := len(out) - 1; i > 0; i-- {
		j := randIndex(i + 1)
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

// Copia e limpa a área de transferência depois de 15 segundos — mas só se ela
// ainda tiver o que o golock copiou (não apaga algo que a pessoa copiou depois,
// e uma cópia nova não é apagada pelo temporizador da anterior).
func (a *App) CopyToClipboard(text string) {
	a.clipMu.Lock()
	defer a.clipMu.Unlock()

	if a.clipTimer != nil {
		a.clipTimer.Stop()
	}
	a.clipGen++
	gen := a.clipGen

	if err := runtime.ClipboardSetText(a.ctx, text); err != nil {
		return
	}
	a.clipTimer = time.AfterFunc(clipboardTTL, func() {
		a.clipMu.Lock()
		defer a.clipMu.Unlock()
		if gen != a.clipGen {
			return // houve uma cópia mais nova
		}
		if cur, err := runtime.ClipboardGetText(a.ctx); err == nil && cur == text {
			_ = runtime.ClipboardSetText(a.ctx, "")
		}
		a.clipTimer = nil
	})
}
