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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

	// Sidecars is the aggregate-hash cover over every per-track .mp3.json
	// and .flac.json under the Webradio root. Optional (older manifests
	// predate the field and leave it zero; callers should treat zero as
	// "not included" rather than "empty set").
	Sidecars SidecarAggregate `json:"sidecars"`
}

// SidecarAggregate summarizes the set of sidecars present when the manifest
// was written. AggregateSHA256 is computed over sorted (rel_path\0file_sha\n)
// tuples — see scripts/plex-sync/plex_webradio_sync.py:_sidecar_aggregate_sha256
// for the canonical algorithm. Go's ComputeSidecarAggregate must replicate it
// exactly for verification to work cross-language.
type SidecarAggregate struct {
	Count           int    `json:"count"`
	AggregateSHA256 string `json:"aggregate_sha256"`
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
//
// Does NOT verify sidecars by default (that's 7000+ file hashes per call; too
// expensive for the per-run importer tick). Call VerifySidecars separately
// when stronger guarantees are needed — e.g. from an admin endpoint or a
// canary probe after a crash.
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

// ErrSidecarsFieldMissing is returned by VerifySidecars when the manifest
// predates the sidecars field (AggregateSHA256 == ""). Distinguishable from
// a genuine drift via errors.Is so callers can 404/409 accordingly.
var ErrSidecarsFieldMissing = errors.New("manifest predates sidecars field")

// ErrSidecarDrift is returned by VerifySidecars when count or aggregate
// disagrees with the manifest. Wrapped with context; use errors.Is to detect.
var ErrSidecarDrift = errors.New("sidecar aggregate drift")

// VerifySidecars walks the sidecar tree, recomputes the aggregate hash exactly
// as scripts/plex-sync/sidecar_hash.py:sidecar_aggregate_sha256 does, and
// compares against m.Sidecars.AggregateSHA256. Three error kinds:
//
//   - wraps ErrSidecarsFieldMissing: manifest predates the field
//   - wraps ErrSidecarDrift: hash or count mismatch (the actual drift case)
//   - any other (walk/IO error from ComputeSidecarAggregate): genuine probe
//     failure, caller should surface as 500, not as drift
//
// Use for: admin verification tools, periodic drift audits, post-crash
// sanity checks. DO NOT call from the hot import path — it hashes every
// sidecar file (~7000 at ~1KB each = ~2s wall-clock per run).
func VerifySidecars(webradioDir string, m *Manifest) error {
	if m.Sidecars.AggregateSHA256 == "" {
		return fmt.Errorf("%w (generation %s)", ErrSidecarsFieldMissing, m.GenerationID)
	}
	gotCount, gotAgg, err := ComputeSidecarAggregate(webradioDir)
	if err != nil {
		// Genuine probe failure — NOT wrapped with ErrSidecarDrift.
		return fmt.Errorf("compute sidecar aggregate: %w", err)
	}
	if gotAgg != m.Sidecars.AggregateSHA256 {
		return fmt.Errorf(
			"%w: sha256 manifest=%s actual=%s (count manifest=%d actual=%d)",
			ErrSidecarDrift, m.Sidecars.AggregateSHA256, gotAgg, m.Sidecars.Count, gotCount,
		)
	}
	if gotCount != m.Sidecars.Count {
		// Aggregate matched but count didn't — shouldn't be reachable if the
		// hash algorithm is correct, but surface loudly if it ever happens.
		return fmt.Errorf(
			"%w: count manifest=%d actual=%d despite matching aggregate",
			ErrSidecarDrift, m.Sidecars.Count, gotCount,
		)
	}
	return nil
}

// ComputeSidecarAggregate replicates Python's _sidecar_aggregate_sha256.
//
// MUST stay in lockstep with scripts/plex-sync/plex_webradio_sync.py. The
// specific choices — UTF-8 encoding of paths, NUL separator between path
// and hash, LF between tuples, ASCII-hex digests — are all load-bearing;
// any deviation breaks cross-language verification silently.
func ComputeSidecarAggregate(webradioDir string) (count int, aggregate string, err error) {
	type entry struct {
		rel  string
		hex  string
	}
	var entries []entry

	walkErr := filepath.WalkDir(webradioDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Log-and-continue semantics match the Python walker, which
			// skips unreadable sidecars instead of aborting the whole run.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".mp3.json") && !strings.HasSuffix(name, ".flac.json") {
			return nil
		}
		hex, herr := fileSHA256(path)
		if herr != nil {
			// Vanished-between-walk-and-open — Python skips too.
			return nil
		}
		rel, rerr := filepath.Rel(webradioDir, path)
		if rerr != nil {
			return nil
		}
		entries = append(entries, entry{rel: rel, hex: hex})
		return nil
	})
	if walkErr != nil {
		return 0, "", walkErr
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].rel < entries[j].rel })

	h := sha256.New()
	for _, e := range entries {
		h.Write([]byte(e.rel))
		h.Write([]byte{0})
		h.Write([]byte(e.hex))
		h.Write([]byte{'\n'})
	}
	return len(entries), hex.EncodeToString(h.Sum(nil)), nil
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
