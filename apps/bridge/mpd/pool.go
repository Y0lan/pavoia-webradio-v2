package mpd

import (
	"context"
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

// Conn wraps a single MPD connection for one stage.
type Conn struct {
	Stage  config.StageConfig
	client *gompd.Client
	mu     sync.Mutex
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

// ConnectAll attempts to connect to every MPD instance.
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
	defer p.mu.RUnlock()

	result := make([]NowPlaying, 0, len(p.conns))
	for _, conn := range p.conns {
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
func (p *Pool) Close() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait() // Wait for all watcher goroutines to exit

	for _, c := range p.conns {
		// Detach the client under the mutex, then close it without the mutex held —
		// Close on a TCP connection that's stuck in Read can itself block indefinitely.
		c.mu.Lock()
		client := c.client
		c.client = nil
		c.alive = false
		c.mu.Unlock()
		if client != nil {
			_ = client.Close()
		}
	}
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

func (c *Conn) connect(addr string) error {
	// Detach any existing client under the mutex, close it outside.
	c.mu.Lock()
	oldClient := c.client
	c.client = nil
	c.alive = false
	c.mu.Unlock()
	if oldClient != nil {
		_ = oldClient.Close()
	}

	// gompd.Dial will fail naturally if the port is unreachable.
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

func (c *Conn) nowPlaying() NowPlaying {
	// Snapshot under the mutex, then release it before any blocking network I/O.
	// Holding c.mu across Status()/Ping()/CurrentSong() froze entire stages for weeks
	// in 2026-03 when a TCP Read never returned (silent NAT drop on shared hosting).
	c.mu.Lock()
	client := c.client
	alive := c.alive
	stageID := c.Stage.ID
	c.mu.Unlock()

	np := NowPlaying{StageID: stageID}

	if client == nil || !alive {
		np.Status = "offline"
		np.Error = "not connected"
		return np
	}

	status, err := client.Status()
	if err != nil {
		// Try a ping before declaring dead — could be a transient error.
		if pingErr := client.Ping(); pingErr != nil {
			np.Status = "error"
			np.Error = err.Error()
			// Compare-and-swap: only null out c.client if it's still the same one we read.
			// A parallel reconnect() may have already replaced it with a healthy connection.
			c.mu.Lock()
			if c.client == client {
				c.alive = false
				c.client = nil
			}
			c.mu.Unlock()
			// Close outside the mutex — Close on a hung TCP conn can itself block.
			_ = client.Close()
			return np
		}
		// Ping succeeded — transient, don't kill the connection.
		np.Status = "error"
		np.Error = err.Error()
		return np
	}

	np.Status = status["state"]
	np.Elapsed = status["elapsed"]
	np.Duration = status["duration"]

	song, err := client.CurrentSong()
	if err != nil {
		np.Error = err.Error()
		return np
	}
	np.Song = song

	return np
}

// watchLoop watches for MPD player changes using idle-based notifications.
// It reuses a single watcher per stage instead of creating/destroying on each event.
// Exits when ctx is cancelled.
func (p *Pool) watchLoop(ctx context.Context, conn *Conn, host string) {
	addr := fmt.Sprintf("%s:%d", host, conn.Stage.MPDPort)
	var lastSong string

	for {
		// Check for shutdown
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn.mu.Lock()
		alive := conn.alive
		conn.mu.Unlock()

		if !alive {
			// Reconnect with exponential backoff, respecting context
			if !p.reconnect(ctx, conn, addr) {
				return // context cancelled during reconnect
			}
		}

		// Get current state and notify if changed
		np := conn.nowPlaying()
		songKey := np.Song["file"]
		if songKey != lastSong && songKey != "" {
			lastSong = songKey
			if p.onChange != nil {
				p.onChange(np)
			}
		}

		// Create a watcher and reuse it for multiple events
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

		// Wait for events from this watcher until it errors or context cancels
		p.drainWatcher(ctx, watcher, conn, &lastSong)
		watcher.Close()
	}
}

// drainWatcher reads events from a watcher until error, context cancellation, or
// the watchdog fires. The watchdog guards against gompd's internal read goroutine
// blocking forever on a silently-dead TCP connection (no FIN, no RST, no error) —
// which is what froze every stage for weeks in 2026-03.
func (p *Pool) drainWatcher(ctx context.Context, watcher *gompd.Watcher, conn *Conn, lastSong *string) {
	// 10 min is comfortably above any realistic track length. False positives only
	// cost one watcher recreate + reconnect (a few seconds on healthy infra).
	const watcherIdleTimeout = 10 * time.Minute

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(watcherIdleTimeout):
			slog.Warn("mpd watcher idle timeout, resetting", "stage", conn.Stage.ID, "after", watcherIdleTimeout)
			return
		case _, ok := <-watcher.Event:
			if !ok {
				return // channel closed
			}
			// Track changed — get new state and notify.
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
			return // will recreate watcher on next loop iteration
		}
	}
}

// reconnect attempts to reconnect with exponential backoff.
// Returns false if ctx is cancelled (caller should exit).
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
