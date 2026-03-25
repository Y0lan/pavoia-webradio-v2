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
		c.mu.Lock()
		if c.client != nil {
			c.client.Close()
			c.client = nil
		}
		c.alive = false
		c.mu.Unlock()
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		c.client.Close()
	}

	// gompd.Dial will fail naturally if the port is unreachable.
	// No need for a separate test connection (removes TOCTOU race).
	client, err := gompd.Dial("tcp", addr)
	if err != nil {
		c.alive = false
		c.client = nil
		return err
	}
	c.client = client
	c.alive = true
	return nil
}

func (c *Conn) nowPlaying() NowPlaying {
	c.mu.Lock()
	defer c.mu.Unlock()

	np := NowPlaying{StageID: c.Stage.ID}

	if c.client == nil || !c.alive {
		np.Status = "offline"
		np.Error = "not connected"
		return np
	}

	status, err := c.client.Status()
	if err != nil {
		// Try a ping before declaring dead — could be a transient error
		if pingErr := c.client.Ping(); pingErr != nil {
			np.Status = "error"
			np.Error = err.Error()
			c.alive = false
			c.client.Close()
			c.client = nil // Clear client so next call returns "offline" immediately
			return np
		}
		// Ping succeeded — transient error, don't kill the connection
		np.Status = "error"
		np.Error = err.Error()
		return np
	}

	np.Status = status["state"]
	np.Elapsed = status["elapsed"]
	np.Duration = status["duration"]

	song, err := c.client.CurrentSong()
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

// drainWatcher reads events from a watcher until error or context cancellation.
// This reuses a single connection for multiple events instead of creating a new one per event.
func (p *Pool) drainWatcher(ctx context.Context, watcher *gompd.Watcher, conn *Conn, lastSong *string) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-watcher.Event:
			if !ok {
				return // channel closed
			}
			// Track changed — get new state and notify
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
