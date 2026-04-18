package disk

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// writeSidecarFile writes body to webradioDir/relPath, creating parents.
func writeSidecarFile(t *testing.T, webradioDir, relPath, body string) {
	t.Helper()
	full := filepath.Join(webradioDir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// expected computes what the aggregate SHOULD be for a known set, using the
// same algorithm as Python's _sidecar_aggregate_sha256. Keeping this inline
// (rather than calling ComputeSidecarAggregate) so the test can't drift
// silently with an algorithmic change.
func expectedAggregate(files map[string]string) string {
	type pair struct{ rel, sum string }
	var pairs []pair
	for rel, body := range files {
		h := sha256.Sum256([]byte(body))
		pairs = append(pairs, pair{rel: rel, sum: hex.EncodeToString(h[:])})
	}
	// sort pairs by rel
	for i := 1; i < len(pairs); i++ {
		for j := i; j > 0 && pairs[j-1].rel > pairs[j].rel; j-- {
			pairs[j-1], pairs[j] = pairs[j], pairs[j-1]
		}
	}
	agg := sha256.New()
	for _, p := range pairs {
		agg.Write([]byte(p.rel))
		agg.Write([]byte{0})
		agg.Write([]byte(p.sum))
		agg.Write([]byte{'\n'})
	}
	return hex.EncodeToString(agg.Sum(nil))
}

func TestComputeSidecarAggregate_Empty(t *testing.T) {
	dir := t.TempDir()
	count, agg, err := ComputeSidecarAggregate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
	// Empty input = empty hash input = constant sha256 of empty string.
	emptyHash := sha256.Sum256(nil)
	if agg != hex.EncodeToString(emptyHash[:]) {
		t.Errorf("empty aggregate = %s, want sha256('')", agg)
	}
}

func TestComputeSidecarAggregate_MatchesExpected(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"AMBIANCE/01 - Foo.mp3.json":          `{"track":{"title":"Foo"}}`,
		"AMBIANCE/02 - Bar.mp3.json":          `{"track":{"title":"Bar"}}`,
		"PALAC - DANCE/z track.mp3.json":      `{"track":{"title":"Z"}}`,
		"closing/something.flac.json":         `{"track":{"title":"S"}}`,
	}
	// Also write an MP3 audio file and a non-sidecar JSON — these must be ignored.
	for rel, body := range files {
		writeSidecarFile(t, dir, rel, body)
	}
	writeSidecarFile(t, dir, "AMBIANCE/01 - Foo.mp3", "fake-audio")
	writeSidecarFile(t, dir, "artists.json", `{"artists":[]}`)

	count, agg, err := ComputeSidecarAggregate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 4 {
		t.Errorf("count = %d, want 4", count)
	}
	want := expectedAggregate(files)
	if agg != want {
		t.Errorf("aggregate mismatch\n  got:  %s\n  want: %s", agg, want)
	}
}

func TestVerifySidecars_DetectsMutation(t *testing.T) {
	dir := t.TempDir()
	writeSidecarFile(t, dir, "s/a.mp3.json", `{"v":1}`)
	writeSidecarFile(t, dir, "s/b.mp3.json", `{"v":2}`)
	count, agg, _ := ComputeSidecarAggregate(dir)
	m := &Manifest{
		GenerationID: "g",
		Sidecars:     SidecarAggregate{Count: count, AggregateSHA256: agg},
	}
	if err := VerifySidecars(dir, m); err != nil {
		t.Fatalf("unmutated should pass: %v", err)
	}
	// Mutate one sidecar.
	writeSidecarFile(t, dir, "s/a.mp3.json", `{"v":1,"added":true}`)
	err := VerifySidecars(dir, m)
	if err == nil {
		t.Fatal("expected verification error after mutation")
	}
	if !errors.Is(err, ErrSidecarDrift) {
		t.Errorf("expected ErrSidecarDrift, got: %v", err)
	}
}

func TestVerifySidecars_DetectsAdd(t *testing.T) {
	dir := t.TempDir()
	writeSidecarFile(t, dir, "s/a.mp3.json", `{"v":1}`)
	count, agg, _ := ComputeSidecarAggregate(dir)
	m := &Manifest{Sidecars: SidecarAggregate{Count: count, AggregateSHA256: agg}}
	// Add a new sidecar after manifest was "written".
	writeSidecarFile(t, dir, "s/b.mp3.json", `{"v":2}`)
	if err := VerifySidecars(dir, m); err == nil {
		t.Fatal("expected verification error after adding a sidecar")
	}
}

func TestVerifySidecars_DetectsRemove(t *testing.T) {
	dir := t.TempDir()
	writeSidecarFile(t, dir, "s/a.mp3.json", `{"v":1}`)
	writeSidecarFile(t, dir, "s/b.mp3.json", `{"v":2}`)
	count, agg, _ := ComputeSidecarAggregate(dir)
	m := &Manifest{Sidecars: SidecarAggregate{Count: count, AggregateSHA256: agg}}
	// Remove one after manifest written.
	os.Remove(filepath.Join(dir, "s/a.mp3.json"))
	if err := VerifySidecars(dir, m); err == nil {
		t.Fatal("expected verification error after removing a sidecar")
	}
}

// TestVerifySidecars_RejectsEmptyField guards against accidental verification
// against older manifests that don't carry the sidecars field. Silent pass
// in that case would claim coverage we don't have.
func TestVerifySidecars_RejectsEmptyField(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{GenerationID: "old", Sidecars: SidecarAggregate{}}
	err := VerifySidecars(dir, m)
	if err == nil {
		t.Fatal("expected error when sidecars field is empty (older manifest)")
	}
	if !errors.Is(err, ErrSidecarsFieldMissing) {
		t.Errorf("expected ErrSidecarsFieldMissing, got: %v", err)
	}
}

// TestSidecarAggregate_CrossLanguage actually proves the Go implementation
// matches Python's — not just matches another Go re-implementation. Skips if
// python3 isn't on PATH (Python is required at runtime for the sync script
// anyway, so it's a safe dep to rely on in this test).
//
// The Python inline script reproduces _sidecar_aggregate_sha256 from
// scripts/plex-sync/plex_webradio_sync.py verbatim — any algorithmic drift
// between the two implementations fails this test.
func TestSidecarAggregate_CrossLanguage(t *testing.T) {
	if _, err := os.Stat("/usr/bin/python3"); err != nil {
		if _, err := os.Stat("/usr/local/bin/python3"); err != nil {
			t.Skip("python3 not available on PATH; skipping cross-language check")
		}
	}

	dir := t.TempDir()
	// Include a variety: nested paths, unicode, whitespace, non-ASCII. These
	// are real playlist-name conditions on the live Whatbox host (`❤️ Tracks`,
	// `PALAC - DANCE`, etc.).
	files := map[string]string{
		"❤️ Tracks/01 - Foo.mp3.json":    `{"track":{"title":"Foo"}}`,
		"PALAC - DANCE/02. Bar.mp3.json": `{"track":{"title":"Bar"}}`,
		"ETAGE 0/z.mp3.json":             `{"track":{"title":"Z"}}`,
		"Outro/audio.flac.json":          `{"track":{"title":"A"}}`,
	}
	for rel, body := range files {
		writeSidecarFile(t, dir, rel, body)
	}

	// Reference hash via the ACTUAL production implementation in
	// scripts/plex-sync/sidecar_hash.py — not an inline duplicate. If that
	// module's algorithm ever changes, this test breaks until the Go side
	// is brought back into lockstep.
	//
	// Find the repo root by walking up from this file's directory.
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	sidecarHashDir := filepath.Join(repoRoot, "scripts", "plex-sync")
	if _, err := os.Stat(filepath.Join(sidecarHashDir, "sidecar_hash.py")); err != nil {
		t.Skipf("sidecar_hash.py not found at %s — repo layout changed?", sidecarHashDir)
	}
	pyScript := `
import sys
sys.path.insert(0, sys.argv[1])
from sidecar_hash import sidecar_aggregate_sha256
count, agg = sidecar_aggregate_sha256(sys.argv[2])
print(count, agg)
`
	cmd := exec.Command("python3", "-c", pyScript, sidecarHashDir, dir)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python3 reference failed: %v", err)
	}
	var pyCount int
	var pyAgg string
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%d %s", &pyCount, &pyAgg); err != nil {
		t.Fatalf("parse python output %q: %v", out, err)
	}

	goCount, goAgg, err := ComputeSidecarAggregate(dir)
	if err != nil {
		t.Fatalf("Go compute failed: %v", err)
	}

	if goCount != pyCount {
		t.Errorf("count mismatch: go=%d python=%d", goCount, pyCount)
	}
	if goAgg != pyAgg {
		t.Errorf("hash mismatch:\n  go:     %s\n  python: %s", goAgg, pyAgg)
	}
}
