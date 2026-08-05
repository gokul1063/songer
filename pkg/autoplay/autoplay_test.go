package autoplay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func lockup(id, title string) map[string]any {
	return map[string]any{
		"lockupViewModel": map[string]any{
			"contentId":   id,
			"contentType": "LOCKUP_CONTENT_TYPE_VIDEO",
			"metadata": map[string]any{
				"lockupMetadataViewModel": map[string]any{
					"title": map[string]any{"content": title},
					"metadata": map[string]any{
						"contentMetadataViewModel": map[string]any{
							"metadataRows": []any{
								map[string]any{"metadataParts": []any{map[string]any{"text": map[string]any{"content": "ChannelX"}}}},
								map[string]any{"metadataParts": []any{map[string]any{"text": map[string]any{"content": "100 views"}}}},
							},
						},
					},
				},
			},
		},
	}
}

func nextDoc(videoID string, suggestions ...string) string {
	results := []any{}
	for _, s := range suggestions {
		results = append(results, lockup(s, s+" title"))
	}
	doc := map[string]any{
		"contents": map[string]any{
			"twoColumnWatchNextResults": map[string]any{
				"secondaryResults": map[string]any{
					"secondaryResults": map[string]any{"results": results},
				},
			},
		},
	}
	b, _ := json.Marshal(doc)
	return string(b)
}

func TestParseRelated(t *testing.T) {
	raw := []byte(`{
		"contents": {
			"twoColumnWatchNextResults": {
				"secondaryResults": {
					"secondaryResults": {
						"results": [
							{"lockupViewModel": {"contentId": "AAA", "contentType": "LOCKUP_CONTENT_TYPE_VIDEO",
								"metadata": {"lockupMetadataViewModel": {"title": {"content": "Track One"},
									"metadata": {"contentMetadataViewModel": {"metadataRows": [
										{"metadataParts": [{"text": {"content": "Artist A"}}]},
										{"metadataParts": [{"text": {"content": "42K views"}}]}
									]}}}}}},
							{"reelShelfRenderer": {"title": "shorts"}},
							{"lockupViewModel": {"contentId": "BBB", "contentType": "LOCKUP_CONTENT_TYPE_PLAYLIST"}}
						]
					}
				}
			}
		}
	}`)

	videos := parseRelated(raw)
	if len(videos) != 1 {
		t.Fatalf("expected 1 video, got %d", len(videos))
	}
	v := videos[0]
	if v.ID != "AAA" || v.Title != "Track One" || v.Channel != "Artist A" || v.Views != "42K views" {
		t.Fatalf("unexpected video: %+v", v)
	}
	if v.URL != "https://youtu.be/AAA" {
		t.Fatalf("bad URL: %s", v.URL)
	}
}

func TestBuildQueueOrderAndDedupe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			VideoID string `json:"videoId"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		var resp string
		switch body.VideoID {
		case "seed":
			resp = nextDoc("seed", "A1", "A1", "A2")
		case "A1":
			resp = nextDoc("A1", "B1", "B2")
		case "A2":
			resp = nextDoc("A2", "C1", "C2")
		default:
			resp = nextDoc(body.VideoID)
		}
		w.Write([]byte(resp))
	}))
	defer srv.Close()

	c := NewClient()
	c.http = srv.Client()
	c.endpoint = srv.URL

	queue, err := c.BuildQueue(context.Background(), "seed", 2, 2)
	if err != nil {
		t.Fatal(err)
	}

	got := []string{}
	for _, v := range queue {
		got = append(got, v.ID)
	}
	want := []string{"A1", "A2", "B1", "B2", "C1", "C2"}
	if !slices.Equal(got, want) {
		t.Fatalf("queue = %v, want %v", got, want)
	}

	if len(queue) != 6 {
		t.Fatalf("expected 6 videos, got %d", len(queue))
	}
}

func TestBuildQueueInvalidInput(t *testing.T) {
	c := NewClient()
	_, err := c.Suggest(context.Background(), "", 5)
	if err == nil {
		t.Fatal("expected error for empty video id")
	}
}
