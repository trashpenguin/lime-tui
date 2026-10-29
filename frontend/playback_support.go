package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

const playbackReportName = "Lime-playback-report.json"

type playbackReportMsg struct {
	err            error
	traceAvailable bool
}

func playbackSupportCmd(eng *engine.Engine) tea.Cmd {
	return func() tea.Msg {
		report := systemDiagnostics()
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
