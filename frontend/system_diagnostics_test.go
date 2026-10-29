package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func TestSystemDiagnosticsDoesNotReadOrExposePersonalData(t *testing.T) {
	previousLimit := debug.SetMemoryLimit(-1)
	defer debug.SetMemoryLimit(previousLimit)
	root := t.TempDir()
	for _, name := range []string{"LIME_DATA_DIR", "AMTUI_CONFIG_DIR", "AMTUI_NATIVE_BACKEND", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"} {
		t.Setenv(name, os.Getenv(name))
	}
	t.Setenv("LIME_DATA_DIR", root)
	cache := filepath.Join(root, "cache", "spotatui", "streaming_cache")
	os.MkdirAll(cache, 0700)
	os.WriteFile(filepath.Join(cache, "credentials.dpapi"), []byte("private fixture"), 0600)
	report := systemDiagnostics()
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private fixture") || strings.Contains(string(data), root) {
		t.Fatal("personal data exposed in support report")
	}
	if report["spotify_credential_storage"] != "windows-user-dpapi" || report["network_accessed"] != false {
		t.Fatal("incorrect storage report")
	}
}
