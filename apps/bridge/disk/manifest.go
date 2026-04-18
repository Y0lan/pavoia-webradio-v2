// Package disk is the authoritative ingest path for library metadata.
//
// It reads the JSON artifacts the Python Plex sync (scripts/plex-sync/) writes
// to $WEBRADIO_FOLDER, verifies them against the sync_manifest.json sha256
// checksums, and upserts into Postgres. Replaces the old apps/bridge/plex
// client role — the bridge no longer authenticates against Plex directly.
//
// Contract (must match scripts/plex-sync/plex_webradio_sync.py):
//
//   - sync_manifest.json is written LAST, atomically, and references the
//     other four artifacts by sha256. A run that sees no manifest, a stale
//     manifest, or a sha256 mismatch rejects the whole batch and retries.
//   - artists.json, albums.json, playlists.json, tracks_index.json — each
//     written atomically via tempfile + os.replace before the manifest.
//   - Per-track <file>.mp3.json sidecars — written under the playlist folder,
//     carry the metadata fields the importer reads to build library_tracks rows.
package disk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ManifestFilename is the artifact Python writes last — the authority.
const ManifestFilename = "sync_manifest.json"

// Artifact describes one top-level JSON file hashed in the manifest.
type Artifact struct {
	Path      string `json:"path"`       // relative to WEBRADIO_FOLDER
	SHA256    string `json:"sha256"`     // hex-lowercase
	SizeBytes int64  `json:"size_bytes"`
	MTime     string `json:"mtime"`      // informational
}

// Manifest is the generation descriptor a reader must verify before trusting
// any of the top-level artifacts.
type Manifest struct {
	GenerationID string              `json:"generation_id"`
	WrittenAt    string              `json:"written_at"`
	Counts       map[string]int      `json:"counts"`
	Artifacts    map[string]Artifact `json:"artifacts"`
}

// LoadManifest reads and unmarshals sync_manifest.json from the Webradio root.
// Returns an error the caller can distinguish from verification errors.
func LoadManifest(webradioDir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(webradioDir, ManifestFilename))
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	if m.GenerationID == "" {
		return nil, fmt.Errorf("manifest missing generation_id")
	}
	if len(m.Artifacts) == 0 {
		return nil, fmt.Errorf("manifest has no artifacts")
	}
	return &m, nil
}

// VerifyManifest checks every artifact's sha256 against its current on-disk
// bytes. Any mismatch means the writer crashed mid-generation or another
// process truncated a file since the manifest was written — reject the batch.
func VerifyManifest(webradioDir string, m *Manifest) error {
	for name, a := range m.Artifacts {
		full := filepath.Join(webradioDir, a.Path)
		gotSum, err := fileSHA256(full)
		if err != nil {
			return fmt.Errorf("hash %s: %w", name, err)
		}
		if gotSum != a.SHA256 {
			return fmt.Errorf(
				"artifact %q sha256 mismatch: manifest=%s actual=%s",
				name, a.SHA256, gotSum,
			)
		}
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
