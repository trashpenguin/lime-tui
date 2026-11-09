package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLimeTerminalFragmentUsesPackagedIcon(t *testing.T) {
	local := t.TempDir()
	app := filepath.Join(t.TempDir(), "Lime test")
	icon := filepath.Join(app, "assets", "lime.png")
	if err := os.MkdirAll(filepath.Dir(icon), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(icon, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(app, "Lime.exe")
	if err := ensureLimeTerminalProfile(local, exe); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(local, "Microsoft", "Windows Terminal", "Fragments", "LimeTUI", "lime.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fragment struct{ Profiles []map[string]string }
	if err := json.Unmarshal(data, &fragment); err != nil {
		t.Fatal(err)
	}
	if len(fragment.Profiles) != 1 || fragment.Profiles[0]["icon"] != icon || fragment.Profiles[0]["guid"] != limeTerminalProfileID || fragment.Profiles[0]["name"] != "Lime" {
		t.Fatal("profile did not identify the packaged Lime icon")
	}
}
