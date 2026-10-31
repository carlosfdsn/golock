package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Isola o cofre numa pasta temporária (o app grava em os.UserConfigDir).
func isolateConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir) // Linux
	t.Setenv("APPDATA", dir)         // Windows
	t.Setenv("HOME", dir)            // macOS
}

const testMaster = "senha-mestra-teste"

func TestAppLifecycle(t *testing.T) {
	isolateConfig(t)
	a := NewApp()

	if a.HasVault() {
		t.Fatal("não deveria haver cofre numa pasta nova")
	}
	recovery, err := a.CreateVault(testMaster)
	if err != nil {
		t.Fatal(err)
	}
	if recovery == "" {
		t.Fatal("o cofre novo deveria devolver a chave de recuperação")
	}
	if !a.HasVault() || !a.HasRecoveryKey() {
		t.Fatal("o cofre criado deveria existir e ter chave de recuperação")
	}
	if _, err := a.CreateVault("outra-senha-123"); err == nil {
		t.Fatal("criar de novo não pode sobrescrever o cofre existente")
	}

	// adicionar, categoria automática e validações
	if err := a.AddEntry("Gmail", "eu@example.com", "s3nha-forte!", ""); err != nil {
		t.Fatal(err)
	}
	list := a.ListEntries()
	if len(list) != 1 || list[0].Category != "email" {
		t.Fatalf("esperava 1 entrada na categoria email; obtive %+v", list)
	}
	id := list[0].ID
	if err := a.AddEntry("", "x", "y", ""); err == nil {
		t.Fatal("nome vazio deveria ser recusado")
	}
	if err := a.AddEntry(strings.Repeat("n", maxNameLen+1), "", "y", ""); err == nil {
		t.Fatal("nome grande demais deveria ser recusado")
	}
	if err := a.AddEntry("Site", "", strings.Repeat("p", maxPasswordLen+1), ""); err == nil {
		t.Fatal("senha grande demais deveria ser recusada")
	}
	if len(a.ListEntries()) != 1 {
		t.Fatal("entradas recusadas não podem ser guardadas")
	}

	if err := a.UpdateEntry(id, "Gmail pessoal", "eu@example.com", "nova-senha-123", "email"); err != nil {
		t.Fatal(err)
	}
	if err := a.UpdateEntry(9999, "x", "", "y", ""); err == nil {
		t.Fatal("editar uma senha que não existe deveria dar erro")
	}

	// confere a senha mestra sem guardá-la
	if err := a.checkMaster(testMaster); err != nil {
		t.Fatalf("a senha mestra correta foi recusada: %v", err)
	}
	if err := a.checkMaster("senha-errada!!"); !errors.Is(err, errWrongPassword) {
		t.Fatalf("esperava errWrongPassword; obtive %v", err)
	}

	// travar de verdade: a chave sai da memória
	a.Lock()
	if a.key != nil {
		t.Fatal("a chave de dados continua na memória depois de travar")
	}
	if len(a.ListEntries()) != 0 {
		t.Fatal("travado não pode listar senhas")
	}
	if err := a.AddEntry("Algo", "", "x", ""); err == nil {
		t.Fatal("travado não pode adicionar senhas")
	}
	if err := a.DeleteEntry(id); err == nil {
		t.Fatal("travado não pode apagar senhas")
	}

	// destravar
	if err := a.Unlock("senha-errada!!"); !errors.Is(err, errWrongPassword) {
		t.Fatalf("esperava errWrongPassword; obtive %v", err)
	}
	if err := a.Unlock(testMaster); err != nil {
		t.Fatal(err)
	}
	list = a.ListEntries()
	if len(list) != 1 || list[0].Name != "Gmail pessoal" || list[0].Password != "nova-senha-123" {
		t.Fatalf("os dados não foram preservados: %+v", list)
	}

	// apagar persiste
	if err := a.DeleteEntry(id); err != nil {
		t.Fatal(err)
	}
	if err := a.DeleteEntry(id); err == nil {
		t.Fatal("apagar duas vezes deveria dar erro")
	}
	a.Lock()
	if err := a.Unlock(testMaster); err != nil {
		t.Fatal(err)
	}
	if len(a.ListEntries()) != 0 {
		t.Fatal("a senha apagada voltou depois de destravar")
	}
}

func TestAppRecoverVault(t *testing.T) {
	isolateConfig(t)
	a := NewApp()

	recovery1, err := a.CreateVault(testMaster)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.AddEntry("Banco", "eu", "segredo-123", ""); err != nil {
		t.Fatal(err)
	}
	a.Lock()

	if _, err := a.RecoverVault(recovery1, "curta"); err == nil {
		t.Fatal("senha nova curta demais deveria ser recusada")
	}

	// a chave aceita minúsculas e espaços no lugar dos hífens
	typed := strings.ToLower(strings.ReplaceAll(recovery1, "-", " "))
	recovery2, err := a.RecoverVault(typed, "nova-mestra-123")
	if err != nil {
		t.Fatal(err)
	}
	if recovery2 == recovery1 {
		t.Fatal("a chave de recuperação deveria mudar depois de usada")
	}
	if list := a.ListEntries(); len(list) != 1 || list[0].Password != "segredo-123" {
		t.Fatalf("os dados não foram preservados: %+v", list)
	}

	a.Lock()
	if err := a.Unlock(testMaster); !errors.Is(err, errWrongPassword) {
		t.Fatalf("a senha antiga não deveria mais abrir; obtive %v", err)
	}
	if err := a.Unlock("nova-mestra-123"); err != nil {
		t.Fatal(err)
	}
	a.Lock()

	if _, err := a.RecoverVault(recovery1, "terceira-senha-1"); err == nil {
		t.Fatal("a chave de recuperação antiga deveria ter deixado de valer")
	}
	if _, err := a.RecoverVault(recovery2, "terceira-senha-1"); err != nil {
		t.Fatalf("a chave nova deveria valer: %v", err)
	}
}

func TestAppGenerateRecoveryKey(t *testing.T) {
	isolateConfig(t)
	a := NewApp()

	oldKey, err := a.CreateVault(testMaster)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.GenerateRecoveryKey("senha-errada!!"); !errors.Is(err, errWrongPassword) {
		t.Fatalf("esperava errWrongPassword; obtive %v", err)
	}
	newKey, err := a.GenerateRecoveryKey(testMaster)
	if err != nil {
		t.Fatal(err)
	}
	a.Lock()
	if _, err := a.RecoverVault(oldKey, "outra-mestra-123"); err == nil {
		t.Fatal("a chave anterior deveria ter deixado de valer")
	}
	if _, err := a.RecoverVault(newKey, "outra-mestra-123"); err != nil {
		t.Fatalf("a chave nova deveria valer: %v", err)
	}
}

// Cofre da versão 1 (formato legado): abre e é migrado para a versão 2.
func TestLegacyVaultMigration(t *testing.T) {
	isolateConfig(t)

	salt, err := randomBytes(saltSize)
	if err != nil {
		t.Fatal(err)
	}
	h := vaultFile{Version: legacyVersion, KDF: kdfName, Time: 1, Memory: 8, Threads: 1, Salt: salt}
	kek := deriveKey(testMaster, h.Salt, h.Time, h.Memory, h.Threads)
	defer zero(kek)
	raw, err := sealVault(h, kek, vaultPayload{
		NextID:  2,
		Entries: []Entry{{ID: 1, Name: "Antigo", Username: "u", Password: "p-antiga-1", Category: "outros"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	path, err := vaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	a := NewApp()
	if a.HasRecoveryKey() {
		t.Fatal("cofre v1 não tem chave de recuperação")
	}
	if err := a.Unlock(testMaster); err != nil {
		t.Fatal(err)
	}
	if list := a.ListEntries(); len(list) != 1 || list[0].Password != "p-antiga-1" {
		t.Fatalf("os dados do cofre v1 não foram preservados: %+v", list)
	}

	// o arquivo foi regravado na versão 2 e o antigo ficou no .bak
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	nh, err := parseHeader(disk)
	if err != nil || nh.Version != vaultVersion {
		t.Fatalf("esperava o arquivo migrado para a versão %d; obtive %+v (%v)", vaultVersion, nh.Version, err)
	}
	bak, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("faltou a cópia .bak do cofre antigo: %v", err)
	}
	if bh, err := parseHeader(bak); err != nil || bh.Version != legacyVersion {
		t.Fatalf("o .bak deveria guardar o cofre v1; obtive %+v (%v)", bh.Version, err)
	}

	// depois de migrado, a chave de recuperação já pode ser criada e o cofre reabre
	if _, err := a.GenerateRecoveryKey(testMaster); err != nil {
		t.Fatal(err)
	}
	a.Lock()
	if err := a.Unlock(testMaster); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratePassword(t *testing.T) {
	a := NewApp()
	all := lowerChars + upperChars + digitChars + symbolChars

	for _, n := range []int{0, 8, 12, 20, 64, 200} {
		want := n
		if want < 8 {
			want = 8
		}
		if want > 64 {
			want = 64
		}
		for i := 0; i < 100; i++ {
			p := a.GeneratePassword(n)
			if len(p) != want {
				t.Fatalf("GeneratePassword(%d): tamanho %d, esperava %d", n, len(p), want)
			}
			for _, c := range p {
				if !strings.ContainsRune(all, c) {
					t.Fatalf("caractere fora do conjunto permitido: %q", c)
				}
			}
			for name, set := range map[string]string{
				"minúscula": lowerChars, "maiúscula": upperChars, "número": digitChars, "símbolo": symbolChars,
			} {
				if !strings.ContainsAny(p, set) {
					t.Fatalf("a senha %q não tem %s", p, name)
				}
			}
		}
	}
	if a.GeneratePassword(20) == a.GeneratePassword(20) {
		t.Fatal("duas senhas geradas seguidas não deveriam ser iguais")
	}
}

func TestDetectCategory(t *testing.T) {
	a := NewApp()
	cases := map[string]string{
		"Gmail":            "email",
		"Conta Nubank":     "banco",
		"XP Investimentos": "banco",
		"C6 Bank":          "banco",
		"Instagram":        "social",
		"HBO Max":          "streaming",
		"Steam":            "jogos",
		"GitHub":           "trabalho",
		"Mercado Livre":    "compras",
		"":                 "outros",
		"Qualquer coisa":   "outros",
		// palavras curtas só valem como palavra inteira
		"Expedia":      "outros",
		"Maxi Atacado": "outros",
		"Pixabay":      "outros",
	}
	for name, want := range cases {
		if got := a.DetectCategory(name); got != want {
			t.Errorf("DetectCategory(%q) = %q; esperava %q", name, got, want)
		}
	}
}

func TestRecoveryKeyText(t *testing.T) {
	raw, err := randomBytes(recoveryKeyBytes)
	if err != nil {
		t.Fatal(err)
	}
	text := formatRecoveryKey(raw)
	if len(strings.Split(text, "-")) != 8 {
		t.Fatalf("esperava 8 grupos de 4 letras; obtive %q", text)
	}
	for _, variant := range []string{
		text,
		strings.ToLower(text),
		strings.ReplaceAll(text, "-", " "),
		strings.ReplaceAll(text, "-", ""),
		"  " + text + "\n",
	} {
		got, err := parseRecoveryKey(variant)
		if err != nil || string(got) != string(raw) {
			t.Fatalf("parseRecoveryKey(%q) falhou: %v", variant, err)
		}
	}
	for _, bad := range []string{"", "ABCD", text + "AAAA", "!!!!-????"} {
		if _, err := parseRecoveryKey(bad); !errors.Is(err, errBadRecoveryFmt) {
			t.Fatalf("parseRecoveryKey(%q): esperava errBadRecoveryFmt; obtive %v", bad, err)
		}
	}
}
