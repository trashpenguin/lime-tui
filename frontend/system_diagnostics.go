package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// This support report reads file metadata, never credentials, catalog or history.
func systemDiagnostics() map[string]any {
	setupSpotifyProfile()
	root := filepath.Dir(os.Getenv("AMTUI_CONFIG_DIR"))
	exists := func(path string) bool { info, err := os.Stat(path); return err == nil && info.Mode().IsRegular() }
	cache := filepath.Join(root, "cache", "spotatui", "streaming_cache")
	storage := "none"
	if exists(filepath.Join(cache, "credentials.dpapi")) {
		storage = "windows-user-dpapi"
	}
	legacy := exists(filepath.Join(cache, "credentials.json"))
	if legacy {
		if storage == "none" {
			storage = "legacy-plaintext-awaiting-migration"
		} else {
			storage = "protected-and-legacy-copy-present"
		}
	}
	binaries := map[string]any{}
	if exe, err := os.Executable(); err == nil {
		for _, name := range []string{"Lime.exe", "lime-engine.exe"} {
			file, err := os.Open(filepath.Join(filepath.Dir(exe), name))
			if err != nil {
				binaries[name] = map[string]any{"present": false}
				continue
			}
			hash := sha256.New()
			size, err := io.Copy(hash, file)
			file.Close()
			if err != nil {
				binaries[name] = map[string]any{"readable": false}
				continue
			}
			binaries[name] = map[string]any{"present": true, "bytes": size, "sha256": hex.EncodeToString(hash.Sum(nil))}
		}
	}
	return map[string]any{"version": version, "go": runtime.Version(), "os": runtime.GOOS, "architecture": runtime.GOARCH, "windows_terminal": os.Getenv("WT_SESSION") != "", "spotify_credential_storage": storage, "legacy_plaintext_present": legacy, "optional_credential_protected": exists(filepath.Join(root, "settings", "lastfm.dpapi")), "binaries": binaries, "network_accessed": false}
}

func writeSystemDiagnostics(path string) error {
	data, err := json.MarshalIndent(systemDiagnostics(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
