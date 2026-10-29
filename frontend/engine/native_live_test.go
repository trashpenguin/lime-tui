package engine

import (
	"context"
	"github.com/k1y0miiii/applemusic-tui/lyrics"
	"os"
	"testing"
	"time"
)

// Explicit opt-in exercises the user's signed-in account; ordinary test runs stay offline.
func TestNativeLiveCatalogPlaybackAndQueue(t *testing.T) {
	if os.Getenv("NATIVE_LIVE_TEST") != "1" {
		t.Skip("requires native backend and signed-in subscription")
	}
	eng, err := Connect(func(status string) { t.Log(status) })
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	defer eng.Pause()
	if err := eng.SetVolume(15); err != nil {
		t.Fatal(err)
	}
	res, err := eng.Search("Kingslayer Bring Me The Horizon")
	if err != nil {
		t.Log("partial search:", err)
	}
	if len(res.Songs) == 0 {
		t.Fatal("no songs returned")
	}
	t.Logf("Search: %d songs, %d albums, %d playlists", len(res.Songs), len(res.Albums), len(res.Playlists))
	id := os.Getenv("NATIVE_LIVE_TRACK_ID")
	if len(id) != 22 {
		t.Fatal("Set NATIVE_LIVE_TRACK_ID to a playable Spotify track ID")
	}
	if err := eng.Play("song", id); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(60 * time.Second)
	var state State
	for time.Now().Before(deadline) {
		state, err = eng.State()
		if err != nil {
			t.Fatal(err)
		}
		if state.Err != "" {
			t.Fatal(state.Err)
		}
		if state.Playing {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if !state.Playing || state.Now.ID != id {
		t.Fatalf("did not start: %+v", state)
	}
	t.Logf("Playing: %s; duration=%s; artwork=%t", state.Now.Title, state.Dur, state.Now.Art != "")
	timed, lyricErr := eng.Lyrics(state.Now.ID)
	if lyricErr == nil && len(timed.Lines) > 0 {
		t.Logf("Spotify lyrics: %d lines; synchronized=%t", len(timed.Lines), timed.Synced)
	} else {
		fallback, fallbackErr := lyrics.Fetch(context.Background(), state.Now.Artist, state.Now.Title, state.Dur)
		if fallbackErr != nil {
			t.Log("Lyrics service unavailable; playback remains usable")
		} else {
			t.Logf("Fallback lyrics: %d lines", len(fallback.Lines))
		}
	}
	if err := eng.PlayPause(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	state, err = eng.State()
	if err != nil || state.Playing {
		t.Fatal("pause failed", err)
	}
	if err := eng.SetVolume(15); err != nil {
		t.Fatal(err)
	}
	if err := eng.SeekTo(12 * time.Second); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	state, err = eng.State()
	if err != nil || state.Volume != 15 {
		t.Fatal("volume failed", err)
	}
	if err := eng.Reload(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(800 * time.Millisecond)
	state, err = eng.State()
	if err != nil || state.Playing || state.Volume != 15 {
		t.Fatal("paused recovery failed", err)
	}
	if err := eng.QueueLater("song", id); err != nil {
		t.Fatal(err)
	}
	nextID := id
	for _, item := range res.Songs {
		if item.ID != id {
			nextID = item.ID
			break
		}
	}
	if err := eng.QueueNext("song", nextID); err != nil {
		t.Fatal(err)
	}
	state, err = eng.State()
	if err != nil || len(state.Queue) != 3 {
		t.Fatal("queue insertion failed", err)
	}
	if err := eng.JumpTo(0); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		state, err = eng.State()
		if err != nil {
			t.Fatal(err)
		}
		if state.Playing {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !state.Playing {
		t.Fatal("first queue song did not play")
	}
	if err := eng.SeekTo(state.Dur - 300*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		state, err = eng.State()
		if err != nil {
			t.Fatal(err)
		}
		if state.QueuePos == 1 && state.Playing {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if state.QueuePos != 1 || !state.Playing || state.Now.ID != nextID {
		t.Fatal("automatic queue advance failed")
	}
	t.Log("End-of-track advanced automatically to the next queue song")
	if err := eng.JumpTo(2); err != nil {
		t.Fatal(err)
	}
	if err := eng.ToggleShuffle(); err != nil {
		t.Fatal(err)
	}
	if err := eng.CycleRepeat(); err != nil {
		t.Fatal(err)
	}
	state, err = eng.State()
	if err != nil || !state.Shuffle || state.Repeat != 1 || state.QueuePos != 2 {
		t.Fatal("queue controls failed", err)
	}
	if err := eng.SetVolume(15); err != nil {
		t.Fatal(err)
	}
	library, err := eng.Library()
	if err != nil {
		t.Log("partial library:", err)
	}
	t.Logf("Library: %d albums, %d playlists, %d recent", len(library.Albums), len(library.Playlists), len(library.Recent))
	if len(res.Albums) > 0 {
		if err := eng.Play("album", res.Albums[0].ID); err != nil {
			t.Fatal("album resolve:", err)
		}
		state, _ = eng.State()
		if len(state.Queue) == 0 {
			t.Fatal("album queue empty")
		}
		t.Logf("Album queue: %d songs", len(state.Queue))
	}
	if len(res.Playlists) > 0 {
		if err := eng.Play("playlist", res.Playlists[0].ID); err != nil {
			t.Fatal("playlist resolve:", err)
		}
		state, _ = eng.State()
		if len(state.Queue) == 0 {
			t.Fatal("playlist queue empty")
		}
		t.Logf("Playlist queue: %d songs", len(state.Queue))
	}
	_, err = eng.Account()
	if err != nil {
		t.Fatal("account:", err)
	}
}

func TestNativeLiveRepeatAndQueueEnd(t *testing.T) {
	if os.Getenv("NATIVE_LIVE_TEST") != "1" {
		t.Skip("requires signed-in backend and explicit audio opt-in")
	}
	id := os.Getenv("NATIVE_LIVE_TRACK_ID")
	if len(id) != 22 {
		t.Fatal("Set a playable Spotify fixture ID")
	}
	eng, err := Connect(func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	defer eng.Pause()
	if err = eng.SetVolume(15); err != nil {
		t.Fatal(err)
	}
	if err = eng.Play("song", id); err != nil {
		t.Fatal(err)
	}
	wait := func(predicate func(State) bool) State {
		t.Helper()
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			state, e := eng.State()
			if e != nil {
				t.Fatal(e)
			}
			if state.Err != "" {
				t.Fatal(state.Err)
			}
			if predicate(state) {
				return state
			}
			time.Sleep(150 * time.Millisecond)
		}
		t.Fatal("Timed out waiting for native playback transition")
		return State{}
	}
	state := wait(func(s State) bool { return s.Playing && !s.Initializing && s.Pos > time.Second && s.Dur > time.Second })
	for state.Repeat != 2 {
		if err = eng.CycleRepeat(); err != nil {
			t.Fatal(err)
		}
		state, err = eng.State()
		if err != nil {
			t.Fatal(err)
		}
	}
	if err = eng.SeekTo(state.Dur - 500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	wait(func(s State) bool { return s.Now.ID == id && s.Playing && !s.Initializing && s.Pos < 5*time.Second })
	t.Log("Repeat-track restarted the same track after its real end event")
	state, err = eng.State()
	if err != nil {
		t.Fatal(err)
	}
	for state.Repeat != 0 {
		if err = eng.CycleRepeat(); err != nil {
			t.Fatal(err)
		}
		state, err = eng.State()
		if err != nil {
			t.Fatal(err)
		}
	}
	if err = eng.SeekTo(state.Dur - 500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	wait(func(s State) bool { return !s.Playing && !s.Initializing && s.Pos > s.Dur-3*time.Second })
	t.Log("Repeat-off stopped at the end of the single-track queue")
}
