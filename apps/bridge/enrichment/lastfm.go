package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const lastfmBaseURL = "https://ws.audioscrobbler.com/2.0/"

// LastFMClient calls the Last.fm API for artist enrichment.
type LastFMClient struct {
	apiKey string
	http   *http.Client
}

// NewLastFMClient creates a Last.fm client with the given API key.
func NewLastFMClient(apiKey string) *LastFMClient {
	return &LastFMClient{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

// ArtistInfo holds enrichment data from Last.fm artist.getInfo.
type ArtistInfo struct {
	Name    string   `json:"name"`
	MBID    string   `json:"mbid"`
	Bio     string   `json:"bio"`
	Image   string   `json:"image"`   // largest available
	Tags    []string `json:"tags"`
	URL     string   `json:"url"`
}

// SimilarArtist holds a similar artist from Last.fm.
type SimilarArtist struct {
	Name  string  `json:"name"`
	MBID  string  `json:"mbid"`
	Match float64 `json:"match"` // 0.0-1.0 similarity
	Image string  `json:"image"`
}

// GetArtistInfo fetches artist bio, image, tags, and MBID from Last.fm.
func (c *LastFMClient) GetArtistInfo(ctx context.Context, artistName string) (*ArtistInfo, error) {
	params := url.Values{
		"method":  {"artist.getinfo"},
		"artist":  {artistName},
		"api_key": {c.apiKey},
		"format":  {"json"},
	}

	body, err := c.get(ctx, params)
	if err != nil {
		return nil, err
	}

	var resp struct {
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

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("lastfm: parse error: %w", err)
	}
	if resp.Error != 0 {
		return nil, fmt.Errorf("lastfm: API error %d: %s", resp.Error, resp.Message)
	}

	info := &ArtistInfo{
		Name: resp.Artist.Name,
		MBID: resp.Artist.MBID,
		URL:  resp.Artist.URL,
	}

	// Use content (full bio) if available, otherwise summary
	if resp.Artist.Bio.Content != "" {
		info.Bio = resp.Artist.Bio.Content
	} else {
		info.Bio = resp.Artist.Bio.Summary
	}

	// Get largest image
	for _, img := range resp.Artist.Image {
		if img.Text != "" {
			info.Image = img.Text // last one is typically largest
		}
	}

	// Collect tags
	for _, tag := range resp.Artist.Tags.Tag {
		if tag.Name != "" {
			info.Tags = append(info.Tags, tag.Name)
		}
	}

	return info, nil
}

// GetSimilarArtists fetches similar artists from Last.fm.
func (c *LastFMClient) GetSimilarArtists(ctx context.Context, artistName string, limit int) ([]SimilarArtist, error) {
	params := url.Values{
		"method":  {"artist.getsimilar"},
		"artist":  {artistName},
		"api_key": {c.apiKey},
		"format":  {"json"},
		"limit":   {fmt.Sprintf("%d", limit)},
	}

	body, err := c.get(ctx, params)
	if err != nil {
		return nil, err
	}

	var resp struct {
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

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("lastfm: parse error: %w", err)
	}
	if resp.Error != 0 {
		return nil, fmt.Errorf("lastfm: API error %d: %s", resp.Error, resp.Message)
	}

	var result []SimilarArtist
	for _, a := range resp.SimilarArtists.Artist {
		sa := SimilarArtist{
			Name: a.Name,
			MBID: a.MBID,
		}
		// Parse match as float
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

func (c *LastFMClient) get(ctx context.Context, params url.Values) ([]byte, error) {
	reqURL := lastfmBaseURL + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lastfm: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lastfm: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lastfm: read body: %w", err)
	}

	return body, nil
}
