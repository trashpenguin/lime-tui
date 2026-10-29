package main

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1y0miiii/applemusic-tui/engine"
)

type libraryActionMsg struct {
	token         uint64
	action, title string
	err           error
}

func (m model) libraryButtons() []string {
	if m.libraryErr != "" {
		return []string{"[r Retry]"}
	}
	list := m.curList()
	if m.sBusy || m.sSel < 0 || m.sSel >= len(list) {
		return nil
	}
	switch list[m.sSel].Kind {
	case "artist", "artists":
		return []string{"[Enter Open artist]"}
	case "album", "albums", "playlist", "playlists":
		return []string{"[Enter Play All]", "[a Add to Queue]", "[A Next]"}
	default:
		return []string{"[Enter Play]", "[a Add to Queue]", "[A Next]"}
	}
}

func (m model) startLibraryAction(track engine.Track, action string) (tea.Model, tea.Cmd) {
	if m.libraryActionBusy {
		return m, nil
	}
	m.libraryActionToken++
	token, eng := m.libraryActionToken, m.eng
	m.libraryActionBusy, m.libraryActionFailed = true, false
	verb := "Adding to queue: "
	if action == "next" {
		verb = "Adding next: "
	}
	if action == "play" {
		verb = "Loading playback: "
		m.loading, m.loadSnap, m.loadStart = track.Title, snap(m.st), m.t
		m.loadHideQueue = true
		m.libraryPlayRequested = true
		m.beginAudioInitialization()
	}
	m.libraryActionStatus = verb + track.Title
	return m, func() tea.Msg {
		var err error
		if eng == nil {
			err = errors.New("Spotify connection unavailable")
		} else {
			switch action {
			case "play":
				err = eng.Play(track.Kind, track.ID)
			case "next":
				err = eng.QueueNext(track.Kind, track.ID)
			default:
				err = eng.QueueLater(track.Kind, track.ID)
			}
		}
		return libraryActionMsg{token: token, action: action, title: track.Title, err: err}
	}
}

func (m model) finishLibraryAction(msg libraryActionMsg) (tea.Model, tea.Cmd) {
	if msg.token != m.libraryActionToken {
		return m, nil
	}
	m.libraryActionBusy = false
	m.libraryActionFailed = msg.err != nil
	if msg.err != nil {
		verb := "Could not add to queue: "
		if msg.action == "play" {
			verb = "Could not play: "
			m.libraryPlayRequested = false
			m.loading = ""
			m.failAudioInitialization()
		}
		m.libraryActionStatus = verb + strings.Join(strings.Fields(msg.err.Error()), " ") + " · Retry the action"
	} else {
		verb := "Added to queue: "
		if msg.action == "next" {
			verb = "Added next: "
		}
		if msg.action == "play" {
			verb = "Starting playback: "
		}
		m.libraryActionStatus = verb + msg.title
	}
	m.note, m.noteAt = m.libraryActionStatus, m.t
	return m, nil
}
