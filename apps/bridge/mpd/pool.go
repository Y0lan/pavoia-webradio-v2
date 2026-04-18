package mpd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	gompd "github.com/fhs/gompd/v2/mpd"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
)

// NowPlaying holds the current track info for a stage.
type NowPlaying struct {
	StageID  string            `json:"stage_id"`
	Status   string            `json:"status"` // "play", "pause", "stop", "offline", "error"
	Song     map[string]string `json:"song"`
	Elapsed  string            `json:"elapsed"`
	Duration string            `json:"duration"`
	Error    string            `json:"error,omitempty"`
}

// defaultCmdTimeout bounds every blocking MPD command. Shared hosting NAT can silently
// drop long-lived connections; without a per-call ceiling a single bad socket would
// wedge every caller waiting on ioMu forever.
const defaultCmdTimeout = 5 * time.Second

// watcherIdleTimeout — if no events on a watcher for this long, assume the underlying
// idle socket has zombied (gompd's Read goroutine blocked with no error) and force a
// full reset: mark the main connection dead, close it, rebuild on the next loop iteration.
// 10 min is comfortably above any realistic track length; false positives only cost a reconnect.
const watcherIdleTimeout = 10 * time.Minute

// Conn wraps a single MPD connection for one stage.
//
// Concurrency model:
//
//   - `mu` protects the `client` pointer and `alive` flag — always short-held,
//     never wraps a blocking call.
//   - `ioMu` serializes MPD protocol commands on `client` (gompd's protocol is
//     single-connection request/response and racing calls interleaves frames).
//     Held only by `withClient`; every call inside it is guarded by `callWithTimeout`
//     so a hung Read can't wedge the mutex queue.
type Conn struct {
	Stage  config.StageConfig
	client *gompd.Client
	mu     sync.Mutex
	ioMu   sync.Mutex
	alive  bool
}

// Pool manages connections to all MPD instances.
type Pool struct {
	conns    map[string]*Conn
	mu       sync.RWMutex
	onChange func(np NowPlaying)
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewPool creates a pool from stage configs. Does not connect yet.
func NewPool(stages []config.StageConfig, onChange func(NowPlaying)) *Pool {
	p := &Pool{
		conns:    make(map[string]*Conn, len(stages)),
		onChange: onChange,
	}
	for _, s := range stages {
		p.conns[s.ID] = &Conn{Stage: s}
	}
	return p
}

// ConnectAll attempts to connect to every MPD instance concurrently.
// Returns the number of successful connections.
func (p *Pool) ConnectAll(host string) int {
	var wg sync.WaitGroup
	var connected int
	var connMu sync.Mutex

	for _, c := range p.conns {
		wg.Add(1)
		go func(conn *Conn) {
			defer wg.Done()
			addr := fmt.Sprintf("%s:%d", host, conn.Stage.MPDPort)
			if err := conn.connect(addr); err != nil {
				slog.Warn("mpd connect failed", "stage", conn.Stage.ID, "addr", addr, "error", err)
				return
			}
			connMu.Lock()
			connected++
			connMu.Unlock()
			slog.Info("mpd connected", "stage", conn.Stage.ID, "addr", addr)
		}(c)
	}
	wg.Wait()
	return connected
}

// NowPlaying returns current track info for a stage.
func (p *Pool) NowPlaying(stageID string) NowPlaying {
	p.mu.RLock()
	conn, ok := p.conns[stageID]
	p.mu.RUnlock()
	if !ok {
		return NowPlaying{StageID: stageID, Status: "unknown", Error: "stage not found"}
	}
	return conn.nowPlaying()
}

// AllNowPlaying returns now-playing for all stages.
func (p *Pool) AllNowPlaying() []NowPlaying {
	p.mu.RLock()
	conns := make([]*Conn, 0, len(p.conns))
	for _, c := range p.conns {
		conns = append(conns, c)
	}
	p.mu.RUnlock()

	result := make([]NowPlaying, 0, len(conns))
	for _, conn := range conns {
		result = append(result, conn.nowPlaying())
	}
	return result
}

// StartWatchers launches a goroutine per stage that watches for track changes.
// Pass the parent context for graceful shutdown.
func (p *Pool) StartWatchers(ctx context.Context, host string) {
	watchCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	for _, c := range p.conns {
		p.wg.Add(1)
		go func(conn *Conn) {
			defer p.wg.Done()
			p.watchLoop(watchCtx, conn, host)
		}(c)
	}
}

// Close disconnects all MPD connections and stops all watchers.
//
// Order matters: we close the clients FIRST (outside any mutex), which causes
// any blocked Read/Write in the watcher goroutines to return immediately with
// an error. Then we cancel the context and wait for the goroutines to exit.
// Waiting on p.wg.Wait before closing would let a stuck I/O call deadlock shutdown.
func (p *Pool) Close() {
	// Step 1: cancel context (watcher select statements notice).
	if p.cancel != nil {
		p.cancel()
	}

	// Step 2: detach + close every client outside any lock. Closing a connection
	// whose Read is blocked unblocks it with io.ErrClosed, letting the goroutine exit.
	for _, c := range p.conns {
		c.mu.Lock()
		client := c.client
		c.client = nil
		c.alive = false
		c.mu.Unlock()
		if client != nil {
			_ = client.Close()
		}
	}

	// Step 3: now that clients are gone, watchers will exit promptly.
	p.wg.Wait()
}

// IsAlive returns whether a stage's MPD connection is active.
func (p *Pool) IsAlive(stageID string) bool {
	p.mu.RLock()
	conn, ok := p.conns[stageID]
	p.mu.RUnlock()
	if !ok {
		return false
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()
	return conn.alive
}

// connect replaces the current MPD client with a freshly-dialed one.
// Old client is closed outside the state mutex.
func (c *Conn) connect(addr string) error {
	c.mu.Lock()
	oldClient := c.client
	c.client = nil
	c.alive = false
	c.mu.Unlock()
	if oldClient != nil {
		_ = oldClient.Close()
	}

	client, err := gompd.Dial("tcp", addr)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.client = client
	c.alive = true
	c.mu.Unlock()
	return nil
}

// markDead forces the connection into a disconnected state and closes the
// client outside the state mutex. Safe to call from any goroutine. Idempotent.
// If `expected` is non-nil, only mark dead if c.client still matches it
// (CAS-style — avoids nullifying a fresh client a reconnect just swapped in).
func (c *Conn) markDead(expected *gompd.Client) {
	c.mu.Lock()
	if expected != nil && c.client != expected {
		c.mu.Unlock()
		return
	}
	client := c.client
	c.client = nil
	c.alive = false
	c.mu.Unlock()
	if client != nil {
		// Close can block on a hung socket; fire-and-forget. Worst case is one
		// leaked goroutine per stuck connection (bounded at 9).
		go client.Close()
	}
}

// withClient serializes an MPD command on client, bounded by defaultCmdTimeout.
// Returns (false, nil) if the connection isn't alive or if client was swapped
// while we waited for ioMu. Returns (true, err) if the command completed.
//
// The timeout matters: without it, a Read blocked on a silently-dead TCP socket
// holds ioMu forever, and every subsequent caller queues behind it — same
// failure mode as the original mutex-across-I/O bug, just moved one layer out.
func (c *Conn) withClient(fn func(*gompd.Client) error) (ok bool, err error) {
	c.mu.Lock()
	client := c.client
	alive := c.alive
	c.mu.Unlock()
	if client == nil || !alive {
		return false, nil
	}

	c.ioMu.Lock()
	defer c.ioMu.Unlock()

	// Re-validate: another goroutine may have swapped the client while we waited.
	c.mu.Lock()
	same := c.client == client && c.alive
	c.mu.Unlock()
	if !same {
		return false, nil
	}

	done := make(chan error, 1)
	go func() { done <- fn(client) }()

	select {
	case err = <-done:
		if err != nil {
			return true, err
		}
		return true, nil
	case <-time.After(defaultCmdTimeout):
		// Timeout — mark dead so the next watcher iteration reconnects.
		c.markDead(client)
		return true, errors.New("mpd command timeout")
	}
}

func (c *Conn) nowPlaying() NowPlaying {
	np := NowPlaying{StageID: c.Stage.ID}

	var status map[string]string
	ok, err := c.withClient(func(client *gompd.Client) error {
		var e error
		status, e = client.Status()
		return e
	})
	if !ok {
		np.Status = "offline"
		np.Error = "not connected"
		return np
	}
	if err != nil {
		// Probe with Ping; a transient error doesn't warrant killing the connection.
		_, pingErr := c.withClient(func(client *gompd.Client) error { return client.Ping() })
		if pingErr != nil {
			c.mu.Lock()
			client := c.client
			c.mu.Unlock()
			c.markDead(client)
		}
		np.Status = "error"
		np.Error = err.Error()
		return np
	}

	np.Status = status["state"]
	np.Elapsed = status["elapsed"]
	np.Duration = status["duration"]

	var song map[string]string
	_, songErr := c.withClient(func(client *gompd.Client) error {
		var e error
		song, e = client.CurrentSong()
		return e
	})
	if songErr != nil {
		np.Error = songErr.Error()
		return np
	}
	np.Song = song
	return np
}

// watchLoop watches for MPD player changes using idle-based notifications.
// Exits when ctx is cancelled.
func (p *Pool) watchLoop(ctx context.Context, conn *Conn, host string) {
	addr := fmt.Sprintf("%s:%d", host, conn.Stage.MPDPort)
	var lastSong string

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn.mu.Lock()
		alive := conn.alive
		conn.mu.Unlock()

		if !alive {
			if !p.reconnect(ctx, conn, addr) {
				return
			}
		}

		// Emit current state at least once per connection so snapshots stay fresh.
		np := conn.nowPlaying()
		songKey := np.Song["file"]
		if songKey != lastSong && songKey != "" {
			lastSong = songKey
			if p.onChange != nil {
				p.onChange(np)
			}
		}

		// Create a watcher on its own dedicated socket (independent from conn.client).
		watcher, err := gompd.NewWatcher("tcp", addr, "", "player")
		if err != nil {
			slog.Warn("mpd watcher failed", "stage", conn.Stage.ID, "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		p.drainWatcher(ctx, watcher, conn, &lastSong)
		_ = watcher.Close()
	}
}

// drainWatcher reads events from a watcher until context cancellation, watcher
// error, or the idle watchdog fires.
//
// On watchdog: we mark conn dead and close the main client. Otherwise the outer
// loop would immediately call nowPlaying() on the same dead socket and wedge
// again — the failure that kept the bridge stalled for 22 days in 2026-03.
//
// Uses a single shared timer rather than `time.After` per select iteration so
// expired timers don't pile up on high-traffic stages.
func (p *Pool) drainWatcher(ctx context.Context, watcher *gompd.Watcher, conn *Conn, lastSong *string) {
	timer := time.NewTimer(watcherIdleTimeout)
	defer timer.Stop()

	resetTimer := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(watcherIdleTimeout)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			slog.Warn("mpd watcher idle timeout, resetting", "stage", conn.Stage.ID, "after", watcherIdleTimeout)
			// Force a full reconnect: the main client may share the same NAT state as
			// the watcher's silently-dead socket, so killing it too lets the outer loop
			// reconnect cleanly on its next iteration.
			conn.mu.Lock()
			client := conn.client
			conn.mu.Unlock()
			conn.markDead(client)
			return
		case _, ok := <-watcher.Event:
			if !ok {
				return
			}
			resetTimer()
			np := conn.nowPlaying()
			songKey := np.Song["file"]
			if songKey != *lastSong && songKey != "" {
				*lastSong = songKey
				if p.onChange != nil {
					p.onChange(np)
				}
			}
		case err := <-watcher.Error:
			slog.Warn("mpd watcher error", "stage", conn.Stage.ID, "error", err)
			return
		}
	}
}

// reconnect loops with exponential backoff until it succeeds or ctx is cancelled.
func (p *Pool) reconnect(ctx context.Context, conn *Conn, addr string) bool {
	for delay := time.Second; ; delay = min(delay*2, 30*time.Second) {
		slog.Info("mpd reconnecting", "stage", conn.Stage.ID, "delay", delay)
		select {
		case <-ctx.Done():
			return false
		case <-time.After(delay):
		}
		if err := conn.connect(addr); err == nil {
			slog.Info("mpd reconnected", "stage", conn.Stage.ID)
			return true
		}
	}
}
