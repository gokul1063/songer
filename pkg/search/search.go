package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	innertubeURL = "https://www.youtube.com/youtubei/v1/search"
	innertubeKey = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	userAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

type Video struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Duration string `json:"duration"`
	Channel  string `json:"channel"`
	Views    string `json:"views"`
}

type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) Search(ctx context.Context, query string, limit int) ([]Video, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("empty search query")
	}

	payload, err := json.Marshal(map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"hl":            "en",
				"gl":            "US",
				"clientName":    "WEB",
				"clientVersion": "2.20240101.00.00",
				"clientScreen":  "WATCH",
			},
		},
		"query": query,
	})
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, innertubeURL+"?key="+innertubeKey, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request youtube: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtube returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	videos := parseVideos(body)
	if limit > 0 && len(videos) > limit {
		videos = videos[:limit]
	}
	return videos, nil
}

func parseVideos(raw []byte) []Video {
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil
	}

	var videos []Video
	var walk func(any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			if vr, ok := t["videoRenderer"].(map[string]any); ok {
				videos = append(videos, parseVideoRenderer(vr))
				return
			}
			for _, child := range t {
				walk(child)
			}
		case []any:
			for _, child := range t {
				walk(child)
			}
		}
	}
	walk(data)
	return videos
}

func parseVideoRenderer(vr map[string]any) Video {
	v := Video{
		ID:       str(vr["videoId"]),
		Title:    text(vr, "title"),
		Duration: text(vr, "lengthText"),
		Channel:  text(vr, "ownerText"),
		Views:    text(vr, "viewCountText"),
	}
	if v.ID != "" {
		v.URL = "https://youtu.be/" + v.ID
	}
	return v
}

func text(m map[string]any, key string) string {
	node, ok := m[key]
	if !ok {
		return ""
	}
	switch t := node.(type) {
	case map[string]any:
		if runs, ok := t["runs"].([]any); ok {
			var b strings.Builder
			for _, r := range runs {
				if rm, ok := r.(map[string]any); ok {
					b.WriteString(str(rm["text"]))
				}
			}
			return b.String()
		}
		return str(t["simpleText"])
	case string:
		return t
	}
	return ""
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
