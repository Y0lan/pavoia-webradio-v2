package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const mbBaseURL = "https://musicbrainz.org/ws/2/"

var mbidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// MBClient calls the MusicBrainz API for artist enrichment.
type MBClient struct {
	http      *http.Client
	userAgent string
}

// NewMBClient creates a MusicBrainz client.
// MusicBrainz requires a descriptive User-Agent per their API terms.
func NewMBClient() *MBClient {
	return &MBClient{
		http:      &http.Client{Timeout: 10 * time.Second},
		userAgent: "GAENDERadio/1.0 (https://github.com/Y0lan/pavoia-webradio-v2)",
	}
}

// MBArtist holds enrichment data from MusicBrainz.
type MBArtist struct {
	MBID    string `json:"mbid"`
	Name    string `json:"name"`
	Country string `json:"country"` // ISO 3166-1 alpha-2
	Area    string `json:"area"`    // human-readable area name
}

// SearchArtist searches MusicBrainz for an artist by name and returns the best match.
func (c *MBClient) SearchArtist(ctx context.Context, name string) (*MBArtist, error) {
	escapedName := strings.ReplaceAll(name, `"`, `\"`)
	params := url.Values{
		"query": {fmt.Sprintf(`artist:"%s"`, escapedName)},
		"fmt":   {"json"},
		"limit": {"1"},
	}

	body, err := c.get(ctx, "artist/", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Artists []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Country string `json:"country"`
			Area struct {
				Name string `json:"name"`
			} `json:"area"`
			Score int `json:"score"`
		} `json:"artists"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("musicbrainz: parse error: %w", err)
	}

	if len(resp.Artists) == 0 {
		return nil, fmt.Errorf("musicbrainz: artist not found: %s", name)
	}

	best := resp.Artists[0]
	// Only accept high-confidence matches (score >= 80)
	if best.Score < 80 {
		return nil, fmt.Errorf("musicbrainz: low confidence match for %s (score=%d)", name, best.Score)
	}

	return &MBArtist{
		MBID:    best.ID,
		Name:    best.Name,
		Country: best.Country,
		Area:    best.Area.Name,
	}, nil
}

// LookupArtist fetches an artist by MBID (more reliable than search).
func (c *MBClient) LookupArtist(ctx context.Context, mbid string) (*MBArtist, error) {
	if !mbidPattern.MatchString(mbid) {
		return nil, fmt.Errorf("musicbrainz: invalid MBID format: %s", mbid)
	}

	params := url.Values{
		"fmt": {"json"},
	}

	body, err := c.get(ctx, "artist/"+mbid, params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Country string `json:"country"`
		Area    struct {
			Name string `json:"name"`
		} `json:"area"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("musicbrainz: parse error: %w", err)
	}

	return &MBArtist{
		MBID:    resp.ID,
		Name:    resp.Name,
		Country: resp.Country,
		Area:    resp.Area.Name,
	}, nil
}

func (c *MBClient) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	reqURL := mbBaseURL + path + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("musicbrainz: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("musicbrainz: rate limited (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("musicbrainz: read body: %w", err)
	}

	return body, nil
}
