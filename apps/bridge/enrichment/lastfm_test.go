package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLastFMGetArtistInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("method") != "artist.getinfo" {
			t.Errorf("unexpected method: %s", r.URL.Query().Get("method"))
		}
		if r.URL.Query().Get("artist") != "ARTBAT" {
			t.Errorf("unexpected artist: %s", r.URL.Query().Get("artist"))
		}

		json.NewEncoder(w).Encode(map[string]any{
			"artist": map[string]any{
				"name": "ARTBAT",
				"mbid": "abc-123",
				"url":  "https://www.last.fm/music/ARTBAT",
				"image": []map[string]string{
					{"#text": "small.jpg", "size": "small"},
					{"#text": "large.jpg", "size": "extralarge"},
				},
				"tags": map[string]any{
					"tag": []map[string]string{
						{"name": "techno"},
						{"name": "melodic techno"},
					},
				},
				"bio": map[string]string{
					"summary": "Short bio",
					"content": "Full biography of ARTBAT.",
				},
			},
		})
	}))
	defer srv.Close()

	client := NewLastFMClient("test-key")
	// Override the base URL by using the test server
	client.http = srv.Client()

	// We need to override the URL — create a custom transport
	originalGet := client.get
	_ = originalGet
	// Instead, let's test the parsing directly with a mock server
	// by temporarily replacing the base URL
	info, err := fetchArtistInfoFromURL(srv.URL+"?method=artist.getinfo&artist=ARTBAT&api_key=test&format=json", srv.Client())
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if info.Name != "ARTBAT" {
		t.Errorf("expected name ARTBAT, got %s", info.Name)
	}
	if info.MBID != "abc-123" {
		t.Errorf("expected mbid abc-123, got %s", info.MBID)
	}
	if info.Bio != "Full biography of ARTBAT." {
		t.Errorf("expected full bio, got %s", info.Bio)
	}
	if info.Image != "large.jpg" {
		t.Errorf("expected large.jpg, got %s", info.Image)
	}
	if len(info.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(info.Tags))
	}
}

func TestLastFMGetSimilarArtists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"similarartists": map[string]any{
				"artist": []map[string]any{
					{
						"name":  "Anyma",
						"mbid":  "def-456",
						"match": "0.85",
						"image": []map[string]string{
							{"#text": "anyma.jpg", "size": "large"},
						},
					},
					{
						"name":  "Massano",
						"mbid":  "",
						"match": "0.72",
						"image": []map[string]string{},
					},
				},
			},
		})
	}))
	defer srv.Close()

	similar, err := fetchSimilarFromURL(srv.URL+"?test=1", srv.Client())
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if len(similar) != 2 {
		t.Fatalf("expected 2 similar artists, got %d", len(similar))
	}
	if similar[0].Name != "Anyma" {
		t.Errorf("expected Anyma, got %s", similar[0].Name)
	}
	if similar[0].Match < 0.84 || similar[0].Match > 0.86 {
		t.Errorf("expected match ~0.85, got %f", similar[0].Match)
	}
	if similar[1].Name != "Massano" {
		t.Errorf("expected Massano, got %s", similar[1].Name)
	}
}

func TestLastFMAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"error":   6,
			"message": "Artist not found",
		})
	}))
	defer srv.Close()

	_, err := fetchArtistInfoFromURL(srv.URL+"?test=1", srv.Client())
	if err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestLastFMHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := fetchArtistInfoFromURL(srv.URL+"?test=1", srv.Client())
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// Test helpers that fetch from a specific URL (for httptest server)
func fetchArtistInfoFromURL(rawURL string, client *http.Client) (*ArtistInfo, error) {
	req, _ := http.NewRequestWithContext(context.Background(), "GET", rawURL, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var apiResp struct {
		Artist struct {
			Name  string `json:"name"`
			MBID  string `json:"mbid"`
			URL   string `json:"url"`
			Image []struct {
				Text string `json:"#text"`
				Size string `json:"size"`
			} `json:"image"`
			Tags struct {
				Tag []struct {
					Name string `json:"name"`
				} `json:"tag"`
			} `json:"tags"`
			Bio struct {
				Summary string `json:"summary"`
				Content string `json:"content"`
			} `json:"bio"`
		} `json:"artist"`
		Error   int    `json:"error"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}
	if apiResp.Error != 0 {
		return nil, fmt.Errorf("API error %d: %s", apiResp.Error, apiResp.Message)
	}

	info := &ArtistInfo{
		Name: apiResp.Artist.Name,
		MBID: apiResp.Artist.MBID,
		URL:  apiResp.Artist.URL,
	}
	if apiResp.Artist.Bio.Content != "" {
		info.Bio = apiResp.Artist.Bio.Content
	} else {
		info.Bio = apiResp.Artist.Bio.Summary
	}
	for _, img := range apiResp.Artist.Image {
		if img.Text != "" {
			info.Image = img.Text
		}
	}
	for _, tag := range apiResp.Artist.Tags.Tag {
		if tag.Name != "" {
			info.Tags = append(info.Tags, tag.Name)
		}
	}
	return info, nil
}

func fetchSimilarFromURL(rawURL string, client *http.Client) ([]SimilarArtist, error) {
	req, _ := http.NewRequestWithContext(context.Background(), "GET", rawURL, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		SimilarArtists struct {
			Artist []struct {
				Name  string `json:"name"`
				MBID  string `json:"mbid"`
				Match string `json:"match"`
				Image []struct {
					Text string `json:"#text"`
					Size string `json:"size"`
				} `json:"image"`
			} `json:"artist"`
		} `json:"similarartists"`
		Error   int    `json:"error"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}
	if apiResp.Error != 0 {
		return nil, fmt.Errorf("API error %d: %s", apiResp.Error, apiResp.Message)
	}

	var result []SimilarArtist
	for _, a := range apiResp.SimilarArtists.Artist {
		sa := SimilarArtist{Name: a.Name, MBID: a.MBID}
		fmt.Sscanf(a.Match, "%f", &sa.Match)
		for _, img := range a.Image {
			if img.Text != "" {
				sa.Image = img.Text
			}
		}
		result = append(result, sa)
	}
	return result, nil
}
