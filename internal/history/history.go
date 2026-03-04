package history

import (
	"encoding/json"
	"os"
	"time"
)

type Entry struct {
	VideoID string `json:"video_id"`
	Title   string `json:"title"`
	Time    int64  `json:"time"`
}

func Append(path string, id string, title string) error {

	entry := Entry{
		VideoID: id,
		Title:   title,
		Time:    time.Now().Unix(),
	}

	var entries []Entry

	data, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(data, &entries)
	}

	entries = append(entries, entry)

	out, _ := json.MarshalIndent(entries, "", "  ")

	return os.WriteFile(path, out, 0644)
}
