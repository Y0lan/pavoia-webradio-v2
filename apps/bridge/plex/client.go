package plex

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Client talks to the Plex API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// Track represents a track from a Plex playlist.
type Track struct {
	RatingKey   string    `json:"ratingKey"`
	Title       string    `json:"title"`
	Artist      string    `json:"grandparentTitle"`
	Album       string    `json:"parentTitle"`
	Duration    int       `json:"duration"` // milliseconds
	AddedAt     int64     `json:"addedAt"` // unix timestamp
	Year        int       `json:"year"`
	FilePath    string    // extracted from media parts
	Genre       string    // extracted from genre tags
	Format      string    // extracted from media parts
}

// Playlist represents a Plex playlist.
type Playlist struct {
	RatingKey string `json:"ratingKey"`
	Title     string `json:"title"`
	Type      string `json:"type"`
	Duration  int    `json:"duration"`
	LeafCount int    `json:"leafCount"`
}

// NewClient creates a Plex API client.
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Plex uses self-signed certs
			},
		},
	}
}

// Playlists returns all playlists.
func (c *Client) Playlists() ([]Playlist, error) {
	var resp struct {
		MediaContainer struct {
			Metadata []Playlist `json:"Metadata"`
		} `json:"MediaContainer"`
	}
	if err := c.get("/playlists", &resp); err != nil {
		return nil, err
	}
	return resp.MediaContainer.Metadata, nil
}

// PlaylistTracks returns all tracks in a playlist.
func (c *Client) PlaylistTracks(ratingKey string) ([]Track, error) {
	var resp struct {
		MediaContainer struct {
			Metadata []json.RawMessage `json:"Metadata"`
		} `json:"MediaContainer"`
	}
	if err := c.get(fmt.Sprintf("/playlists/%s/items", ratingKey), &resp); err != nil {
		return nil, err
	}

	tracks := make([]Track, 0, len(resp.MediaContainer.Metadata))
	for _, raw := range resp.MediaContainer.Metadata {
		var item struct {
			RatingKey        string `json:"ratingKey"`
			Title            string `json:"title"`
			GrandparentTitle string `json:"grandparentTitle"` // artist
			ParentTitle      string `json:"parentTitle"`      // album
			Duration         int    `json:"duration"`
			AddedAt          int64  `json:"addedAt"`
			Year             int    `json:"year"`
			Media            []struct {
				Container string `json:"container"`
				Part      []struct {
					File string `json:"file"`
				} `json:"Part"`
			} `json:"Media"`
			Genre []struct {
				Tag string `json:"tag"`
			} `json:"Genre"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			slog.Warn("plex: skipping track with unmarshal error", "error", err)
			continue
		}

		t := Track{
			RatingKey: item.RatingKey,
			Title:     item.Title,
			Artist:    item.GrandparentTitle,
			Album:     item.ParentTitle,
			Duration:  item.Duration,
			AddedAt:   item.AddedAt,
			Year:      item.Year,
		}

		if len(item.Media) > 0 {
			t.Format = item.Media[0].Container
			if len(item.Media[0].Part) > 0 {
				t.FilePath = item.Media[0].Part[0].File
			}
		}
		if len(item.Genre) > 0 {
			t.Genre = item.Genre[0].Tag
		}

		tracks = append(tracks, t)
	}

	return tracks, nil
}

// Healthy checks if the Plex server is reachable (uses the client's default 30s timeout).
func (c *Client) Healthy() bool {
	return c.HealthyWithTimeout(0)
}

// HealthyWithTimeout checks if Plex is reachable with a custom timeout.
// Pass 0 to use the client's default timeout.
func (c *Client) HealthyWithTimeout(timeout time.Duration) bool {
	req, err := http.NewRequest("GET", c.baseURL+"/identity", nil)
	if err != nil {
		return false
	}
	req.Header.Set("X-Plex-Token", c.token)

	client := c.http
	if timeout > 0 {
		client = &http.Client{
			Timeout:   timeout,
			Transport: c.http.Transport,
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (c *Client) get(path string, result any) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Plex-Token", c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("plex %s returned %d: %s", path, resp.StatusCode, string(body))
	}

	// Limit response body to 50MB to prevent memory exhaustion
	limited := io.LimitReader(resp.Body, 50*1024*1024)
	if err := json.NewDecoder(limited).Decode(result); err != nil {
		return fmt.Errorf("decode response from %s: %w", path, err)
	}

	return nil
}
