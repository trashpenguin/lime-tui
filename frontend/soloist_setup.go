package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

func soloistConfigPath() string { return filepath.Join(configDir(), "soloist-settings.json") }

func loadSoloistSettings() {
	config, err := readSoloistSettings()
	engine.ConfigureSoloist(config, err)
}

func readSoloistSettings() (engine.SoloistConfig, error) {
	var config engine.SoloistConfig
	value, err := readProfileSecret(soloistConfigPath())
	if os.IsNotExist(err) {
		return config, nil
	}
	if err != nil {
		return config, errors.New("Soloist settings could not be decrypted. Run Setup Soloist.cmd to repair or disable them")
	}
	if json.Unmarshal([]byte(value), &config) != nil || config.Path == "" || config.Key == "" {
		return engine.SoloistConfig{}, errors.New("Soloist settings are invalid. Run Setup Soloist.cmd")
	}
	return config, nil
}

func runSoloistSetup() error {
	if runtime.GOOS != "windows" {
		return errors.New("Soloist setup is currently supported in the Windows release")
	}
	setupSpotifyProfile()
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Lime: official Spotify Soloist setup")
	fmt.Println("Download and extract the Windows x64 build from:")
	fmt.Println("https://developer.spotify.com/documentation/soloist/reference/downloads-and-updates")
	fmt.Println("Get your own API key: https://developer.spotify.com/dashboard/soloist")
	fmt.Println("The key is stored using Windows encryption. Soloist requires it on its command line while running.")
	fmt.Println("Soloist builds expire after 90 days; replace the executable with a current official build when needed.")
	fmt.Print("Full path to soloist.exe (or type native to switch back): ")
	line, err := reader.ReadString('\n')
	if err != nil {
		return errors.New("Setup cancelled before a path was entered")
	}
	path := strings.Trim(strings.TrimSpace(line), "\"")
	if strings.EqualFold(path, "native") {
		for _, name := range []string{soloistConfigPath(), soloistConfigPath() + ".dpapi"} {
			if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
				return errors.New("Could not clear Soloist settings")
			}
		}
		loadSoloistSettings()
		fmt.Println("Native playback selected. Existing Spotify pairing is retained.")
		return nil
	}
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Base(path), "soloist.exe") {
		return errors.New("Choose the extracted soloist.exe using its full path")
	}
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		return errors.New("The selected soloist.exe was not found")
	}
	fmt.Print("Soloist API key (hidden): ")
	secret, err := term.ReadPassword(os.Stdin.Fd())
	fmt.Println()
	if err != nil {
		return errors.New("Could not read the key privately; run setup in Windows Terminal")
	}
	defer clear(secret)
	key := strings.TrimSpace(string(secret))
	if !strings.HasPrefix(key, "spak_") || len(key) > 4096 || strings.ContainsAny(key, "\r\n\t ") {
		return errors.New("Enter a valid Soloist API key")
	}
	config := engine.SoloistConfig{Path: path, Key: key}
	data, err := json.Marshal(config)
	if err != nil {
		return errors.New("Could not prepare Soloist settings")
	}
	defer clear(data)
	if err := writeProfileSecret(soloistConfigPath(), string(data)); err != nil {
		return errors.New("Could not securely save Soloist settings")
	}
	loadSoloistSettings()
	fmt.Println("Saved. Start Lime, open Spotify on a phone or computer on the same network, and select Lime Soloist in its device menu once.")
	fmt.Println("Lime's library, queue, lyrics and transport controls will use Soloist. Its visualizer has no audio tap in this preview.")
	return nil
}

type soloistSetupMsg struct{ err error }

func setupSoloistCmd() tea.Cmd {
	path, err := os.Executable()
	if err != nil {
		return func() tea.Msg { return soloistSetupMsg{errors.New("Could not open setup")} }
	}
	return tea.ExecProcess(exec.Command(path, "--setup-soloist"), func(err error) tea.Msg { return soloistSetupMsg{err} })
}
