package mpd

import (
	"sync"
	"testing"

	gompd "github.com/fhs/gompd/v2/mpd"

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

// TestMarkDeadWhenNoClient exercises the idempotent-when-empty path. This is the
// call pattern used by the watchdog + error handling code — they shouldn't crash
// or deadlock when the connection is already gone.
func TestMarkDeadWhenNoClient(t *testing.T) {
	c := &Conn{Stage: config.StageConfig{ID: "stage"}, alive: true}
	c.markDead(nil)
	if c.alive {
		t.Fatal("expected alive=false after markDead")
	}
	if c.client != nil {
		t.Fatal("expected client=nil after markDead")
	}
	// Second call is a no-op, must not panic or block.
	c.markDead(nil)
}

// TestWithClientOfflineReturnsFalse verifies that withClient short-circuits
// when the connection is down. This is the fast path used by HTTP handlers
// so a dead stage doesn't block any of its siblings.
func TestWithClientOfflineReturnsFalse(t *testing.T) {
	c := &Conn{Stage: config.StageConfig{ID: "stage"}}
	called := false
	ok, err := c.withClient(func(client *gompd.Client) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false when offline")
	}
	if called {
		t.Fatal("fn must not be invoked when offline")
	}
}

// TestConcurrentNowPlayingNoDataRace exercises the two-mutex pattern under -race.
// If we ever reintroduce racy access to Conn.client / Conn.alive, the Go race
// detector will catch it here.
func TestConcurrentNowPlayingNoDataRace(t *testing.T) {
	stages := []config.StageConfig{{ID: "s", MPDPort: 6600}}
	pool := NewPool(stages, nil)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _ = pool.IsAlive("s") }()
		go func() { defer wg.Done(); _ = pool.NowPlaying("s") }()
	}
	wg.Wait()
}
