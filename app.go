package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"math/big"
	"os"
	"strings"
	"time"

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

type App struct {
	ctx        context.Context
	masterPass string // TEMPORÁRIO: só na memória
	unlocked   bool
	entries    []Entry
	nextID     int

	header vaultFile // cabeçalho (sal e parâmetros) do cofre aberto
	key    []byte    // chave derivada, só na memória enquanto o cofre está aberto
}

func NewApp() *App { return &App{nextID: 1} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

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
	for _, r := range rules {
		for _, w := range r.words {
			if strings.Contains(n, w) {
				return r.key
			}
		}
	}
	return "outros"
}

func validCategory(k string) bool {
	for _, c := range categories {
		if c.Key == k {
			return true
		}
	}
	return false
}

// ---------- Senha mestra ----------

// Existe cofre salvo no computador?
func (a *App) HasVault() bool {
	path, err := vaultPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Cria o cofre e devolve a CHAVE DE RECUPERAÇÃO (mostrada uma única vez).
// Quem perder a senha mestra e essa chave perde os dados: não há outro caminho.
func (a *App) CreateVault(master string) (string, error) {
	if len(master) < 8 {
		return "", errors.New("a senha mestra precisa ter pelo menos 8 caracteres")
	}
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

	a.masterPass = master
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
	h, key, p, err := openVault(raw, master)
	if err != nil {
		return err
	}

	a.masterPass = master
	a.header = h
	a.key = key
	a.setPayload(p)
	a.unlocked = true

	// Cofre da versão 1: migra para o formato com chave de dados + recuperação.
	// A cópia .bak guarda o arquivo antigo; se a migração falhar, segue funcionando.
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
	if !a.unlocked {
		return "", errors.New("cofre travado")
	}
	if subtle.ConstantTimeCompare([]byte(master), []byte(a.masterPass)) != 1 {
		return "", errWrongPassword
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
	if len(newMaster) < 8 {
		return "", errors.New("a nova senha mestra precisa ter pelo menos 8 caracteres")
	}
	path, err := vaultPath()
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", errors.New("não consegui ler o arquivo do cofre")
	}
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
	a.masterPass = newMaster
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

func (a *App) resetState() {
	zero(a.key)
	a.key = nil
	a.masterPass = ""
	a.entries = nil
	a.nextID = 1
	a.unlocked = false
}

func (a *App) Lock() { a.unlocked = false }

// ---------- Senhas ----------

// category vazio = detectar automaticamente
func (a *App) AddEntry(name, user, pass, category string) error {
	if !a.unlocked {
		return errors.New("cofre travado")
	}
	name = strings.TrimSpace(name)
	if name == "" || pass == "" {
		return errors.New("nome e senha são obrigatórios")
	}
	if !validCategory(category) {
		category = a.DetectCategory(name)
	}
	a.entries = append(a.entries, Entry{a.nextID, name, strings.TrimSpace(user), pass, category})
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
	if !a.unlocked {
		return errors.New("cofre travado")
	}
	name = strings.TrimSpace(name)
	if name == "" || pass == "" {
		return errors.New("nome e senha são obrigatórios")
	}
	if !validCategory(category) {
		category = a.DetectCategory(name)
	}
	for i, e := range a.entries {
		if e.ID != id {
			continue
		}
		old := e
		a.entries[i] = Entry{id, name, strings.TrimSpace(user), pass, category}
		if err := a.save(); err != nil {
			a.entries[i] = old // não gravou: mantém os dados antigos
			return errors.New("não consegui salvar no disco: " + err.Error())
		}
		return nil
	}
	return errors.New("senha não encontrada")
}

func (a *App) DeleteEntry(id int) {
	if !a.unlocked {
		return
	}
	for i, e := range a.entries {
		if e.ID == id {
			old := append([]Entry(nil), a.entries...)
			a.entries = append(a.entries[:i], a.entries[i+1:]...)
			if err := a.save(); err != nil {
				a.entries = old // não gravou: mantém a senha
			}
			return
		}
	}
}

func (a *App) GeneratePassword(n int) string {
	if n < 8 {
		n = 8
	}
	if n > 64 {
		n = 64
	}
	const chars = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!@#$%&*?"
	out := make([]byte, n)
	for i := range out {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		out[i] = chars[idx.Int64()]
	}
	return string(out)
}

// Copia e limpa a área de transferência depois de 15 segundos
func (a *App) CopyToClipboard(text string) {
	runtime.ClipboardSetText(a.ctx, text)
	time.AfterFunc(15*time.Second, func() {
		runtime.ClipboardSetText(a.ctx, "")
	})
}