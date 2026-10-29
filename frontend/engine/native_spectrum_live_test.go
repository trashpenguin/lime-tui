package engine

import (
	"math"
	"os"
	"testing"
	"time"
)

// This opt-in test uses an isolated queue and the user's existing signed-in account.
func TestNativeLiveSpectrumTracksOutput(t *testing.T) {
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
	id := os.Getenv("NATIVE_LIVE_TRACK_ID")
	if len(id) != 22 {
		t.Fatal("requires NATIVE_LIVE_TRACK_ID")
	}
	if err := eng.Play("song", id); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(60 * time.Second)
	var previous [32]float64
	active, changing := 0, 0
	for time.Now().Before(deadline) {
		bands, err := eng.Spectrum()
		if err != nil {
			t.Fatal(err)
		}
		peak := 0.0
		for _, value := range bands {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
				t.Fatal("invalid spectrum")
			}
			peak = math.Max(peak, value)
		}
		if peak > 0.05 {
			active++
			if bands != previous {
				changing++
			}
		}
		previous = bands
		if active >= 60 && changing >= 30 {
			break
		}
		time.Sleep(time.Second / 30)
	}
	if active < 60 || changing < 30 {
		t.Fatalf("no live audio response: active=%d changing=%d", active, changing)
	}
	if err := eng.Pause(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	for i := 0; i < 10; i++ {
		bands, err := eng.Spectrum()
		if err != nil || bands != [32]float64{} {
			t.Fatal("paused output should be silent", err)
		}
		time.Sleep(time.Second / 30)
	}
	if err := eng.SeekTo(30 * time.Second); err != nil {
		t.Fatal(err)
	}
	if err := eng.PlayPause(); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(15 * time.Second)
	resumed := false
	for time.Now().Before(deadline) {
		bands, err := eng.Spectrum()
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range bands {
			if value > .05 {
				resumed = true
			}
		}
		if resumed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !resumed {
		t.Fatal("spectrum did not resume after seek")
	}
	diagnostic, err := eng.PlaybackDiagnostics()
	if err != nil || len(diagnostic.Events) == 0 || len(diagnostic.Events) > 64 {
		t.Fatal("live playback trace unavailable or unbounded", err)
	}
	loaded, played := false, false
	for _, event := range diagnostic.Events {
		loaded = loaded || event.Event == "load_sent"
		played = played || event.Event == "playing"
	}
	if !loaded || !played {
		t.Fatal("diagnostic trace did not capture actual playback")
	}
	t.Logf("live frames=%d changing=%d; pause silence and seek/resume passed", active, changing)
}
