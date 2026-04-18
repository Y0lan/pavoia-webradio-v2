package main

import "testing"

func TestParseDurationSec(t *testing.T) {
	cases := []struct {
		in   string
		want any
	}{
		{"234.123", 234},
		{"60", 60},
		{"0.5", nil},   // sub-second tracks are meaningless here; store NULL
		{"0", nil},
		{"", nil},
		{"not a number", nil},
		{"-12.0", nil}, // negative durations are noise
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
