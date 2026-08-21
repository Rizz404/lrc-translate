// Package lrclib is a client for the public LRCLIB API (https://lrclib.net/api),
// used to search for songs and fetch their synced lyrics.
package lrclib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client talks to the LRCLIB API.
type Client struct {
	baseURL string
	http    *http.Client
}

// New creates a Client rooted at baseURL (e.g. "https://lrclib.net/api").
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Track is one candidate result from LRCLIB, as returned by /search and /get.
type Track struct {
	ID            int64   `json:"id"`
	TrackName     string  `json:"trackName"`
	ArtistName    string  `json:"artistName"`
	AlbumName     string  `json:"albumName"`
	Duration      float64 `json:"duration"` // seconds
	Instrumental  bool    `json:"instrumental"`
	PlainLyrics   string  `json:"plainLyrics"`
	SyncedLyrics  string  `json:"syncedLyrics"`
}

// Search queries LRCLIB by track title and artist name, returning candidate
// tracks. Results may be ambiguous (multiple versions/remixes/live takes) —
// the caller decides how to disambiguate.
func (c *Client) Search(ctx context.Context, title, artist string) ([]Track, error) {
	q := url.Values{}
	if title != "" {
		q.Set("track_name", title)
	}
	if artist != "" {
		q.Set("artist_name", artist)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/search?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lrclib search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lrclib search: unexpected status %d", resp.StatusCode)
	}

	var tracks []Track
	if err := json.NewDecoder(resp.Body).Decode(&tracks); err != nil {
		return nil, fmt.Errorf("lrclib search: decode response: %w", err)
	}
	return tracks, nil
}

// GetByID fetches a single track's full details (including synced lyrics) by
// its LRCLIB ID.
func (c *Client) GetByID(ctx context.Context, id int64) (*Track, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/get/%d", c.baseURL, id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lrclib get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lrclib get: unexpected status %d", resp.StatusCode)
	}

	var track Track
	if err := json.NewDecoder(resp.Body).Decode(&track); err != nil {
		return nil, fmt.Errorf("lrclib get: decode response: %w", err)
	}
	return &track, nil
}
