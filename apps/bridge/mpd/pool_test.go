package mpd

import (
	"testing"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
)

func TestNewPool(t *testing.T) {
	stages := []config.StageConfig{
		{ID: "test-1", Name: "Test 1", MPDPort: 6600},
		{ID: "test-2", Name: "Test 2", MPDPort: 6601},
	}

	called := 0
	pool := NewPool(stages, func(np NowPlaying) {
		called++
	})

	if len(pool.conns) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(pool.conns))
	}

	if _, ok := pool.conns["test-1"]; !ok {
		t.Fatal("missing connection for test-1")
	}
	if _, ok := pool.conns["test-2"]; !ok {
		t.Fatal("missing connection for test-2")
	}
}

func TestNowPlayingOffline(t *testing.T) {
	stages := []config.StageConfig{
		{ID: "offline-stage", Name: "Offline", MPDPort: 6600},
	}
	pool := NewPool(stages, nil)

	np := pool.NowPlaying("offline-stage")
	if np.Status != "offline" {
		t.Fatalf("expected status 'offline', got '%s'", np.Status)
	}
	if np.Error != "not connected" {
		t.Fatalf("expected error 'not connected', got '%s'", np.Error)
	}
}

func TestNowPlayingUnknownStage(t *testing.T) {
	pool := NewPool(nil, nil)
	np := pool.NowPlaying("nonexistent")
	if np.Status != "unknown" {
		t.Fatalf("expected status 'unknown', got '%s'", np.Status)
	}
}

func TestAllNowPlaying(t *testing.T) {
	stages := []config.StageConfig{
		{ID: "s1", Name: "S1", MPDPort: 6600},
		{ID: "s2", Name: "S2", MPDPort: 6601},
		{ID: "s3", Name: "S3", MPDPort: 6602},
	}
	pool := NewPool(stages, nil)

	all := pool.AllNowPlaying()
	if len(all) != 3 {
		t.Fatalf("expected 3 results, got %d", len(all))
	}

	// All should be offline since we haven't connected
	for _, np := range all {
		if np.Status != "offline" {
			t.Errorf("stage %s: expected 'offline', got '%s'", np.StageID, np.Status)
		}
	}
}

func TestIsAlive(t *testing.T) {
	stages := []config.StageConfig{
		{ID: "test", Name: "Test", MPDPort: 6600},
	}
	pool := NewPool(stages, nil)

	if pool.IsAlive("test") {
		t.Fatal("expected not alive before connect")
	}
	if pool.IsAlive("nonexistent") {
		t.Fatal("expected false for nonexistent stage")
	}
}

func TestConnectAllNoServer(t *testing.T) {
	stages := []config.StageConfig{
		{ID: "fail-1", Name: "Fail 1", MPDPort: 19999},
		{ID: "fail-2", Name: "Fail 2", MPDPort: 19998},
	}
	pool := NewPool(stages, nil)

	// Connect to a port where nothing is listening — should return 0
	connected := pool.ConnectAll("127.0.0.1")
	if connected != 0 {
		t.Fatalf("expected 0 connections, got %d", connected)
	}
}
