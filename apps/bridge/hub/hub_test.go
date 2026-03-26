package hub

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewHub(t *testing.T) {
	h := New("s1", "s2")
	if h == nil {
		t.Fatal("New() returned nil")
	}
	counts := h.ListenerCounts()
	if len(counts) != 0 {
		t.Fatalf("expected empty listener counts, got %v", counts)
	}
}

func TestRegisterUnregister(t *testing.T) {
	h := New()
	c := h.NewClient()

	h.Register(c)
	if h.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", h.ClientCount())
	}

	h.Unregister(c)
	if h.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", h.ClientCount())
	}
}

func TestUnregisterIdempotent(t *testing.T) {
	h := New()
	c := h.NewClient()
	h.Register(c)
	h.Unregister(c)
	h.Unregister(c) // should not panic
	if h.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", h.ClientCount())
	}
}

func TestSubscribe(t *testing.T) {
	h := New("gaende-favorites", "etage-0")
	c := h.NewClient()
	h.Register(c)

	c.Subscribe([]string{"gaende-favorites", "etage-0"})

	counts := h.ListenerCounts()
	if counts["gaende-favorites"] != 1 {
		t.Errorf("expected 1 listener on gaende-favorites, got %d", counts["gaende-favorites"])
	}
	if counts["etage-0"] != 1 {
		t.Errorf("expected 1 listener on etage-0, got %d", counts["etage-0"])
	}
}

func TestSubscribeIgnoresInvalidStages(t *testing.T) {
	h := New("gaende-favorites")
	c := h.NewClient()
	h.Register(c)

	c.Subscribe([]string{"gaende-favorites", "banana", "nonexistent"})

	stages := c.SubscribedStages()
	if len(stages) != 1 || stages[0] != "gaende-favorites" {
		t.Errorf("expected only gaende-favorites, got %v", stages)
	}
}

func TestUnsubscribe(t *testing.T) {
	h := New("gaende-favorites", "etage-0")
	c := h.NewClient()
	h.Register(c)

	c.Subscribe([]string{"gaende-favorites", "etage-0"})
	c.Unsubscribe([]string{"etage-0"})

	counts := h.ListenerCounts()
	if counts["gaende-favorites"] != 1 {
		t.Errorf("expected 1 listener on gaende-favorites, got %d", counts["gaende-favorites"])
	}
	if counts["etage-0"] != 0 {
		t.Errorf("expected 0 listeners on etage-0, got %d", counts["etage-0"])
	}
}

func TestBroadcastNowPlaying(t *testing.T) {
	h := New("gaende-favorites", "etage-0")
	c1 := h.NewClient()
	c2 := h.NewClient()
	h.Register(c1)
	h.Register(c2)

	c1.Subscribe([]string{"gaende-favorites"})
	c2.Subscribe([]string{"etage-0"})

	event := NowPlayingEvent{
		StageID:  "gaende-favorites",
		Status:   "play",
		Title:    "Meridian",
		Artist:   "ARTBAT",
		Album:    "Meridian EP",
		Elapsed:  "120.5",
		Duration: "420.0",
		File:     "/music/artbat-meridian.flac",
	}

	h.BroadcastNowPlaying(event)

	// c1 should receive (subscribed to gaende-favorites)
	select {
	case msg := <-c1.Send:
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if env.Type != "now_playing" {
			t.Errorf("expected type now_playing, got %s", env.Type)
		}
		if env.StageID != "gaende-favorites" {
			t.Errorf("expected stage_id gaende-favorites, got %s", env.StageID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("c1 did not receive broadcast")
	}

	// c2 should NOT receive (subscribed to etage-0, not gaende-favorites)
	select {
	case <-c2.Send:
		t.Error("c2 should not have received broadcast for gaende-favorites")
	case <-time.After(50 * time.Millisecond):
		// expected
	}
}

func TestBroadcastLatestWins(t *testing.T) {
	h := New("s1")
	c := h.NewClient()
	h.Register(c)
	c.Subscribe([]string{"s1"})

	// Fill the send buffer completely
	for i := 0; i < sendBufSize; i++ {
		h.BroadcastNowPlaying(NowPlayingEvent{StageID: "s1", Title: "old"})
	}

	// This broadcast should evict a stale message and deliver the latest
	h.BroadcastNowPlaying(NowPlayingEvent{StageID: "s1", Title: "latest"})

	// Drain all messages — the last one should be "latest"
	var lastTitle string
	for {
		select {
		case msg := <-c.Send:
			var env Envelope
			json.Unmarshal(msg, &env)
			if dataMap, ok := env.Data.(map[string]any); ok {
				if t, ok := dataMap["title"].(string); ok {
					lastTitle = t
				}
			}
		case <-time.After(50 * time.Millisecond):
			if lastTitle != "latest" {
				t.Errorf("expected last message to be 'latest', got %q", lastTitle)
			}
			return
		}
	}
}

func TestMultipleClientsPerStage(t *testing.T) {
	h := New("gaende-favorites")
	clients := make([]*Client, 5)
	for i := range clients {
		clients[i] = h.NewClient()
		h.Register(clients[i])
		clients[i].Subscribe([]string{"gaende-favorites"})
	}

	counts := h.ListenerCounts()
	if counts["gaende-favorites"] != 5 {
		t.Errorf("expected 5 listeners, got %d", counts["gaende-favorites"])
	}

	h.BroadcastNowPlaying(NowPlayingEvent{
		StageID: "gaende-favorites",
		Status:  "play",
		Title:   "Test",
		Artist:  "Test",
	})

	for i, c := range clients {
		select {
		case <-c.Send:
		case <-time.After(100 * time.Millisecond):
			t.Errorf("client %d did not receive broadcast", i)
		}
	}
}

func TestUnregisterCleansUpListenerCounts(t *testing.T) {
	h := New("gaende-favorites", "etage-0")
	c := h.NewClient()
	h.Register(c)
	c.Subscribe([]string{"gaende-favorites", "etage-0"})

	if h.ListenerCounts()["gaende-favorites"] != 1 {
		t.Fatal("expected 1 listener before unregister")
	}

	h.Unregister(c)

	counts := h.ListenerCounts()
	if counts["gaende-favorites"] != 0 {
		t.Errorf("expected 0 listeners after unregister, got %d", counts["gaende-favorites"])
	}
}

func TestSnapshot(t *testing.T) {
	h := New("s1", "s2")

	// No snapshot before any broadcast
	snaps := h.Snapshot([]string{"s1"})
	if len(snaps) != 0 {
		t.Fatalf("expected 0 snapshots, got %d", len(snaps))
	}

	// Broadcast caches snapshot
	h.BroadcastNowPlaying(NowPlayingEvent{StageID: "s1", Title: "Track A"})

	snaps = h.Snapshot([]string{"s1"})
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}

	var env Envelope
	json.Unmarshal(snaps[0], &env)
	if env.StageID != "s1" {
		t.Errorf("expected stage s1, got %s", env.StageID)
	}
}

func TestSSERegisterUnregister(t *testing.T) {
	h := New()

	sc := h.NewSSEClient()
	h.RegisterSSE(sc)

	if h.SSEClientCount() != 1 {
		t.Fatalf("expected 1 SSE client, got %d", h.SSEClientCount())
	}

	h.UnregisterSSE(sc)
	if h.SSEClientCount() != 0 {
		t.Fatalf("expected 0 SSE clients, got %d", h.SSEClientCount())
	}
}

func TestSSEBroadcast(t *testing.T) {
	h := New()

	sc := h.NewSSEClient()
	h.RegisterSSE(sc)

	h.BroadcastSSE(SSEEvent{
		Event: "listeners",
		Data:  map[string]int{"gaende-favorites": 3},
	})

	select {
	case ev := <-sc.Events:
		if ev.Event != "listeners" {
			t.Errorf("expected event type listeners, got %s", ev.Event)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("SSE client did not receive broadcast")
	}
}

func TestSSEBroadcastDropsWhenFull(t *testing.T) {
	h := New()

	sc := h.NewSSEClient()
	h.RegisterSSE(sc)

	// Fill the channel
	for i := 0; i < sseBufSize; i++ {
		h.BroadcastSSE(SSEEvent{Event: "listeners", Data: i})
	}

	// Should not block
	h.BroadcastSSE(SSEEvent{Event: "listeners", Data: "overflow"})
}

func TestSSENewlineInjection(t *testing.T) {
	h := New()

	sc := h.NewSSEClient()
	h.RegisterSSE(sc)

	h.BroadcastSSE(SSEEvent{
		Event: "listeners\ndata: {\"evil\":true}\n\nevent: injected",
		Data:  "legit",
		ID:    "1\ndata: pwned",
	})

	ev := <-sc.Events
	if strings.Contains(ev.Event, "\n") {
		t.Error("SSE event name should not contain newlines")
	}
	if strings.Contains(ev.ID, "\n") {
		t.Error("SSE ID should not contain newlines")
	}
}
