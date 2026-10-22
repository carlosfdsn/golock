
package main

import (
	"encoding/json"
	"errors"
	"testing"
)

func testVault(t *testing.T) ([]byte, string, string) {
	t.Helper()

	master := "SenhaDeTeste-2026!"
	salt, err := randomBytes(saltSize)
	if err != nil {
		t.Fatal(err)
	}

	dek, err := randomBytes(keySize)
	if err != nil {
		t.Fatal(err)
	}

	h := vaultFile{
		Version: vaultVersion,
		KDF:     kdfName,
		Time:    1,
		Memory:  8,
		Threads: 1,
		Salt:    salt,
	}

	kek := deriveKey(master, salt, h.Time, h.Memory, h.Threads)
	defer zero(kek)
	defer zero(dek)

	h.KeyNonce, h.KeyWrap, err = wrapKey(
		kek, dek, h.aad("master"),
	)
	if err != nil {
		t.Fatal(err)
	}

	recovery, err := setRecovery(&h, dek)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := sealVault(h, dek, vaultPayload{
		NextID: 2,
		Entries: []Entry{
			{
				ID:       1,
				Name:     "Conta de teste",
				Username: "usuario-ficticio",
				Password: "senha-ficticia-123",
				Category: "outros",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	return raw, master, recovery
}

func TestVaultOpensWithCorrectPassword(t *testing.T) {
	raw, master, _ := testVault(t)

	_, key, payload, err := openVault(raw, master)
	if err != nil {
		t.Fatalf("não abriu com a senha correta: %v", err)
	}
	defer zero(key)

	if len(payload.Entries) != 1 {
		t.Fatalf("esperava 1 entrada; obtive %d", len(payload.Entries))
	}

	if payload.Entries[0].Password != "senha-ficticia-123" {
		t.Fatal("a senha de teste não foi preservada")
	}
}

func TestVaultRejectsWrongPassword(t *testing.T) {
	raw, _, _ := testVault(t)

	_, key, _, err := openVault(raw, "senha-incorreta")
	if key != nil {
		zero(key)
		t.Fatal("uma chave foi devolvida com senha incorreta")
	}
	if !errors.Is(err, errWrongPassword) {
		t.Fatalf("esperava errWrongPassword; obtive %v", err)
	}
}

func TestVaultRejectsTamperedData(t *testing.T) {
	raw, master, _ := testVault(t)

	var h vaultFile
	if err := json.Unmarshal(raw, &h); err != nil {
		t.Fatal(err)
	}
	if len(h.Data) == 0 {
		t.Fatal("o cofre de teste não tem dados cifrados")
	}

	h.Data[0] ^= 0x01

	tampered, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}

	_, key, _, err := openVault(tampered, master)
	if key != nil {
		zero(key)
		t.Fatal("uma chave foi devolvida para dados adulterados")
	}
	if !errors.Is(err, errCorruptVault) {
		t.Fatalf("esperava errCorruptVault; obtive %v", err)
	}
}

func TestVaultRecoveryKey(t *testing.T) {
	raw, _, recovery := testVault(t)

	_, key, payload, err := openVaultWithRecovery(raw, recovery)
	if err != nil {
		t.Fatalf("a chave de recuperação válida falhou: %v", err)
	}
	defer zero(key)

	if len(payload.Entries) != 1 {
		t.Fatalf("esperava 1 entrada; obtive %d", len(payload.Entries))
	}
}

func TestVaultRejectsInvalidRecoveryKey(t *testing.T) {
	raw, _, _ := testVault(t)

	_, key, _, err := openVaultWithRecovery(raw, "CHAVE-INVALIDA")
	if key != nil {
		zero(key)
		t.Fatal("uma chave foi devolvida para recuperação inválida")
	}
	if err == nil {
		t.Fatal("a chave de recuperação inválida foi aceita")
	}
}

func TestVaultRejectsMalformedFile(t *testing.T) {
	_, _, _, err := openVault([]byte("isso não é um cofre"), "qualquer-senha")
	if !errors.Is(err, errCorruptVault) {
		t.Fatalf("esperava errCorruptVault; obtive %v", err)
	}
}
