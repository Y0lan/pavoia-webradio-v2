package disk

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeSidecar(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadSidecar_RealShape(t *testing.T) {
	// Minimal real-shape sidecar as emitted by plex_webradio_sync.py after the
	// Phase C5 token-scrub (cover_path not cover_url).
	body := `{
		"track": {
			"title": "Vertigo (Original Mix)",
			"artist": "Cafius",
			"album_artist": "Cafius",
			"album": "Merin EP, Vol. 7",
			"year": 2023,
			"duration_ms": 412345,
			"genres": ["melodic techno"],
			"moods": []
		},
		"artist": {"name": "Cafius", "rating_key": 12345, "thumb_path": "/library/metadata/12345/thumb"},
		"album": {"cover_path": "/library/metadata/67/thumb", "rating_key": 67},
		"file": {"path": null, "original_path": "/home/x/y.mp3"},
		"metadata": {
			"plex_rating_key": 99,
			"plex_guid": "plex://track/99",
			"added_to_webradio": "2025-08-13 12:49:12",
			"updated_at": "2026-04-18 06:49:58"
		}
	}`
	p := writeSidecar(t, t.TempDir(), "track.mp3.json", body)

	sc, err := LoadSidecar(p)
	if err != nil {
		t.Fatalf("LoadSidecar: %v", err)
	}
	if sc.Track.Title != "Vertigo (Original Mix)" {
		t.Errorf("title wrong")
	}
	if sc.PrimaryGenre() != "melodic techno" {
		t.Errorf("primary genre wrong: %q", sc.PrimaryGenre())
	}
	if got := sc.DurationSeconds(); got != 412 {
		t.Errorf("duration wrong: %v (want 412)", got)
	}
	if RatingKeyString(sc.Metadata.PlexRatingKey) != "99" {
		t.Errorf("rating key string wrong: %q", RatingKeyString(sc.Metadata.PlexRatingKey))
	}
}

func TestLoadSidecar_RejectsMissingTitle(t *testing.T) {
	p := writeSidecar(t, t.TempDir(), "bad.mp3.json", `{"track":{"title":""}}`)
	if _, err := LoadSidecar(p); err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestLoadSidecar_MalformedJSON(t *testing.T) {
	p := writeSidecar(t, t.TempDir(), "garbage.mp3.json", `{{ not json`)
	if _, err := LoadSidecar(p); err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestSidecarAddedAt(t *testing.T) {
	cases := []struct {
		name     string
		payload  string
		fallback time.Time
		wantYear int
	}{
		{"rfc3339 UTC (preferred format)", `"2025-08-13T12:49:12Z"`, time.Time{}, 2025},
		{"legacy naive (pre-Phase-D-follow-up)", `"2025-08-13 12:49:12"`, time.Time{}, 2025},
		{"ISO-8601 T-separator with microseconds (observed in old sidecars)", `"2025-08-11T19:33:06.634322"`, time.Time{}, 2025},
		{"ISO-8601 T-separator no tz no fractional", `"2025-08-11T19:33:06"`, time.Time{}, 2025},
		{"malformed → fallback mtime", `"not a date"`, time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), 2024},
		{"empty + zero mtime → now", `""`, time.Time{}, time.Now().UTC().Year()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"track":{"title":"t"},"metadata":{"added_to_webradio":` + tc.payload + `}}`
			p := writeSidecar(t, t.TempDir(), "t.mp3.json", body)
			sc, _ := LoadSidecar(p)
			got := sc.AddedAt(tc.fallback).Year()
			if got != tc.wantYear {
				t.Errorf("year = %d, want %d", got, tc.wantYear)
			}
		})
	}
}

// TestAddedAt_RFC3339_AlwaysUTC guards against a regression where a non-UTC
// host would shift times during parsing. The RFC3339 Z suffix is explicit, so
// the parsed time must come back as UTC regardless of local tz.
func TestAddedAt_RFC3339_AlwaysUTC(t *testing.T) {
	body := `{"track":{"title":"t"},"metadata":{"added_to_webradio":"2025-08-13T12:49:12Z"}}`
	p := writeSidecar(t, t.TempDir(), "t.mp3.json", body)
	sc, _ := LoadSidecar(p)
	got := sc.AddedAt(time.Time{})
	if got.Location() != time.UTC {
		t.Errorf("location = %v, want UTC", got.Location())
	}
	if got.Hour() != 12 {
		t.Errorf("hour = %d, want 12 (UTC)", got.Hour())
	}
}

func TestDurationSeconds_EdgeCases(t *testing.T) {
	mk := func(ms *int) *Sidecar {
		s := &Sidecar{}
		s.Track.DurationMS = ms
		return s
	}
	ptr := func(i int) *int { return &i }

	cases := []struct {
		name string
		in   *int
		want any
	}{
		{"nil", nil, nil},
		{"zero", ptr(0), nil},
		{"sub-second (500ms)", ptr(500), nil},
		{"1s exactly", ptr(1000), 1},
		{"234.5s rounds up to 235", ptr(234500), 235}, // 234500 + 500 = 235000 / 1000 = 235
		{"normal track", ptr(412345), 412},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc := mk(tc.in)
			got := sc.DurationSeconds()
			if got != tc.want {
				t.Errorf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestRatingKeyString(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{"abc", "abc"},
		{nil, ""},
		{12345, "12345"},
		{int64(99), "99"},
		{float64(42), "42"},
		{float64(0), "0"},
	}
	for _, tc := range cases {
		if got := RatingKeyString(tc.in); got != tc.want {
			t.Errorf("RatingKeyString(%#v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
