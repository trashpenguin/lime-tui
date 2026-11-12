package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

const playbackReportName = "Lime-playback-report.json"

type playbackReportMsg struct {
	err            error
	traceAvailable bool
}

func playbackIssueCode(messages ...string) string {
	for _, message := range messages {
		switch {
		case strings.HasPrefix(message, "Waiting for Spotify Connect."):
			return "soloist_waiting_for_connect"
		case strings.HasPrefix(message, "Soloist build expired."):
			return "soloist_build_expired"
		case strings.HasPrefix(message, "Soloist rejected playback."):
			return "soloist_command_rejected"
		}
	}
	return ""
}

func playbackSupportCmd(eng *engine.Engine, issue string) tea.Cmd {
	return func() tea.Msg {
		report := systemDiagnostics()
		if issue != "" {
			report["playback_issue"] = issue
		}
		available := false
		if eng != nil {
			if trace, err := eng.PlaybackDiagnostics(); err == nil {
				report["playback"] = trace
				available = true
			}
		}
		report["playback_trace_available"] = available
		exe, err := os.Executable()
		if err == nil {
			err = writePlaybackSupport(filepath.Join(filepath.Dir(exe), playbackReportName), report)
		}
		return playbackReportMsg{err: err, traceAvailable: available}
	}
}

func writePlaybackSupport(path string, report map[string]any) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
