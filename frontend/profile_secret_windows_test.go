package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOptionalCredentialProtectionAndMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lastfm")
	secret := "synthetic-optional-account-secret"
	if err := os.WriteFile(path, []byte(secret), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := readProfileSecret(path); err != nil || got != secret {
		t.Fatal("migration failed", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("plaintext credential remains")
	}
	data, err := os.ReadFile(path + ".dpapi")
	if err != nil || strings.Contains(string(data), secret) {
		t.Fatal("credential not protected", err)
	}
	if err := writeProfileSecret(path, "updated-fixture"); err != nil {
		t.Fatal(err)
	}
	if got, err := readProfileSecret(path); err != nil || got != "updated-fixture" {
		t.Fatal("replacement failed", err)
	}
	data[len(data)-1] ^= 0x40
	os.WriteFile(path+".dpapi", data, 0600)
	os.WriteFile(path, []byte(secret), 0600)
	if _, err := readProfileSecret(path); err == nil {
		t.Fatal("corrupted credential fell back to plaintext")
	}
}
