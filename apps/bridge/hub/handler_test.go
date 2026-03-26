package hub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestWSHandlerUpgrade(t *testing.T) {
	h := New("gaende-favorites")
	srv := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:], nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.CloseNow()

	// Client should be registered
	time.Sleep(50 * time.Millisecond)
	if h.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", h.ClientCount())
	}

	conn.Close(websocket.StatusNormalClosure, "bye")
	time.Sleep(200 * time.Millisecond)

	if h.ClientCount() != 0 {
		t.Fatalf("expected 0 clients after close, got %d", h.ClientCount())
	}
}

func TestWSHandlerSubscribeAndReceive(t *testing.T) {
	h := New("gaende-favorites")
	srv := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:], nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.CloseNow()

	// Subscribe to a stage
	subMsg := ClientMessage{Type: "subscribe", Stages: []string{"gaende-favorites"}}
	data, _ := json.Marshal(subMsg)
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Wait for subscription to be processed
	time.Sleep(50 * time.Millisecond)

	// Broadcast a now_playing event
	h.BroadcastNowPlaying(NowPlayingEvent{
		StageID: "gaende-favorites",
		Status:  "play",
		Title:   "Meridian",
		Artist:  "ARTBAT",
	})

	// Read the message (may get snapshot first, then the broadcast)
	var lastType, lastStage string
	for i := 0; i < 3; i++ {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		var env Envelope
		json.Unmarshal(msg, &env)
		lastType = env.Type
		lastStage = env.StageID
		if env.Type == "now_playing" {
			break
		}
	}
	if lastType != "now_playing" {
		t.Errorf("expected type now_playing, got %s", lastType)
	}
	if lastStage != "gaende-favorites" {
		t.Errorf("expected stage gaende-favorites, got %s", lastStage)
	}
}

func TestWSHandlerUnsubscribe(t *testing.T) {
	h := New("gaende-favorites")
	srv := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:], nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.CloseNow()

	// Subscribe
	subMsg, _ := json.Marshal(ClientMessage{Type: "subscribe", Stages: []string{"gaende-favorites"}})
	conn.Write(ctx, websocket.MessageText, subMsg)
	time.Sleep(50 * time.Millisecond)

	// Unsubscribe
	unsubMsg, _ := json.Marshal(ClientMessage{Type: "unsubscribe", Stages: []string{"gaende-favorites"}})
	conn.Write(ctx, websocket.MessageText, unsubMsg)
	time.Sleep(50 * time.Millisecond)

	counts := h.ListenerCounts()
	if counts["gaende-favorites"] != 0 {
		t.Errorf("expected 0 listeners after unsubscribe, got %d", counts["gaende-favorites"])
	}
}

func TestWSHandlerInvalidJSON(t *testing.T) {
	h := New("gaende-favorites")
	srv := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:], nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.CloseNow()

	// Send invalid JSON — should not disconnect the client
	conn.Write(ctx, websocket.MessageText, []byte("not json"))
	time.Sleep(50 * time.Millisecond)

	// Connection should still be alive
	if h.ClientCount() != 1 {
		t.Fatalf("expected 1 client after invalid JSON, got %d", h.ClientCount())
	}

	// Can still subscribe after the bad message
	subMsg, _ := json.Marshal(ClientMessage{Type: "subscribe", Stages: []string{"gaende-favorites"}})
	if err := conn.Write(ctx, websocket.MessageText, subMsg); err != nil {
		t.Fatalf("write after invalid JSON failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	counts := h.ListenerCounts()
	if counts["gaende-favorites"] != 1 {
		t.Errorf("expected 1 listener after recovery, got %d", counts["gaende-favorites"])
	}
}

func TestWSHandlerUnknownMessageType(t *testing.T) {
	h := New("gaende-favorites")
	srv := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:], nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.CloseNow()

	// Send unknown type
	unknownMsg, _ := json.Marshal(ClientMessage{Type: "banana", Stages: []string{"gaende-favorites"}})
	conn.Write(ctx, websocket.MessageText, unknownMsg)
	time.Sleep(50 * time.Millisecond)

	// Connection still alive
	if h.ClientCount() != 1 {
		t.Fatalf("expected 1 client after unknown type, got %d", h.ClientCount())
	}
}

func TestSSEHandler(t *testing.T) {
	h := New()
	srv := httptest.NewServer(http.HandlerFunc(h.HandleSSE))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %s", ct)
	}

	// Wait for SSE client to register
	time.Sleep(50 * time.Millisecond)
	if h.SSEClientCount() != 1 {
		t.Fatalf("expected 1 SSE client, got %d", h.SSEClientCount())
	}

	// Send an event
	h.BroadcastSSE(SSEEvent{
		Event: "listeners",
		Data:  map[string]int{"gaende-favorites": 5},
		ID:    "evt-1",
	})

	buf := make([]byte, 1024)
	n, err := resp.Body.Read(buf)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	body := string(buf[:n])

	if !strings.Contains(body, "event: listeners") {
		t.Errorf("expected event: listeners in body, got %q", body)
	}
	if !strings.Contains(body, "id: evt-1") {
		t.Errorf("expected id: evt-1 in body, got %q", body)
	}
	if !strings.Contains(body, `"gaende-favorites":5`) {
		t.Errorf("expected listener data in body, got %q", body)
	}
}

func TestWSHandlerMultipleClients(t *testing.T) {
	h := New("gaende-favorites")
	srv := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conns := make([]*websocket.Conn, 3)
	for i := range conns {
		var err error
		conns[i], _, err = websocket.Dial(ctx, "ws"+srv.URL[4:], nil)
		if err != nil {
			t.Fatalf("dial %d failed: %v", i, err)
		}
		defer conns[i].CloseNow()

		subMsg, _ := json.Marshal(ClientMessage{Type: "subscribe", Stages: []string{"gaende-favorites"}})
		conns[i].Write(ctx, websocket.MessageText, subMsg)
	}

	time.Sleep(100 * time.Millisecond)

	if h.ClientCount() != 3 {
		t.Fatalf("expected 3 clients, got %d", h.ClientCount())
	}

	counts := h.ListenerCounts()
	if counts["gaende-favorites"] != 3 {
		t.Fatalf("expected 3 listeners, got %d", counts["gaende-favorites"])
	}

	h.BroadcastNowPlaying(NowPlayingEvent{
		StageID: "gaende-favorites",
		Status:  "play",
		Title:   "Test",
		Artist:  "Test",
	})

	for i, conn := range conns {
		_, _, err := conn.Read(ctx)
		if err != nil {
			t.Errorf("client %d read failed: %v", i, err)
		}
	}
}

func TestWSHandlerSnapshotOnSubscribe(t *testing.T) {
	h := New("s1")
	srv := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer srv.Close()

	// Broadcast before any client connects — creates snapshot
	h.BroadcastNowPlaying(NowPlayingEvent{StageID: "s1", Title: "Cached Track"})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[4:], nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.CloseNow()

	// Subscribe — should receive snapshot immediately
	subMsg, _ := json.Marshal(ClientMessage{Type: "subscribe", Stages: []string{"s1"}})
	conn.Write(ctx, websocket.MessageText, subMsg)

	// Read the snapshot
	_, msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var env Envelope
	json.Unmarshal(msg, &env)
	if env.Type != "now_playing" {
		t.Errorf("expected now_playing snapshot, got %s", env.Type)
	}
	if env.StageID != "s1" {
		t.Errorf("expected stage s1, got %s", env.StageID)
	}
}
