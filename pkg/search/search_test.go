package search

import (
	"context"
	"encoding/json"
	"testing"
)

func TestParseVideos(t *testing.T) {
	raw := fixture()
	videos := parseVideos(raw)
	if len(videos) != 2 {
		t.Fatalf("expected 2 videos, got %d", len(videos))
	}

	want := Video{
		ID:       "dQw4w9WgXcQ",
		Title:    "Rick Astley - Never Gonna Give You Up (Official Video)",
		URL:      "https://youtu.be/dQw4w9WgXcQ",
		Duration: "3:32",
		Channel:  "Rick Astley",
		Views:    "1,234 views",
	}
	if videos[0] != want {
		t.Fatalf("unexpected video:\n got %+v\nwant %+v", videos[0], want)
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	_, err := NewClient().Search(context.Background(), "   ", 5)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func fixture() []byte {
	doc := map[string]any{
		"contents": map[string]any{
			"twoColumnSearchResultsRenderer": map[string]any{
				"primaryContents": map[string]any{
					"sectionListRenderer": map[string]any{
						"contents": []any{
							map[string]any{
								"itemSectionRenderer": map[string]any{
									"contents": []any{
										map[string]any{
											"videoRenderer": map[string]any{
												"videoId": "dQw4w9WgXcQ",
												"title": map[string]any{
													"runs": []any{
														map[string]any{"text": "Rick Astley - Never Gonna Give You Up (Official Video)"},
													},
												},
												"lengthText": map[string]any{"simpleText": "3:32"},
												"ownerText":  map[string]any{"runs": []any{map[string]any{"text": "Rick Astley"}}},
												"viewCountText": map[string]any{
													"simpleText": "1,234 views",
												},
											},
										},
										map[string]any{
											"videoRenderer": map[string]any{
												"videoId":       "xyz123",
												"title":         map[string]any{"simpleText": "Second result"},
												"lengthText":    map[string]any{"simpleText": "4:00"},
												"ownerText":     map[string]any{"runs": []any{map[string]any{"text": "Channel B"}}},
												"viewCountText": map[string]any{"simpleText": "99 views"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	b, _ := json.Marshal(doc)
	return b
}
