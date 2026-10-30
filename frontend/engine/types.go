package engine

import "time"

type Track struct {
	ID, Title, Artist, Album string
	Kind                     string
	Duration                 time.Duration
	Art                      string
}
type State struct {
	PlaybackEngine                string
	Authed, Playing, Initializing bool
	Pos, Dur                      time.Duration
	Volume                        int
	Shuffle                       bool
	Repeat                        int
	Now                           Track
	Queue                         []Track
	QueuePos                      int
	Err                           string
}
type SearchResults struct{ Songs, Albums, Playlists, Recent, Artists []Track }
type HistoryPage struct {
	Tracks     []Track
	RetryAfter time.Duration
}
type PlaybackTraceEvent struct {
	AtMs           uint64  `json:"atMs"`
	Event          string  `json:"event"`
	Request        *uint64 `json:"request"`
	MatchesCurrent *bool   `json:"matchesCurrent"`
	PositionMs     uint64  `json:"positionMs"`
	FailureStage   string  `json:"failureStage,omitempty"`
	FailureKind    *int    `json:"failureKind,omitempty"`
}
type PlaybackDiagnostics struct {
	PlaybackEngine string               `json:"playbackEngine,omitempty"`
	Playing        bool                 `json:"playing"`
	Loading        bool                 `json:"loading"`
	HasError       bool                 `json:"hasError"`
	Connected      bool                 `json:"connected"`
	Recovering     bool                 `json:"recovering"`
	PositionMs     uint64               `json:"positionMs"`
	Events         []PlaybackTraceEvent `json:"events"`
}
type DiscoverySection struct {
	Title string  `json:"title"`
	Items []Track `json:"items"`
}
type Discovery struct {
	Sections []DiscoverySection `json:"sections"`
}
type Account struct {
	Name       string `json:"name"`
	Handle     string `json:"handle"`
	Storefront string `json:"storefront"`
	Country    string `json:"country"`
}

type TimedLyrics struct {
	Synced bool `json:"synced"`
	Lines  []struct {
		AtMs int64  `json:"atMs"`
		Text string `json:"text"`
	} `json:"lines"`
}
type wireTrack struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Kind   string `json:"kind"`
	DurMs  uint64 `json:"durMs"`
	Art    string `json:"art"`
}

func (t wireTrack) track() Track {
	kind := t.Kind
	switch kind {
	case "songs":
		kind = "song"
	case "albums":
		kind = "album"
	case "playlists":
		kind = "playlist"
	case "artists":
		kind = "artist"
	}
	return Track{ID: t.ID, Title: t.Title, Artist: t.Artist, Album: t.Album, Kind: kind, Duration: time.Duration(t.DurMs) * time.Millisecond, Art: t.Art}
}
func tracks(items []wireTrack) []Track {
	out := make([]Track, len(items))
	for i, t := range items {
		out[i] = t.track()
	}
	return out
}
