package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParsePagination(t *testing.T) {
	tests := []struct {
		query   string
		page    int
		perPage int
		offset  int
	}{
		{"", 1, 50, 0},
		{"page=2", 2, 50, 50},
		{"page=3&per_page=20", 3, 20, 40},
		{"page=0", 1, 50, 0},             // invalid page → default
		{"page=-1", 1, 50, 0},            // negative → default
		{"per_page=999", 1, 50, 0},       // over 200 → default
		{"per_page=200", 1, 200, 0},      // max allowed
		{"page=abc", 1, 50, 0},           // non-numeric → default
		{"per_page=abc", 1, 50, 0},       // non-numeric → default
	}

	for _, tt := range tests {
		r := httptest.NewRequest("GET", "/?"+tt.query, nil)
		pg := ParsePagination(r)

		if pg.Page != tt.page || pg.PerPage != tt.perPage || pg.Offset != tt.offset {
			t.Errorf("query=%q: got page=%d perPage=%d offset=%d, want page=%d perPage=%d offset=%d",
				tt.query, pg.Page, pg.PerPage, pg.Offset, tt.page, tt.perPage, tt.offset)
		}
	}
}

func TestParseTimeRange(t *testing.T) {
	r := httptest.NewRequest("GET", "/?from=2026-01-01&to=2026-03-26", nil)
	tr := ParseTimeRange(r)

	if tr.From == nil {
		t.Fatal("expected From to be set")
	}
	if tr.To == nil {
		t.Fatal("expected To to be set")
	}
	if tr.From.Year() != 2026 || tr.From.Month() != 1 || tr.From.Day() != 1 {
		t.Errorf("unexpected From: %v", tr.From)
	}
	if tr.To.Year() != 2026 || tr.To.Month() != 3 || tr.To.Day() != 26 {
		t.Errorf("unexpected To: %v", tr.To)
	}
}

func TestParseTimeRangeRFC3339(t *testing.T) {
	r := httptest.NewRequest("GET", "/?from=2026-01-01T00:00:00Z", nil)
	tr := ParseTimeRange(r)

	if tr.From == nil {
		t.Fatal("expected From to be set for RFC3339")
	}
}

func TestParseTimeRangeEmpty(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	tr := ParseTimeRange(r)

	if tr.From != nil || tr.To != nil {
		t.Error("expected nil From/To for empty query")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %s", body["status"])
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, http.StatusNotFound, "not found")

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "not found" {
		t.Errorf("expected 'not found', got %s", body["error"])
	}
}

func TestWritePaged(t *testing.T) {
	w := httptest.NewRecorder()
	pg := Pagination{Page: 2, PerPage: 10, Offset: 10}
	WritePaged(w, []string{"a", "b"}, pg, 100)

	var body PagedResponse
	json.NewDecoder(w.Body).Decode(&body)

	if body.Meta.Page != 2 {
		t.Errorf("expected page 2, got %d", body.Meta.Page)
	}
	if body.Meta.Total != 100 {
		t.Errorf("expected total 100, got %d", body.Meta.Total)
	}
}

func TestAdminAuth(t *testing.T) {
	handler := AdminAuth("secret-token", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"access": "granted"})
	})

	// No token
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	handler(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", w.Code)
	}

	// Wrong token
	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/", nil)
	r.Header.Set("Authorization", "Bearer wrong")
	handler(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong token, got %d", w.Code)
	}

	// Correct token
	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/", nil)
	r.Header.Set("Authorization", "Bearer secret-token")
	handler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with correct token, got %d", w.Code)
	}
}

func TestAdminAuthNotConfigured(t *testing.T) {
	handler := AdminAuth("", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	handler(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when admin not configured, got %d", w.Code)
	}
}

func TestQueryInt(t *testing.T) {
	r := httptest.NewRequest("GET", "/?limit=25&bad=abc", nil)

	if v := QueryInt(r, "limit", 10); v != 25 {
		t.Errorf("expected 25, got %d", v)
	}
	if v := QueryInt(r, "bad", 10); v != 10 {
		t.Errorf("expected default 10 for bad input, got %d", v)
	}
	if v := QueryInt(r, "missing", 42); v != 42 {
		t.Errorf("expected default 42, got %d", v)
	}
}
