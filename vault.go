package main

// Persistência do cofre: arquivo único criptografado no computador do usuário.
//
// Esquema de chaves (versão 2, "envelope"):
//
//   - Os dados são cifrados com uma CHAVE DE DADOS aleatória de 256 bits (AES-256-GCM).
//   - Essa chave de dados é guardada no próprio arquivo, "embrulhada" de duas formas:
//       1. pela chave derivada da SENHA MESTRA (Argon2id + sal aleatório);
//       2. (opcional) pela chave derivada da CHAVE DE RECUPERAÇÃO, um segredo
//          aleatório de 160 bits que o usuário guarda offline.
//     Qualquer um dos dois caminhos abre o cofre.
//   - Os parâmetros do KDF, os sais e a versão são autenticados (AAD).
//   - Nem a senha mestra nem a chave de recuperação são gravadas.
//   - Gravação atômica (arquivo temporário + rename) e cópia .bak da versão anterior.
//
// Versão 1 (legada): a chave derivada da senha mestra cifrava os dados diretamente.
// Ela ainda pode ser aberta e é migrada para a versão 2 ao destravar.

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	vaultVersion  = 2
	legacyVersion = 1
	kdfName       = "argon2id"
	saltSize      = 16
	keySize       = 32 // AES-256

	// 160 bits de aleatoriedade: impraticável de adivinhar
	recoveryKeyBytes = 20

	defaultTime    uint32 = 3
	defaultMemory  uint32 = 64 * 1024 // KiB (64 MiB)
	defaultThreads uint8  = 4

	// Limites aceitos ao LER um arquivo (evita arquivo malicioso pedir recursos absurdos)
	maxKDFTime    uint32 = 10
	maxKDFMemory  uint32 = 512 * 1024 // KiB
	maxKDFThreads uint8  = 16
)

var (
	errWrongPassword  = errors.New("senha mestra incorreta")
	errWrongRecovery  = errors.New("chave de recuperação incorreta")
	errBadRecoveryFmt = errors.New("a chave de recuperação deve ter 32 letras e números (ex.: ABCD-EFGH-...)")
	errNoRecovery     = errors.New("este cofre não tem chave de recuperação")
	errCorruptVault   = errors.New("o arquivo do cofre está corrompido ou não é do golock")
	errNewerVault     = errors.New("este cofre foi criado por uma versão mais nova do golock")
)

// Cabeçalho do arquivo. Campos []byte viram base64 no JSON.
type vaultFile struct {
	Version int    `json:"v"`
	KDF     string `json:"kdf"`
	Time    uint32 `json:"t"`
	Memory  uint32 `json:"m"`
	Threads uint8  `json:"p"`
	Salt    []byte `json:"salt"` // sal da senha mestra

	// versão 2: chave de dados embrulhada
	KeyNonce []byte `json:"kn,omitempty"` // pela senha mestra
	KeyWrap  []byte `json:"kw,omitempty"`
	RecSalt  []byte `json:"rs,omitempty"` // pela chave de recuperação (opcional)
	RecNonce []byte `json:"rn,omitempty"`
	RecWrap  []byte `json:"rw,omitempty"`

	Nonce []byte `json:"nonce"`
	Data  []byte `json:"data"`
}

// Conteúdo que vai criptografado
type vaultPayload struct {
	NextID  int     `json:"next_id"`
	Entries []Entry `json:"entries"`
}

func (h vaultFile) hasRecovery() bool {
	return len(h.RecSalt) > 0 && len(h.RecNonce) > 0 && len(h.RecWrap) > 0
}

// Dados autenticados. "purpose" separa os usos (dados, embrulho da senha mestra,
// embrulho da recuperação) para um texto cifrado nunca servir no lugar de outro.
func (h vaultFile) aad(purpose string) []byte {
	base := fmt.Sprintf("golock|v%d|%s|%d|%d|%d|%x",
		h.Version, h.KDF, h.Time, h.Memory, h.Threads, h.Salt)
	if h.Version == legacyVersion {
		return []byte(base) // formato original, não pode mudar
	}
	if purpose == "recovery" {
		purpose += fmt.Sprintf("|%x", h.RecSalt)
	}
	return []byte(base + "|" + purpose)
}

func (h vaultFile) validate() error {
	if h.Version > vaultVersion {
		return errNewerVault
	}
	if (h.Version != vaultVersion && h.Version != legacyVersion) || h.KDF != kdfName {
		return errCorruptVault
	}
	if h.Time < 1 || h.Time > maxKDFTime ||
		h.Memory < 8 || h.Memory > maxKDFMemory ||
		h.Threads < 1 || h.Threads > maxKDFThreads ||
		len(h.Salt) < 8 || len(h.Salt) > 64 {
		return errCorruptVault
	}
	if h.Version == vaultVersion {
		if len(h.KeyNonce) == 0 || len(h.KeyWrap) == 0 || len(h.KeyWrap) > 64 {
			return errCorruptVault
		}
		if len(h.RecSalt)+len(h.RecNonce)+len(h.RecWrap) > 0 {
			if len(h.RecSalt) < 8 || len(h.RecSalt) > 64 ||
				len(h.RecNonce) == 0 || len(h.RecWrap) == 0 || len(h.RecWrap) > 64 {
				return errCorruptVault
			}
		}
	}
	return nil
}

// Pasta de dados do usuário:
//
//	Windows: %AppData%\golock\vault.golock
//	macOS:   ~/Library/Application Support/golock/vault.golock
//	Linux:   ~/.config/golock/vault.golock
func vaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "golock", "vault.golock"), nil
}

func deriveKey(secret string, salt []byte, t, m uint32, p uint8) []byte {
	return argon2.IDKey([]byte(secret), salt, t, m, p, keySize)
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// ---------- embrulho da chave de dados ----------

func wrapKey(kek, dek, aad []byte) (nonce, sealed []byte, err error) {
	gcm, err := newGCM(kek)
	if err != nil {
		return nil, nil, err
	}
	nonce, err = randomBytes(gcm.NonceSize())
	if err != nil {
		return nil, nil, err
	}
	return nonce, gcm.Seal(nil, nonce, dek, aad), nil
}

func unwrapKey(kek, nonce, sealed, aad []byte) ([]byte, error) {
	gcm, err := newGCM(kek)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, errCorruptVault
	}
	dek, err := gcm.Open(nil, nonce, sealed, aad)
	if err != nil {
		return nil, err
	}
	if len(dek) != keySize {
		zero(dek)
		return nil, errCorruptVault
	}
	return dek, nil
}

// Cria um cabeçalho novo (sal novo) e uma chave de dados aleatória, embrulhada
// pela senha mestra. Ainda sem chave de recuperação.
func newVaultHeader(master string) (vaultFile, []byte, error) {
	salt, err := randomBytes(saltSize)
	if err != nil {
		return vaultFile{}, nil, err
	}
	dek, err := randomBytes(keySize)
	if err != nil {
		return vaultFile{}, nil, err
	}
	h := vaultFile{
		Version: vaultVersion,
		KDF:     kdfName,
		Time:    defaultTime,
		Memory:  defaultMemory,
		Threads: defaultThreads,
		Salt:    salt,
	}
	kek := deriveKey(master, h.Salt, h.Time, h.Memory, h.Threads)
	defer zero(kek)
	h.KeyNonce, h.KeyWrap, err = wrapKey(kek, dek, h.aad("master"))
	if err != nil {
		zero(dek)
		return vaultFile{}, nil, err
	}
	return h, dek, nil
}

// Gera uma chave de recuperação nova (a anterior, se houver, deixa de valer) e
// devolve o texto para mostrar ao usuário, ex.: "ABCD-EFGH-...".
func setRecovery(h *vaultFile, dek []byte) (string, error) {
	raw, err := randomBytes(recoveryKeyBytes)
	if err != nil {
		return "", err
	}
	defer zero(raw)
	salt, err := randomBytes(saltSize)
	if err != nil {
		return "", err
	}
	next := *h
	next.RecSalt = salt
	kek := deriveKey(string(raw), salt, h.Time, h.Memory, h.Threads)
	defer zero(kek)
	next.RecNonce, next.RecWrap, err = wrapKey(kek, dek, next.aad("recovery"))
	if err != nil {
		return "", err
	}
	*h = next
	return formatRecoveryKey(raw), nil
}

// ---------- texto da chave de recuperação ----------

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// 20 bytes -> 32 caracteres em grupos de 4: ABCD-EFGH-IJKL-...
func formatRecoveryKey(raw []byte) string {
	s := b32.EncodeToString(raw)
	parts := make([]string, 0, len(s)/4)
	for i := 0; i < len(s); i += 4 {
		parts = append(parts, s[i:i+4])
	}
	return strings.Join(parts, "-")
}

// Aceita minúsculas, espaços e hífens a mais; 0 vira O e 1 vira I.
func parseRecoveryKey(s string) ([]byte, error) {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		switch r {
		case '-', ' ', '\t', '\r', '\n':
			continue
		case '0':
			r = 'O'
		case '1':
			r = 'I'
		}
		b.WriteRune(r)
	}
	raw, err := b32.DecodeString(b.String())
	if err != nil || len(raw) != recoveryKeyBytes {
		return nil, errBadRecoveryFmt
	}
	return raw, nil
}

// ---------- cifrar e abrir ----------

// Criptografa o conteúdo com um nonce novo a cada gravação.
func sealVault(h vaultFile, dek []byte, p vaultPayload) ([]byte, error) {
	if p.Entries == nil {
		p.Entries = []Entry{}
	}
	plain, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	defer zero(plain)

	gcm, err := newGCM(dek)
	if err != nil {
		return nil, err
	}
	nonce, err := randomBytes(gcm.NonceSize())
	if err != nil {
		return nil, err
	}
	h.Nonce = nonce
	h.Data = gcm.Seal(nil, nonce, plain, h.aad("data"))
	return json.Marshal(h)
}

func parseHeader(raw []byte) (vaultFile, error) {
	var h vaultFile
	if err := json.Unmarshal(raw, &h); err != nil {
		return h, errCorruptVault
	}
	return h, h.validate()
}

// Decifra o conteúdo com a chave de dados.
func openPayload(h vaultFile, dek []byte) (vaultPayload, error) {
	var p vaultPayload
	gcm, err := newGCM(dek)
	if err != nil {
		return p, errCorruptVault
	}
	if len(h.Nonce) != gcm.NonceSize() {
		return p, errCorruptVault
	}
	plain, err := gcm.Open(nil, h.Nonce, h.Data, h.aad("data"))
	if err != nil {
		return p, errCorruptVault
	}
	defer zero(plain)
	if err := json.Unmarshal(plain, &p); err != nil {
		return p, errCorruptVault
	}
	if p.Entries == nil {
		p.Entries = []Entry{}
	}
	return p, nil
}

// Abre o arquivo com a senha mestra. Devolve o cabeçalho, a chave de dados
// (para regravar sem pedir a senha de novo) e o conteúdo.
func openVault(raw []byte, master string) (vaultFile, []byte, vaultPayload, error) {
	var p vaultPayload
	h, err := parseHeader(raw)
	if err != nil {
		return h, nil, p, err
	}

	kek := deriveKey(master, h.Salt, h.Time, h.Memory, h.Threads)
	defer zero(kek)

	var dek []byte
	if h.Version == legacyVersion {
		// versão 1: a chave derivada é a própria chave de dados
		dek = append([]byte(nil), kek...)
	} else {
		dek, err = unwrapKey(kek, h.KeyNonce, h.KeyWrap, h.aad("master"))
		if err != nil {
			// senha errada ou arquivo adulterado: o GCM não distingue os dois
			if errors.Is(err, errCorruptVault) {
				return h, nil, p, err
			}
			return h, nil, p, errWrongPassword
		}
	}

	p, err = openPayload(h, dek)
	if err != nil {
		zero(dek)
		if h.Version == legacyVersion {
			return h, nil, p, errWrongPassword // v1 não separa senha errada de adulteração
		}
		return h, nil, p, err
	}
	return h, dek, p, nil
}

// Abre o arquivo com a chave de recuperação.
func openVaultWithRecovery(raw []byte, recovery string) (vaultFile, []byte, vaultPayload, error) {
	var p vaultPayload
	h, err := parseHeader(raw)
	if err != nil {
		return h, nil, p, err
	}
	if h.Version != vaultVersion || !h.hasRecovery() {
		return h, nil, p, errNoRecovery
	}
	rk, err := parseRecoveryKey(recovery)
	if err != nil {
		return h, nil, p, err
	}
	defer zero(rk)

	kek := deriveKey(string(rk), h.RecSalt, h.Time, h.Memory, h.Threads)
	defer zero(kek)
	dek, err := unwrapKey(kek, h.RecNonce, h.RecWrap, h.aad("recovery"))
	if err != nil {
		if errors.Is(err, errCorruptVault) {
			return h, nil, p, err
		}
		return h, nil, p, errWrongRecovery
	}
	p, err = openPayload(h, dek)
	if err != nil {
		zero(dek)
		return h, nil, p, err
	}
	return h, dek, p, nil
}

// Grava de forma atômica: escreve num temporário na mesma pasta, força o
// flush no disco e troca pelo arquivo final. Guarda a versão anterior em .bak.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if old, err := os.ReadFile(path); err == nil {
		_ = os.WriteFile(path+".bak", old, 0o600) // melhor esforço
	}

	tmp, err := os.CreateTemp(dir, "vault-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
}