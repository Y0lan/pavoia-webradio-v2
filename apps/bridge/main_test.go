package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseDurationSec(t *testing.T) {
	cases := []struct {
		in   string
		want any
	}{
		{"234.123", 234}, // rounds down (<.5)
		{"234.5", 235},   // rounds up — was truncated before the math.Round fix
		{"234.9", 235},
		{"60", 60},
		{"0.5", nil},     // sub-second tracks are meaningless here; store NULL
		{"0.9", nil},     // still rounds to <1s
		{"0", nil},
		{"", nil},
		{"not a number", nil},
		{"-12.0", nil},   // negative durations are noise
	}
	for _, tc := range cases {
		got := parseDurationSec(tc.in)
		if got != tc.want {
			t.Errorf("parseDurationSec(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestOverallHealth(t *testing.T) {
	cases := []struct {
		name   string
		checks map[string]string
		want   string
	}{
		{
			name: "all green",
			checks: map[string]string{
				"mpd": "ok", "postgres": "ok", "plex": "ok",
				"redis": "not_connected", "meilisearch": "not_connected",
			},
			want: "ok",
		},
		{
			name: "postgres down -> down (critical)",
			checks: map[string]string{
				"mpd": "ok", "postgres": "down", "plex": "ok",
				"redis": "not_connected", "meilisearch": "not_connected",
			},
			want: "down",
		},
		{
			name: "mpd entirely down -> down (critical)",
			checks: map[string]string{
				"mpd": "down", "postgres": "ok", "plex": "ok",
				"redis": "not_connected", "meilisearch": "not_connected",
			},
			want: "down",
		},
		{
			name: "mpd partial -> degraded",
			checks: map[string]string{
				"mpd": "partial (5/9)", "postgres": "ok", "plex": "ok",
				"redis": "not_connected", "meilisearch": "not_connected",
			},
			want: "degraded",
		},
		{
			name: "plex down -> degraded (advisory only)",
			checks: map[string]string{
				"mpd": "ok", "postgres": "ok", "plex": "down",
				"redis": "not_connected", "meilisearch": "not_connected",
			},
			want: "degraded",
		},
		{
			name: "everything not_configured -> ok",
			checks: map[string]string{
				"mpd": "ok", "postgres": "ok", "plex": "not_configured",
				"redis": "not_connected", "meilisearch": "not_connected",
			},
			want: "ok",
		},
		{
			name: "stale bridge (mpd down, postgres ok) -> down — the exact 2026-04 failure",
			checks: map[string]string{
				"mpd": "down", "postgres": "ok", "plex": "not_configured",
				"redis": "not_connected", "meilisearch": "not_connected",
			},
			want: "down",
		},
		{
			name: "disk sync stale but everything else ok -> degraded (advisory)",
			checks: map[string]string{
				"mpd": "ok", "postgres": "ok", "plex": "not_used",
				"redis": "not_connected", "meilisearch": "not_connected",
				"disk_sync": "stale",
			},
			want: "degraded",
		},
		{
			name: "disk sync never_ran during cold boot -> degraded so it surfaces",
			checks: map[string]string{
				"mpd": "ok", "postgres": "ok", "plex": "not_used",
				"redis": "not_connected", "meilisearch": "not_connected",
				"disk_sync": "never_ran",
			},
			want: "degraded",
		},
		{
			name: "pg_backup stale -> degraded (backups silently stopped)",
			checks: map[string]string{
				"mpd": "ok", "postgres": "ok", "plex": "not_used",
				"redis": "not_connected", "meilisearch": "not_connected",
				"disk_sync": "ok", "pg_backup": "stale",
			},
			want: "degraded",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := overallHealth(tc.checks)
			if got != tc.want {
				t.Errorf("overallHealth(%v) = %q, want %q", tc.checks, got, tc.want)
			}
		})
	}
}

func TestCanonicalFilePath(t *testing.T) {
	const base = "/home/yolan/files/Webradio"
	cases := []struct {
		name string
		in   string
		base string
		want string
	}{
		{
			name: "real gaende-favorites track",
			in:   "00_❤️ Tracks/01 - Cafius - Vertigo (Original Mix).mp3",
			base: base,
			want: "/home/yolan/files/Webradio/❤️ Tracks/01 - Cafius - Vertigo (Original Mix).mp3",
		},
		{
			name: "bermuda stage with spaces in playlist name",
			in:   "00_BERMUDA - AFTER 6/04 Sweet Dreams (Avicii Sweeder Dreams Mix).mp3",
			base: base,
			want: "/home/yolan/files/Webradio/BERMUDA - AFTER 6/04 Sweet Dreams (Avicii Sweeder Dreams Mix).mp3",
		},
		{
			name: "nested subfolder preserved",
			in:   "00_AMBIANCE/CD 1/01 - Erly Tepshi - Pluvia.mp3",
			base: base,
			want: "/home/yolan/files/Webradio/AMBIANCE/CD 1/01 - Erly Tepshi - Pluvia.mp3",
		},
		{
			name: "no NN_ prefix — playlist has none, leave alone",
			in:   "some-folder/track.mp3",
			base: base,
			want: "/home/yolan/files/Webradio/some-folder/track.mp3",
		},
		{
			name: "empty musicBasePath — return raw (safe fallback)",
			in:   "00_❤️ Tracks/01 - Cafius - Vertigo.mp3",
			base: "",
			want: "00_❤️ Tracks/01 - Cafius - Vertigo.mp3",
		},
		{
			name: "empty input — return empty",
			in:   "",
			base: base,
			want: "",
		},
		{
			name: "no slash at all — return raw",
			in:   "weird.mp3",
			base: base,
			want: "weird.mp3",
		},
		{
			name: "numeric prefix without underscore should NOT be stripped",
			in:   "99abc/file.mp3",
			base: base,
			want: "/home/yolan/files/Webradio/99abc/file.mp3",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := canonicalFilePath(tc.in, tc.base)
			if got != tc.want {
				t.Errorf("canonicalFilePath(%q, %q) = %q, want %q", tc.in, tc.base, got, tc.want)
			}
		})
	}
}

// TestHealthHandler exercises writeHealthResponse end-to-end via httptest.
// Covers both the rollup logic and the HTTP-layer contract: degraded stays 200
// so watchdogs don't thrash on non-critical blips; only hard "down" returns 503.
func TestHealthHandler(t *testing.T) {
	cases := []struct {
		name       string
		checks     map[string]string
		wantStatus string
		wantCode   int
	}{
		{
			name: "all green returns 200 ok",
			checks: map[string]string{
				"mpd": "ok", "postgres": "ok", "plex": "not_used",
				"redis": "not_connected", "meilisearch": "not_connected",
			},
			wantStatus: "ok", wantCode: 200,
		},
		{
			name: "not_used dependencies are neutral",
			checks: map[string]string{
				"mpd": "ok", "postgres": "ok", "plex": "not_used",
				"redis": "not_used", "meilisearch": "not_used",
			},
			wantStatus: "ok", wantCode: 200,
		},
		{
			name: "postgres down returns 503 (critical)",
			checks: map[string]string{
				"mpd": "ok", "postgres": "down", "plex": "ok",
				"redis": "not_connected", "meilisearch": "not_connected",
			},
			wantStatus: "down", wantCode: 503,
		},
		{
			name: "all MPDs down returns 503 (critical)",
			checks: map[string]string{
				"mpd": "down", "postgres": "ok", "plex": "ok",
				"redis": "not_connected", "meilisearch": "not_connected",
			},
			wantStatus: "down", wantCode: 503,
		},
		{
			name: "plex down alone returns 200 degraded (kept for monitoring contract)",
			checks: map[string]string{
				"mpd": "ok", "postgres": "ok", "plex": "down",
				"redis": "not_connected", "meilisearch": "not_connected",
			},
			wantStatus: "degraded", wantCode: 200,
		},
		{
			name: "partial MPD returns 200 degraded (not a hard outage)",
			checks: map[string]string{
				"mpd": "partial (5/9)", "postgres": "ok", "plex": "ok",
				"redis": "not_connected", "meilisearch": "not_connected",
			},
			wantStatus: "degraded", wantCode: 200,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			writeHealthResponse(rec, tc.checks, 9)
			if rec.Code != tc.wantCode {
				t.Errorf("status code = %d, want %d", rec.Code, tc.wantCode)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("response body not JSON: %v", err)
			}
			if got := body["status"]; got != tc.wantStatus {
				t.Errorf("body.status = %v, want %q", got, tc.wantStatus)
			}
			if body["checks"] == nil {
				t.Error("body.checks missing")
			}
			_ = req
		})
	}
}
