package disk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadArtists_RealShape(t *testing.T) {
	body := `{
		"updated_at": "2026-04-18 06:49:58",
		"total_artists": 2,
		"artists": [
			{
				"name": "Björk",
				"bio": "Icelandic singer.",
				"thumb_path": "/library/metadata/1/thumb",
				"rating_key": 1,
				"genres": ["electronica"],
				"moods": ["experimental"],
				"similar": ["aphex twin"]
			},
			{
				"name": "Bjork",
				"genres": ["electronica"]
			}
		]
	}`
	p := filepath.Join(t.TempDir(), "artists.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	a, err := LoadArtists(p)
	if err != nil {
		t.Fatalf("LoadArtists: %v", err)
	}
	if a.TotalArtists != 2 {
		t.Errorf("total wrong")
	}
	if len(a.Artists) != 2 {
		t.Errorf("artist count wrong")
	}
	if a.Artists[0].Name != "Björk" {
		t.Errorf("name wrong")
	}
	if a.Artists[0].Bio == nil || *a.Artists[0].Bio != "Icelandic singer." {
		t.Errorf("bio wrong")
	}
}

func TestMergeTagSlices(t *testing.T) {
	cases := []struct {
		name string
		a, b []string
		want []string
	}{
		{"empty", nil, nil, []string{}},
		{"no overlap", []string{"techno"}, []string{"dark"}, []string{"techno", "dark"}},
		{"case-insensitive dedup", []string{"Techno"}, []string{"techno"}, []string{"Techno"}},
		{"trim whitespace", []string{"  techno "}, []string{"techno"}, []string{"  techno "}},
		{"drop empties", []string{"", "techno"}, []string{""}, []string{"techno"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeTagSlices(tc.a, tc.b)
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d (%v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
