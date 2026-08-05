package autoplay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"songer/pkg/search"
)

const (
	innertubeNextURL = "https://www.youtube.com/youtubei/v1/next"
	innertubeKey     = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	userAgent        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

type Client struct {
	http     *http.Client
	endpoint string
}

func NewClient() *Client {
	return &Client{
		http:     &http.Client{Timeout: 20 * time.Second},
		endpoint: innertubeNextURL,
	}
}

// Suggest returns the related/up-next videos for a video ID, in YouTube's order.
func (c *Client) Suggest(ctx context.Context, videoID string, limit int) ([]search.Video, error) {
	if strings.TrimSpace(videoID) == "" {
		return nil, fmt.Errorf("empty video id")
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
		"videoId": videoID,
	})
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"?key="+innertubeKey, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request related: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtube returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	videos := parseRelated(body)
	if limit > 0 && len(videos) > limit {
		videos = videos[:limit]
	}
	return videos, nil
}

// BuildQueue grows a queue of suggested videos level by level:
// first the top `perNode` suggestions of the seed, then the top `perNode`
// suggestions of each of those, and so on down `depth` levels.
// With perNode=2, depth=2 that is 2 + 4 = 6 videos.
func (c *Client) BuildQueue(ctx context.Context, seedID string, perNode, depth int) ([]search.Video, error) {
	if perNode <= 0 {
		perNode = 2
	}
	if depth <= 0 {
		depth = 2
	}

	seen := map[string]bool{seedID: true}
	var queue []search.Video

	var expand func(id string, level int) error
	expand = func(id string, level int) error {
		if level >= depth {
			return nil
		}
		vids, err := c.Suggest(ctx, id, perNode*3)
		if err != nil {
			return err
		}
		var picked []search.Video
		for _, v := range vids {
			if len(picked) >= perNode {
				break
			}
			if v.ID == "" || seen[v.ID] {
				continue
			}
			seen[v.ID] = true
			picked = append(picked, v)
		}
		queue = append(queue, picked...)
		for _, v := range picked {
			if err := expand(v.ID, level+1); err != nil {
				return err
			}
		}
		return nil
	}

	if err := expand(seedID, 0); err != nil {
		return nil, err
	}
	return queue, nil
}

func parseRelated(raw []byte) []search.Video {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil
	}

	var results []any
	if contents, ok := root["contents"].(map[string]any); ok {
		if twin, ok := contents["twoColumnWatchNextResults"].(map[string]any); ok {
			if sec, ok := twin["secondaryResults"].(map[string]any); ok {
				if sr, ok := sec["secondaryResults"].(map[string]any); ok {
					results, _ = sr["results"].([]any)
				}
			}
		}
	}

	var videos []search.Video
	for _, item := range results {
		im, ok := item.(map[string]any)
		if !ok {
			continue
		}
		vm, ok := im["lockupViewModel"].(map[string]any)
		if !ok {
			continue
		}
		if vm["contentType"] != "LOCKUP_CONTENT_TYPE_VIDEO" {
			continue
		}
		v := parseLockup(vm)
		if v.ID == "" {
			continue
		}
		videos = append(videos, v)
	}
	return videos
}

func parseLockup(vm map[string]any) search.Video {
	v := search.Video{ID: str(vm["contentId"])}
	if md, ok := vm["metadata"].(map[string]any); ok {
		if lmd, ok := md["lockupMetadataViewModel"].(map[string]any); ok {
			v.Title = contentString(lmd, "title", "content")
			if meta, ok := lmd["metadata"].(map[string]any); ok {
				if cmv, ok := meta["contentMetadataViewModel"].(map[string]any); ok {
					if rows, ok := cmv["metadataRows"].([]any); ok && len(rows) > 0 {
						v.Channel = firstPartText(rows[0])
						if len(rows) > 1 {
							v.Views = firstPartText(rows[1])
						}
					}
				}
			}
		}
	}
	if v.ID != "" {
		v.URL = "https://youtu.be/" + v.ID
	}
	return v
}

func firstPartText(row any) string {
	rm, ok := row.(map[string]any)
	if !ok {
		return ""
	}
	parts, ok := rm["metadataParts"].([]any)
	if !ok || len(parts) == 0 {
		return ""
	}
	return contentString(parts[0], "text", "content")
}

func contentString(m any, path ...string) string {
	cur := m
	for _, p := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = mm[p]
	}
	if s, ok := cur.(string); ok {
		return s
	}
	return ""
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
