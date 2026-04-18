package mpd

import (
	"context"
	"sync"
	"testing"
	"time"

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

// TestIsAliveStrict verifies IsAlive is the strict "main client queryable now"
// signal — it does NOT fall back to watcher freshness. /api/stages uses this
// field and must see it flip to false the moment the main client is dead, so
// the frontend can correctly flag a stage as unqueryable. Watcher-based
// liveness for /health lives on HasRecentActivity (see tests below).
func TestIsAliveStrict(t *testing.T) {
	stages := []config.StageConfig{{ID: "s", MPDPort: 6600}}
	pool := NewPool(stages, nil)
	conn := pool.conns["s"]

	// Simulate a fresh watcher event; IsAlive should still report false
	// because the main client was markDead'd.
	conn.mu.Lock()
	conn.alive = false
	conn.client = nil
	conn.mu.Unlock()
	conn.lastEventAtNanos.Store(time.Now().UnixNano())
	if pool.IsAlive("s") {
		t.Fatal("IsAlive must be strict about main-client state, not watcher freshness")
	}

	// Main client revived → IsAlive true.
	conn.mu.Lock()
	conn.alive = true
	conn.mu.Unlock()
	if !pool.IsAlive("s") {
		t.Fatal("expected IsAlive=true when main client is alive")
	}
}

// TestHasRecentActivity_WatcherFallback guards the Phase-F /health regression
// fix. Scenario: the main client gets markDead'd (HTTP probe timeout, MPD
// connection_timeout, whatever) BUT the watcher's separate socket is still
// delivering events. HasRecentActivity should report true because the stage
// is producing real data — saying "down" in this state was the Phase-F
// false-negative.
func TestHasRecentActivity_WatcherFallback(t *testing.T) {
	stages := []config.StageConfig{{ID: "s", MPDPort: 6600}}
	pool := NewPool(stages, nil)
	conn := pool.conns["s"]

	conn.mu.Lock()
	conn.alive = false
	conn.client = nil
	conn.mu.Unlock()
	conn.lastEventAtNanos.Store(time.Now().UnixNano())

	if !pool.HasRecentActivity("s") {
		t.Fatal("expected HasRecentActivity=true when watcher event is fresh")
	}
}

// TestHasRecentActivity_StaleEvent — if the watcher's last event is older
// than eventFreshnessWindow, the fallback no longer rescues the stage.
// A genuinely-stuck watcher still reports dead.
func TestHasRecentActivity_StaleEvent(t *testing.T) {
	stages := []config.StageConfig{{ID: "s", MPDPort: 6600}}
	pool := NewPool(stages, nil)
	conn := pool.conns["s"]

	conn.mu.Lock()
	conn.alive = false
	conn.mu.Unlock()
	// Event from 20 minutes ago — well past the 9-minute window.
	conn.lastEventAtNanos.Store(time.Now().Add(-20 * time.Minute).UnixNano())

	if pool.HasRecentActivity("s") {
		t.Fatal("expected HasRecentActivity=false when last event is stale (>9m)")
	}
}

// TestHasRecentActivity_NeverHadEvent — a brand-new conn that never produced
// an event is dead when alive=false, regardless of clock (lastEventAtNanos=0).
func TestHasRecentActivity_NeverHadEvent(t *testing.T) {
	stages := []config.StageConfig{{ID: "s", MPDPort: 6600}}
	pool := NewPool(stages, nil)
	conn := pool.conns["s"]
	conn.mu.Lock()
	conn.alive = false
	conn.mu.Unlock()
	if pool.HasRecentActivity("s") {
		t.Fatal("expected HasRecentActivity=false when no event has ever been recorded")
	}
}

// TestHasRecentActivity_MainClientWins — if the main client is alive,
// HasRecentActivity should return true regardless of the event timestamp
// (initial connect/reconnect path, before the first track-change event).
func TestHasRecentActivity_MainClientWins(t *testing.T) {
	stages := []config.StageConfig{{ID: "s", MPDPort: 6600}}
	pool := NewPool(stages, nil)
	conn := pool.conns["s"]
	conn.mu.Lock()
	conn.alive = true
	conn.mu.Unlock()
	// No event recorded — but alive=true should dominate.
	if !pool.HasRecentActivity("s") {
		t.Fatal("expected HasRecentActivity=true when main client is alive")
	}
}

// TestEventFreshnessWindowBelowWatchdog guards the invariant that the
// watcher-freshness fallback must expire BEFORE the watchdog idle timeout,
// so a genuinely-dead watcher eventually gets reconnected instead of
// indefinitely claiming liveness from a stale timestamp. Codex round-1 on
// the P0 fix caught the 15m>10m inversion; this test locks that ordering in.
func TestEventFreshnessWindowBelowWatchdog(t *testing.T) {
	if eventFreshnessWindow >= watcherIdleTimeout {
		t.Fatalf(
			"eventFreshnessWindow (%v) must be < watcherIdleTimeout (%v); "+
				"otherwise HasRecentActivity reports alive for a stage whose "+
				"watcher is already being killed by the watchdog",
			eventFreshnessWindow, watcherIdleTimeout,
		)
	}
}

// TestKeepaliveIntervalBelowMPDDefault locks in the Bug-2 fix invariant:
// the bridge's keepalive must tick more often than MPD's server-side
// connection_timeout (default 60s). If we drift to >= 60s the idle timeout
// can fire between pings, MPD closes the main client, and we're back to the
// original "alive:false for every stage" cascade (/api/stages wedges because
// drainWatcher's event loop keeps resetting the 10-min watchdog forever).
//
// Regression: 2026-04-19 — see pool.go keepaliveInterval comment + codex
// challenge log in git history.
func TestKeepaliveIntervalBelowMPDDefault(t *testing.T) {
	const mpdDefaultConnectionTimeout = 60 * time.Second
	if keepaliveInterval >= mpdDefaultConnectionTimeout {
		t.Fatalf(
			"keepaliveInterval (%v) must be < MPD's default connection_timeout (%v); "+
				"otherwise the main client can be idle-closed between pings and the "+
				"bridge will fail to self-heal",
			keepaliveInterval, mpdDefaultConnectionTimeout,
		)
	}
}

// TestMarkDeadCASProtectsFreshClient — the race fix for Bug 2. When
// nowPlaying's Status() returns EOF, we markDead the EXACT client that
// failed, not a fresh one that a concurrent reconnect just swapped in.
//
// Setup: install fake client A, fail Status on it, concurrently swap to
// client B (simulating reconnect). markDead(A) must leave B alone.
func TestMarkDeadCASProtectsFreshClient(t *testing.T) {
	c := &Conn{Stage: config.StageConfig{ID: "s"}}

	// Use a non-nil sentinel pointer we can compare against. markDead only
	// touches c.client if expected matches — the client doesn't need to be a
	// real gompd.Client because markDead's Close() path is fire-and-forget.
	clientA := &gompd.Client{}
	clientB := &gompd.Client{}

	c.mu.Lock()
	c.client = clientA
	c.alive = true
	c.mu.Unlock()

	// Simulate a reconnect: main client swapped A → B while our caller was
	// mid-failure on A.
	c.mu.Lock()
	c.client = clientB
	c.alive = true
	c.mu.Unlock()

	// The caller still has a reference to the stale A and calls markDead(A).
	// CAS must refuse to nil out B.
	c.markDead(clientA)

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != clientB {
		t.Fatal("markDead(stale) must not swap out the current fresh client")
	}
	if !c.alive {
		t.Fatal("markDead(stale) must not flip alive for a fresh client")
	}
}

// TestKeepaliveLoopShutsDown — the keepalive goroutine exits promptly when
// context is cancelled; no leaks, no panics even when the conn has never
// been connected.
func TestKeepaliveLoopShutsDown(t *testing.T) {
	stages := []config.StageConfig{{ID: "s", MPDPort: 6600}}
	pool := NewPool(stages, nil)
	conn := pool.conns["s"]

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		pool.keepaliveLoop(ctx, conn)
	}()

	// Loop is running with a 30s ticker. Cancel and verify it exits quickly.
	cancel()
	select {
	case <-done:
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("keepaliveLoop did not exit within 2s of ctx cancel")
	}
}
