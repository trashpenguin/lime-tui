package engine

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"
)

func TestPlaybackDiagnosticsDropsUnexpectedPrivateFields(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	rpc := newRPC(client, client)
	defer rpc.close()
	eng := &Engine{rpc: rpc}
	go func() {
		var request map[string]any
		if json.NewDecoder(server).Decode(&request) != nil {
			return
		}
		_ = json.NewEncoder(server).Encode(map[string]any{"id": request["id"], "result": map[string]any{
			"username": "private-fixture", "track_id": "private-fixture", "connected": true,
			"events": []map[string]any{{"event": "private-fixture", "title": "private-fixture"}, {"event": "stopped", "positionMs": 1000}},
		}})
	}()
	report, err := eng.PlaybackDiagnostics()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private-fixture") || report.Events[1].Event != "stopped" {
		t.Fatal("private/unrecognized trace fields escaped the allowlist")
	}
}

func TestHistoryRPCMapsSpotifyMetadataAndPreservesRetryDeadline(t *testing.T) {
	for _, failed := range []bool{false, true} {
		client, server := net.Pipe()
		rpc := newRPC(client, client)
		eng := &Engine{rpc: rpc}
		go func() {
			defer server.Close()
			var request map[string]any
			if json.NewDecoder(server).Decode(&request) != nil {
				return
			}
			var err any
			if failed {
				err = "Spotify history rate-limited"
			}
			_ = json.NewEncoder(server).Encode(map[string]any{"id": request["id"], "result": map[string]any{
				"recent": []map[string]any{{"id": "fixture", "title": "Remote play", "kind": "songs", "durMs": 120000, "art": "https://example.invalid/cover"}},
				"err":    err, "retryAfterSeconds": 7200,
			}})
		}()
		page, err := eng.RecentHistory()
		rpc.close()
		if (err != nil) != failed || page.RetryAfter != 2*time.Hour {
			t.Fatal("lost error or shortened server cooldown", err)
		}
		if len(page.Tracks) != 1 || page.Tracks[0].Kind != "song" || page.Tracks[0].Duration != 2*time.Minute || page.Tracks[0].Art == "" {
			t.Fatal("Spotify history metadata lost")
		}
	}
}
