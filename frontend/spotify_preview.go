package main

import (
	"fmt"
	"time"

	"github.com/k1y0miiii/applemusic-tui/engine"
)

// A static UI preview makes layout inspection possible without a Spotify login.
// It never starts an engine, opens a browser, fetches metadata, or plays audio.
func printSpotifyPreview() {
	applyTheme(themes[0])
	tracks := []engine.Track{
		{ID: "preview1", Title: "Your Spotify tracks appear here", Artist: "Lime TUI", Album: "Search with /", Duration: 210 * time.Second},
		{ID: "preview2", Title: "Queue songs, albums and playlists", Artist: "a adds to queue · A plays next", Duration: 185 * time.Second},
		{ID: "preview3", Title: "Your music. Your terminal.", Artist: "Lyrics · Library · Listening stats", Duration: 240 * time.Second},
	}
	m := model{w: 140, h: 40, phase: phaseReady, themeName: "lime", t: 1.7,
		st: engine.State{Authed: true, Now: tracks[0], Queue: tracks, Dur: tracks[0].Duration, Pos: 42 * time.Second, Volume: 70},
	}
	fmt.Println(m.View())
	fmt.Println("STATIC PREVIEW · Run without --preview to pair your Spotify Premium account.")
}

func printSpotifyHelp() {
	fmt.Print(`Lime TUI

Run Lime.exe to sign in through Spotify's pairing page.
A Spotify Premium account is required. No developer client ID is needed.

  /             Search songs, albums, playlists and artists
  Enter         Play or open selection
  Space         Play / pause
  n / p         Next / previous track
  a / A         Queue selection / play next while browsing
  L             Library: songs, albums, playlists, artists, liked songs
  Left / Right  Seek while the player is focused
  Up / Down     Volume while the player is focused
  s / r         Shuffle / repeat
  t / v         Theme / decorative visualizer
  m / T         Mini player / sleep timer
  ?             Context-sensitive keyboard help
  l             Full-screen lyrics (or click the lyric panel)
  [ / ]         Adjust lyric timing by 0.5 seconds; backslash resets
  R             Reconnect the audio engine
  q / Ctrl+C    Quit

  --preview     Print a static sample interface without logging in
  --version     Show build version
  --diagnose-system FILE  Write a local support report without account data

Profile: %LOCALAPPDATA%\LimeTUI (existing Pomme Spotify profiles are reused)
Override it with LIME_DATA_DIR. Keep this folder private.
`)
}
