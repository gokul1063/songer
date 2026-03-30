package youtube

import (
	"encoding/json"
	"fmt"
	"regexp"

	"songer-v3/internal/logger"
	"songer-v3/internal/workflow"
	"songer-v3/model"
)

func parseHTML(html string) []model.Song {
	workflow.Enter("parseHTML")
	defer workflow.Exit("parseHTML", "done")

	jsonData := extractJSON(html)
	if jsonData == "" {
		err := fmt.Errorf("ytInitialData not found")
		logger.LogError(err)
		return nil
	}

	return parseJSON(jsonData)
}

func extractJSON(html string) string {
	workflow.Enter("extractJSON")
	defer workflow.Exit("extractJSON", "done")

	re := regexp.MustCompile(`var ytInitialData = (.*?);</script>`)
	match := re.FindStringSubmatch(html)

	if len(match) < 2 {
		err := fmt.Errorf("failed to extract ytInitialData")
		logger.LogError(err)
		return ""
	}

	return match[1]
}

func parseJSON(data string) []model.Song {
	workflow.Enter("parseJSON")
	defer workflow.Exit("parseJSON", "done")

	var result map[string]interface{}

	err := json.Unmarshal([]byte(data), &result)
	if err != nil {
		logger.LogError(err)
		return nil
	}

	return extractVideos(result)
}

func extractVideos(data map[string]interface{}) []model.Song {
	workflow.Enter("extractVideos")
	defer workflow.Exit("extractVideos", "done")

	var songs []model.Song

	contents, ok := data["contents"].(map[string]interface{})
	if !ok {
		logger.LogError(fmt.Errorf("missing contents field"))
		return songs
	}

	twoCol, ok := contents["twoColumnSearchResultsRenderer"].(map[string]interface{})
	if !ok {
		logger.LogError(fmt.Errorf("missing twoColumnSearchResultsRenderer"))
		return songs
	}

	primary, ok := twoCol["primaryContents"].(map[string]interface{})
	if !ok {
		logger.LogError(fmt.Errorf("missing primaryContents"))
		return songs
	}

	sectionList, ok := primary["sectionListRenderer"].(map[string]interface{})
	if !ok {
		logger.LogError(fmt.Errorf("missing sectionListRenderer"))
		return songs
	}

	sections, ok := sectionList["contents"].([]interface{})
	if !ok {
		logger.LogError(fmt.Errorf("missing sections array"))
		return songs
	}

	for _, sec := range sections {
		secMap, ok := sec.(map[string]interface{})
		if !ok {
			continue
		}

		itemSection, ok := secMap["itemSectionRenderer"].(map[string]interface{})
		if !ok {
			continue
		}

		items, ok := itemSection["contents"].([]interface{})
		if !ok {
			continue
		}

		for _, item := range items {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			video, ok := itemMap["videoRenderer"].(map[string]interface{})
			if !ok {
				continue
			}

			videoID, _ := video["videoId"].(string)

			title := ""
			if t, ok := video["title"].(map[string]interface{}); ok {
				if runs, ok := t["runs"].([]interface{}); ok && len(runs) > 0 {
					if r, ok := runs[0].(map[string]interface{}); ok {
						title, _ = r["text"].(string)
					}
				}
			}

			duration := ""
			if d, ok := video["lengthText"].(map[string]interface{}); ok {
				if txt, ok := d["simpleText"].(string); ok {
					duration = txt
				}
			}

			if videoID != "" {
				songs = append(songs, model.Song{
					Title:    title,
					VideoID:  videoID,
					Duration: duration,
				})
			}
		}
	}

	return songs
}
