package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const limeTerminalProfileID = "{619c4fa4-cf11-4994-8e5e-35d92ea4a7d9}"

func ensureLimeTerminalProfile(localData, exe string) error {
	if !filepath.IsAbs(localData) || !filepath.IsAbs(exe) || strings.ContainsAny(exe, "\";\r\n") {
		return fmt.Errorf("invalid Lime terminal profile path")
	}
	icon := filepath.Join(filepath.Dir(exe), "assets", "lime.png")
	if info, err := os.Stat(icon); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("Lime terminal icon is unavailable")
	}
	folder := filepath.Join(localData, "Microsoft", "Windows Terminal", "Fragments", "LimeTUI")
	if err := os.MkdirAll(folder, 0700); err != nil {
		return err
	}
	profile := map[string]string{
		"guid": limeTerminalProfileID, "name": "Lime", "icon": icon,
		"commandline":       `"` + exe + `" --terminal-window`,
		"startingDirectory": filepath.Dir(exe), "closeOnExit": "graceful",
	}
	data, err := json.MarshalIndent(map[string][]map[string]string{"profiles": {profile}}, "", "  ")
	if err != nil {
		return err
	}
	target := filepath.Join(folder, "lime.json")
	if existing, err := os.ReadFile(target); err == nil && bytes.Equal(existing, data) {
		return nil
	}
	file, err := os.CreateTemp(folder, ".lime-profile-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), target)
}
