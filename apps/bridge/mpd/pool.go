package mpd

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	gompd "github.com/fhs/gompd/v2/mpd"

	"github.com/Y0lan/pavoia-webradio-v2/apps/bridge/config"
)

// NowPlaying holds the current track info for a stage.
type NowPlaying struct {
	StageID  string            `json:"stage_id"`
	Status   string            `json:"status"` // "play", "pause", "stop"
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
	onChange func(np NowPlaying) // callback when now-playing changes
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

// StartWatchers launches a goroutine per stage that polls for track changes.
func (p *Pool) StartWatchers(host string) {
	for _, c := range p.conns {
		go p.watchLoop(c, host)
	}
}

// Close disconnects all MPD connections.
func (p *Pool) Close() {
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

	// Verify port is reachable with a timeout before gompd.Dial (which has no timeout)
	testConn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		c.alive = false
		c.client = nil
		return err
	}
	testConn.Close()

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
		np.Status = "error"
		np.Error = err.Error()
		c.alive = false
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

// watchLoop polls MPD for changes and calls onChange when the track changes.
// Uses MPD's idle command for efficient waiting, with reconnection on failure.
func (p *Pool) watchLoop(conn *Conn, host string) {
	addr := fmt.Sprintf("%s:%d", host, conn.Stage.MPDPort)
	var lastSong string

	for {
		conn.mu.Lock()
		alive := conn.alive
		conn.mu.Unlock()

		if !alive {
			// Reconnect with exponential backoff
			for delay := time.Second; ; delay = min(delay*2, 30*time.Second) {
				slog.Info("mpd reconnecting", "stage", conn.Stage.ID, "delay", delay)
				time.Sleep(delay)
				if err := conn.connect(addr); err == nil {
					slog.Info("mpd reconnected", "stage", conn.Stage.ID)
					break
				}
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

		// Wait for MPD player subsystem change (blocks until something happens)
		conn.mu.Lock()
		client := conn.client
		conn.mu.Unlock()

		if client == nil {
			time.Sleep(2 * time.Second)
			continue
		}

		// Use a watcher for efficient idle-based notifications
		watcher, err := gompd.NewWatcher("tcp", addr, "", "player")
		if err != nil {
			slog.Warn("mpd watcher failed", "stage", conn.Stage.ID, "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		select {
		case <-watcher.Event:
			// Track changed, loop will pick up new state
		case err := <-watcher.Error:
			slog.Warn("mpd watcher error", "stage", conn.Stage.ID, "error", err)
		}
		watcher.Close()
	}
}
