package disk

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Sidecar mirrors the structure plex_webradio_sync.py writes alongside each
// audio symlink as <file>.mp3.json. Only the fields the bridge consumes are
// modeled — extra fields (labels, moods, etc.) are preserved by the writer
// but ignored here.
type Sidecar struct {
	Track struct {
		Title       string   `json:"title"`
		Artist      string   `json:"artist"`
		AlbumArtist string   `json:"album_artist"`
		Album       string   `json:"album"`
		Year        *int     `json:"year"`
		DurationMS  *int     `json:"duration_ms"`
		Genres      []string `json:"genres"`
		TrackNumber *int     `json:"track_number"`
		DiscNumber  *int     `json:"disc_number"`
		// BPM + CamelotKey come from the audio file's ID3 tags (TBPM + TKEY /
		// INITIALKEY), extracted by the Python sync or backfill pass. Both are
		// nullable — tracks without Mixed-In-Key analysis keep them as nil/""
		// rather than zero-value-poisoning the stats aggregates. See
		// scripts/plex-sync/tag_extract.py for the extraction + normalization.
		BPM         *int   `json:"bpm"`
		CamelotKey  string `json:"camelot_key"`
	} `json:"track"`
	Artist struct {
		Name       string `json:"name"`
		RatingKey  any    `json:"rating_key"` // Plex may emit int or string; keep opaque
		ThumbPath  string `json:"thumb_path"`
		ArtPath    string `json:"art_path"`
	} `json:"artist"`
	Album struct {
		CoverPath string `json:"cover_path"`
		ArtPath   string `json:"art_path"`
		RatingKey any    `json:"rating_key"`
	} `json:"album"`
	Metadata struct {
		PlexRatingKey   any    `json:"plex_rating_key"`
		PlexGUID        string `json:"plex_guid"`
		AddedToWebradio string `json:"added_to_webradio"`
		UpdatedAt       string `json:"updated_at"`
	} `json:"metadata"`
}

// LoadSidecar reads a per-track metadata JSON. Returns an error wrapping the
// path so the importer can skip one bad file without failing the whole run.
func LoadSidecar(path string) (*Sidecar, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var s Sidecar
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if s.Track.Title == "" {
		return nil, fmt.Errorf("%s: missing track.title", path)
	}
	return &s, nil
}

// addedAtLayouts covers every timestamp format the Python writer has emitted
// across the project's lifetime. In order of preference:
//
//  1. RFC3339 UTC with Z suffix — what the Phase D follow-up writes.
//  2. ISO-8601 T-separator, fractional seconds, no tz — older save_metadata_json
//     that used datetime.now().isoformat() (naive local; treat as UTC since host
//     is UTC).
//  3. ISO-8601 T-separator, seconds resolution, no tz.
//  4. Space-separated naive "YYYY-MM-DD HH:MM:SS" — oldest format.
//
// Go's time.Parse treats missing-tz layouts as UTC, which matches the Python
// writer's assumption on a UTC host.
var addedAtLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
}

// AddedAt parses the Python-written timestamp. Falls back to the file's mtime
// if every layout fails, then to now() so a sidecar with no date doesn't crash
// the import.
func (s *Sidecar) AddedAt(fallbackMTime time.Time) time.Time {
	if s.Metadata.AddedToWebradio != "" {
		for _, layout := range addedAtLayouts {
			if t, err := time.Parse(layout, s.Metadata.AddedToWebradio); err == nil {
				return t.UTC()
			}
		}
	}
	if !fallbackMTime.IsZero() {
		return fallbackMTime
	}
	return time.Now().UTC()
}

// DurationSeconds rounds duration_ms to an integer second, returning nil if
// the field is absent or non-positive. Matches the main.go parseDurationSec
// contract so SUM(duration_sec) stays honest across both ingest paths.
func (s *Sidecar) DurationSeconds() any {
	if s.Track.DurationMS == nil || *s.Track.DurationMS < 1000 {
		return nil
	}
	return (*s.Track.DurationMS + 500) / 1000 // round-half-up
}

// RatingKeyString normalizes a Plex ratingKey to a string regardless of whether
// Python wrote it as int or string.
func RatingKeyString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%d", int64(x))
	case int:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", x)
	}
}

// PrimaryGenre returns the first Plex genre tag, empty if none.
func (s *Sidecar) PrimaryGenre() string {
	if len(s.Track.Genres) == 0 {
		return ""
	}
	return s.Track.Genres[0]
}
