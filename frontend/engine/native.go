package engine

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Engine struct {
	rpc    *rpcClient
	cmd    *exec.Cmd
	waited chan struct{}
	once   sync.Once
}

func backendPath() (string, error) {
	if path := os.Getenv("AMTUI_NATIVE_BACKEND"); path != "" {
		return filepath.Abs(path)
	}
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), "lime-engine.exe"), nil
}
func start(path string) (*Engine, error) {
	cmd := nativeCommand(path)
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	writer, err := cmd.StdinPipe()
	if err != nil {
		reader.Close()
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		reader.Close()
		writer.Close()
		return nil, err
	}
	engine := &Engine{cmd: cmd, rpc: newRPC(reader, writer), waited: make(chan struct{})}
	go func() { _ = cmd.Wait(); engine.rpc.close(); close(engine.waited) }()
	return engine, nil
}

// Connect restores native auth and opens the temporary login helper only if needed.
func Connect(status func(string)) (*Engine, error) {
	return ConnectContext(context.Background(), status)
}
func ConnectContext(parent context.Context, status func(string)) (*Engine, error) {
	path, err := backendPath()
	if err != nil {
		return nil, err
	}
	status("connecting to Spotify…")
	engine, err := start(path)
	if err != nil {
		return nil, err
	}
	var hello struct {
		Protocol int
		Authed   bool
	}
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	err = engine.rpc.call(ctx, "hello", nil, &hello)
	cancel()
	if err != nil {
		engine.Close()
		return nil, err
	}
	if hello.Protocol != 1 {
		engine.Close()
		return nil, errors.New("unsupported Spotify engine protocol")
	}
	if !hello.Authed {
		var pairing struct{ URL, Code string }
		ctx, cancel = context.WithTimeout(parent, 30*time.Second)
		err = engine.rpc.call(ctx, "login_begin", nil, &pairing)
		cancel()
		if err != nil {
			engine.Close()
			return nil, err
		}
		status("Spotify: enter " + pairing.Code + " at " + pairing.URL)
		ctx, cancel = context.WithTimeout(parent, 5*time.Minute)
		err = engine.rpc.call(ctx, "login_wait", nil, nil)
		cancel()
		if err != nil {
			engine.Close()
			return nil, err
		}
	}
	status("connected · Spotify native playback")
	return engine, nil
}

func (e *Engine) Close() {
	if e == nil {
		return
	}
	e.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_ = e.rpc.call(ctx, "shutdown", nil, nil)
		cancel()
		e.rpc.close()
		select {
		case <-e.waited:
		case <-time.After(3 * time.Second):
			_ = e.cmd.Process.Kill()
			<-e.waited
		}
	})
}
func (e *Engine) call(method string, params any, out any) error {
	ctx, cancel := context.WithTimeout(context.Background(), methodTimeout(method))
	defer cancel()
	return e.rpc.call(ctx, method, params, out)
}

func methodTimeout(method string) time.Duration {
	switch method {
	case "lyrics":
		return 10 * time.Second
	case "artwork":
		return 15 * time.Second
	case "play", "reload", "jump", "play_pause", "next", "prev", "queue_edit":
		return 60 * time.Second
	case "save_playlist":
		return 310 * time.Second
	case "recent_history", "search", "library", "library_page", "artist_page", "queue_next", "queue_later", "save_item", "discovery", "discovery_item":
		return 35 * time.Second
	default:
		return 5 * time.Second
	}
}
func (e *Engine) State() (State, error) {
	var raw struct {
		Authed, Playing, Initializing bool
		PosMs, DurMs                  uint64
		Volume                        int
		Shuffle                       bool
		Repeat                        int
		Now                           *wireTrack
		Queue                         []wireTrack
		QueuePos                      int
		Err                           string
	}
	if err := e.call("state", nil, &raw); err != nil {
		return State{}, err
	}
	state := State{Authed: raw.Authed, Playing: raw.Playing, Initializing: raw.Initializing, Pos: time.Duration(raw.PosMs) * time.Millisecond, Dur: time.Duration(raw.DurMs) * time.Millisecond,
		Volume: raw.Volume, Shuffle: raw.Shuffle, Repeat: raw.Repeat, Queue: tracks(raw.Queue), QueuePos: raw.QueuePos, Err: raw.Err}
	if raw.Now != nil {
		state.Now = raw.Now.track()
	}
	return state, nil
}
func (e *Engine) Spectrum() ([32]float64, error) {
	var bands [32]float64
	err := e.call("spectrum", nil, &bands)
	return bands, err
}
func (e *Engine) results(method string, params any) (SearchResults, error) {
	var raw struct {
		Songs, Albums, Playlists, Recent, Artists []wireTrack
		Err                                       string
	}
	if err := e.call(method, params, &raw); err != nil {
		return SearchResults{}, err
	}
	result := SearchResults{Songs: tracks(raw.Songs), Albums: tracks(raw.Albums), Playlists: tracks(raw.Playlists), Recent: tracks(raw.Recent), Artists: tracks(raw.Artists)}
	if raw.Err != "" {
		return result, errors.New(raw.Err)
	}
	return result, nil
}
func (e *Engine) Search(term string) (SearchResults, error) {
	return e.results("search", map[string]any{"query": term})
}
func (e *Engine) Library() (SearchResults, error) { return e.results("library", nil) }
func (e *Engine) PlaybackDiagnostics() (PlaybackDiagnostics, error) {
	var report PlaybackDiagnostics
	err := e.call("playback_diagnostics", nil, &report)
	if len(report.Events) > 64 {
		report.Events = report.Events[len(report.Events)-64:]
	}
	for i := range report.Events {
		switch report.Events[i].FailureStage {
		case "metadata", "availability", "format", "download", "audio_key", "decode", "seek", "local_file", "internal":
		default:
			report.Events[i].FailureStage = ""
		}
		if code := report.Events[i].FailureKind; code != nil && (*code < -1 || *code > 16 || *code == 0 || report.Events[i].FailureStage == "") {
			report.Events[i].FailureKind = nil
		}
		switch report.Events[i].Event {
		case "playing", "paused", "loading", "stopped", "unavailable", "end_of_track", "track_changed", "progress", "handoff", "session_lost", "local_disconnect", "request_play", "request_pause", "load_sent", "early_end", "retry_start", "stream_failed":
		default:
			report.Events[i].Event = "other"
		}
	}
	return report, err
}
func (e *Engine) RecentHistory() (HistoryPage, error) {
	var raw struct {
		Recent            []wireTrack
		Err               string
		RetryAfterSeconds int64
	}
	if err := e.call("recent_history", nil, &raw); err != nil {
		return HistoryPage{}, err
	}
	page := HistoryPage{Tracks: tracks(raw.Recent), RetryAfter: time.Duration(min(max(raw.RetryAfterSeconds, 60), 31536000)) * time.Second}
	if raw.Err != "" {
		return page, errors.New(raw.Err)
	}
	return page, nil
}
func (e *Engine) Account() (Account, error) {
	var account Account
	err := e.call("account", nil, &account)
	return account, err
}
func (e *Engine) SignOut() error         { return e.call("sign_out", nil, nil) }
func (e *Engine) Reload() error          { return e.call("reload", nil, nil) }
func (e *Engine) PlayPause() error       { return e.call("play_pause", nil, nil) }
func (e *Engine) Next() error            { return e.call("next", nil, nil) }
func (e *Engine) Prev() error            { return e.call("prev", nil, nil) }
func (e *Engine) ToggleShuffle() error   { return e.call("toggle_shuffle", nil, nil) }
func (e *Engine) CycleRepeat() error     { return e.call("cycle_repeat", nil, nil) }
func (e *Engine) JumpTo(index int) error { return e.call("jump", map[string]any{"index": index}, nil) }
func (e *Engine) SeekTo(at time.Duration) error {
	return e.call("seek", map[string]any{"atMs": max(at.Milliseconds(), 0)}, nil)
}
func (e *Engine) SetVolume(volume int) error {
	return e.call("set_volume", map[string]any{"volume": min(max(volume, 0), 100)}, nil)
}
func (e *Engine) Play(kind, id string) error {
	return e.call("play", map[string]any{"kind": kind, "id": id}, nil)
}
func (e *Engine) QueueNext(kind, id string) error {
	return e.call("queue_next", map[string]any{"kind": kind, "id": id}, nil)
}
func (e *Engine) QueueLater(kind, id string) error {
	return e.call("queue_later", map[string]any{"kind": kind, "id": id}, nil)
}

func (e *Engine) EditQueue(action string, index, target int) error {
	return e.call("queue_edit", map[string]any{"action": action, "index": index, "target": target}, nil)
}
func (e *Engine) LibraryPage(category string, offset int) (SearchResults, error) {
	return e.results("library_page", map[string]any{"category": category, "offset": offset})
}
func (e *Engine) ArtistPage(id string) (SearchResults, error) {
	return e.results("artist_page", map[string]any{"id": id})
}

func (e *Engine) Discovery() (Discovery, error) {
	var feed Discovery
	err := e.call("discovery", nil, &feed)
	return feed, err
}

func (e *Engine) DiscoveryTracks(kind, id string) ([]Track, error) {
	var raw []wireTrack
	err := e.call("discovery_item", map[string]any{"kind": kind, "id": id}, &raw)
	return tracks(raw), err
}

func (e *Engine) SaveToLibrary(kind, id string) error {
	return e.call("save_item", map[string]any{"kind": kind, "id": id}, nil)
}

func (e *Engine) SavePlaylist(name string) error {
	_, err := e.results("save_playlist", map[string]any{"name": name})
	return err
}

func (e *Engine) Pause() error { return e.call("pause", nil, nil) }

func (e *Engine) Lyrics(id string) (TimedLyrics, error) {
	var result TimedLyrics
	err := e.call("lyrics", map[string]any{"id": id}, &result)
	return result, err
}

func (e *Engine) Artwork(kind, id string) (string, error) {
	var result struct {
		URL string `json:"url"`
	}
	err := e.call("artwork", map[string]any{"kind": kind, "id": id}, &result)
	return result.URL, err
}
