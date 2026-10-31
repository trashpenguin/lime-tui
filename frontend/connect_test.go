package main

import (
	"testing"
	"time"

	"github.com/k1y0miiii/applemusic-tui/engine"
)

func TestPhoneTrackWaitsForMetadataBeforeFetchingLyrics(t *testing.T) {
	m := model{phase: phaseReady, lyFor: "previous"}
	pending := engine.State{
		Authed: true, Initializing: true,
		Now: engine.Track{ID: "phone-track", Title: "Loading Spotify track…"},
	}
	next, _ := m.Update(stateMsg{st: pending})
	m = next.(model)
	if m.lyFor != "previous" {
		t.Fatal("lyrics requested before Connect metadata arrived")
	}
	ready := pending
	ready.Initializing, ready.Playing = false, true
	ready.Now.Title, ready.Now.Artist = "Phone song", "Phone artist"
	ready.Now.Art = "https://i.scdn.co/image/test-cover"
	ready.Pos, ready.Dur = 17*time.Second, 180*time.Second
	ready.Now.Duration = ready.Dur
	next, _ = m.Update(stateMsg{st: ready})
	m = next.(model)
	if m.lyFor != ready.Now.ID || !m.lyBusy {
		t.Fatal("confirmed Connect track did not schedule fresh lyrics")
	}
	if m.st.Now.Title != "Phone song" || m.st.Now.Art != ready.Now.Art || m.st.Pos != ready.Pos {
		t.Fatal("Connect song metadata or position was not applied")
	}
}
