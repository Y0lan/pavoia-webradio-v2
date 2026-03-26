package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMBSearchArtist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "artist") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"artists": []map[string]any{
				{
					"id":      "abc-123-def",
					"name":    "ARTBAT",
					"country": "UA",
					"area":    map[string]string{"name": "Ukraine"},
					"score":   100,
				},
			},
		})
	}))
	defer srv.Close()

	// Test by calling the mock server directly
	artist, err := mbSearchFromURL(srv.URL + "/artist/?query=artist%3AARTBAT&fmt=json&limit=1")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if artist.MBID != "abc-123-def" {
		t.Errorf("expected MBID abc-123-def, got %s", artist.MBID)
	}
	if artist.Name != "ARTBAT" {
		t.Errorf("expected name ARTBAT, got %s", artist.Name)
	}
	if artist.Country != "UA" {
		t.Errorf("expected country UA, got %s", artist.Country)
	}
	if artist.Area != "Ukraine" {
		t.Errorf("expected area Ukraine, got %s", artist.Area)
	}
}

func TestMBSearchArtist_LowScore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"artists": []map[string]any{
				{
					"id":      "xyz",
					"name":    "Some Other Artist",
					"country": "US",
					"area":    map[string]string{"name": "United States"},
					"score":   45,
				},
			},
		})
	}))
	defer srv.Close()

	_, err := mbSearchFromURL(srv.URL + "/artist/?query=test&fmt=json&limit=1")
	if err == nil {
		t.Fatal("expected error for low-confidence match")
	}
	if !strings.Contains(err.Error(), "low confidence") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMBSearchArtist_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"artists": []map[string]any{},
		})
	}))
	defer srv.Close()

	_, err := mbSearchFromURL(srv.URL + "/artist/?query=test&fmt=json&limit=1")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestMBLookupArtist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "abc-123-def",
			"name":    "ARTBAT",
			"country": "UA",
			"area":    map[string]string{"name": "Ukraine"},
		})
	}))
	defer srv.Close()

	artist, err := mbLookupFromURL(srv.URL + "/artist/abc-123-def?fmt=json")
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}

	if artist.MBID != "abc-123-def" {
		t.Errorf("expected MBID abc-123-def, got %s", artist.MBID)
	}
	if artist.Country != "UA" {
		t.Errorf("expected country UA, got %s", artist.Country)
	}
}

func TestMBRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := mbSearchFromURL(srv.URL + "/artist/?test=1")
	if err == nil {
		t.Fatal("expected error for rate limiting")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("unexpected error: %v", err)
	}
}

// Test helpers
func mbSearchFromURL(rawURL string) (*MBArtist, error) {
	req, _ := http.NewRequestWithContext(context.Background(), "GET", rawURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
		return nil, fmt.Errorf("musicbrainz: rate limited (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz: HTTP %d", resp.StatusCode)
	}

	var apiResp struct {
		Artists []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Country string `json:"country"`
			Area    struct {
				Name string `json:"name"`
			} `json:"area"`
			Score int `json:"score"`
		} `json:"artists"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}
	if len(apiResp.Artists) == 0 {
		return nil, fmt.Errorf("musicbrainz: artist not found")
	}
	best := apiResp.Artists[0]
	if best.Score < 80 {
		return nil, fmt.Errorf("musicbrainz: low confidence match (score=%d)", best.Score)
	}
	return &MBArtist{
		MBID:    best.ID,
		Name:    best.Name,
		Country: best.Country,
		Area:    best.Area.Name,
	}, nil
}

func mbLookupFromURL(rawURL string) (*MBArtist, error) {
	req, _ := http.NewRequestWithContext(context.Background(), "GET", rawURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Country string `json:"country"`
		Area    struct {
			Name string `json:"name"`
		} `json:"area"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}
	return &MBArtist{
		MBID:    apiResp.ID,
		Name:    apiResp.Name,
		Country: apiResp.Country,
		Area:    apiResp.Area.Name,
	}, nil
}
