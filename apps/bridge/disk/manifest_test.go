package disk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndVerifyManifest_Roundtrip(t *testing.T) {
	dir := t.TempDir()

	// Write a fake artifact and a manifest that correctly hashes it.
	artPath := "artists.json"
	artBytes := []byte(`{"updated_at":"x","total_artists":0,"artists":[]}`)
	if err := os.WriteFile(filepath.Join(dir, artPath), artBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(artBytes)
	m := Manifest{
		GenerationID: "20260418T000000",
		WrittenAt:    "2026-04-18 00:00:00",
		Counts:       map[string]int{"artists": 0},
		Artifacts: map[string]Artifact{
			"artists": {Path: artPath, SHA256: hex.EncodeToString(sum[:]), SizeBytes: int64(len(artBytes))},
		},
	}
	mBytes, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(dir, ManifestFilename), mBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if got.GenerationID != m.GenerationID {
		t.Errorf("generation mismatch")
	}
	if err := VerifyManifest(dir, got); err != nil {
		t.Errorf("VerifyManifest on untouched files: %v", err)
	}
}

func TestVerifyManifest_DetectsTamper(t *testing.T) {
	dir := t.TempDir()
	artPath := "artists.json"
	original := []byte(`{"a":1}`)
	if err := os.WriteFile(filepath.Join(dir, artPath), original, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(original)
	m := &Manifest{
		GenerationID: "gen",
		Artifacts: map[string]Artifact{
			"artists": {Path: artPath, SHA256: hex.EncodeToString(sum[:])},
		},
	}

	// Tamper: overwrite the artifact with different bytes.
	if err := os.WriteFile(filepath.Join(dir, artPath), []byte(`{"a":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	err := VerifyManifest(dir, m)
	if err == nil {
		t.Fatal("expected mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Errorf("expected sha256 mismatch, got %v", err)
	}
}

func TestLoadManifest_RejectsMissingFields(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name string
		body string
	}{
		{"missing generation_id", `{"artifacts":{"a":{"path":"x","sha256":"y"}}}`},
		{"no artifacts", `{"generation_id":"g","artifacts":{}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(dir, ManifestFilename)
			if err := os.WriteFile(p, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadManifest(dir); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestLoadManifest_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for missing manifest, got nil")
	}
}
