package disk

import (
	"encoding/json"
	"fmt"
	"os"
)

// ArtistRecord is the shape plex_webradio_sync.py writes inside artists.json.
// Albums/tracks/playlists are parsed as raw JSON to keep the structure flexible;
// the bridge only uses name, bio, thumb_path, rating_key today.
type ArtistRecord struct {
	Name       string   `json:"name"`
	Bio        *string  `json:"bio"`
	ThumbPath  string   `json:"thumb_path"`
	ArtPath    string   `json:"art_path"`
	RatingKey  any      `json:"rating_key"`
	Genres     []string `json:"genres"`
	Moods      []string `json:"moods"`
	Similar    []string `json:"similar"`
	// Albums, tracks, playlists are present but we don't model them here —
	// the importer derives those relationships from sidecars + filesystem walk.
}

// ArtistsArtifact is the top-level shape of artists.json.
type ArtistsArtifact struct {
	UpdatedAt    string         `json:"updated_at"`
	TotalArtists int            `json:"total_artists"`
	Artists      []ArtistRecord `json:"artists"`
}

// LoadArtists reads and parses artists.json from the Webradio root. Caller
// is expected to have verified the manifest first.
func LoadArtists(path string) (*ArtistsArtifact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read artists.json: %w", err)
	}
	var a ArtistsArtifact
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("decode artists.json: %w", err)
	}
	return &a, nil
}
