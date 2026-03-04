package youtube

import (
	"bytes"
	"encoding/json"
	"errors"
)

func ExtractInitialData(html []byte) ([]byte, error) {

	start := bytes.Index(html, []byte("ytInitialData"))
	if start == -1 {
		return nil, errors.New("ytInitialData not found")
	}

	start = bytes.Index(html[start:], []byte("{")) + start

	end := bytes.Index(html[start:], []byte("};"))
	if end == -1 {
		return nil, errors.New("json end not found")
	}

	return html[start : start+end+1], nil
}

func ParseVideos(data []byte) ([]Video, error) {

	var root map[string]interface{}

	err := json.Unmarshal(data, &root)
	if err != nil {
		return nil, err
	}

	var results []Video

	search := root["contents"].(map[string]interface{})
	searchResults := search["twoColumnSearchResultsRenderer"].(map[string]interface{})
	primary := searchResults["primaryContents"].(map[string]interface{})
	section := primary["sectionListRenderer"].(map[string]interface{})
	contents := section["contents"].([]interface{})

	for _, c := range contents {

		item := c.(map[string]interface{})

		isr, ok := item["itemSectionRenderer"]
		if !ok {
			continue
		}

		items := isr.(map[string]interface{})["contents"].([]interface{})

		for _, it := range items {

			obj := it.(map[string]interface{})

			v, ok := obj["videoRenderer"]
			if !ok {
				continue
			}

			video := v.(map[string]interface{})

			id := video["videoId"].(string)

			titleRuns := video["title"].(map[string]interface{})["runs"].([]interface{})
			title := titleRuns[0].(map[string]interface{})["text"].(string)

			channelRuns := video["ownerText"].(map[string]interface{})["runs"].([]interface{})
			channel := channelRuns[0].(map[string]interface{})["text"].(string)

			results = append(results, Video{
				ID:      id,
				Title:   title,
				Channel: channel,
				URL:     "https://youtube.com/watch?v=" + id,
			})
		}
	}

	return results, nil
}
