package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k1y0miiii/applemusic-tui/engine"
)

func TestSoloistSettingsAreEncryptedAndClearedOnSignOut(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AMTUI_CONFIG_DIR", dir)
	config := engine.SoloistConfig{Path: filepath.Join(dir, "soloist.exe"), Key: "spak_private-fixture"}
	data, _ := json.Marshal(config)
	if err := writeProfileSecret(soloistConfigPath(), string(data)); err != nil {
		t.Fatal(err)
	}
	protected, err := os.ReadFile(soloistConfigPath() + ".dpapi")
	if err != nil || strings.Contains(string(protected), config.Key) {
		t.Fatal("key was not encrypted")
	}
	if _, err := os.Stat(soloistConfigPath()); !os.IsNotExist(err) {
		t.Fatal("plaintext settings remained")
	}
	got, err := readSoloistSettings()
	if err != nil || got != config {
		t.Fatal("settings did not round-trip")
	}
	m := model{}
	if err := m.clearSpotifyPersonalData(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(soloistConfigPath() + ".dpapi"); !os.IsNotExist(err) {
		t.Fatal("sign-out retained the key")
	}
}

func TestSoloistSettingsDoNotSilentlyFallBackAfterTampering(t *testing.T) {
	t.Setenv("AMTUI_CONFIG_DIR", t.TempDir())
	if err := os.WriteFile(soloistConfigPath()+".dpapi", []byte("invalid fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSoloistSettings(); err == nil {
		t.Fatal("invalid encrypted settings were ignored")
	}
}
