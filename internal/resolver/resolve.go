package resolver

import (
	"path/filepath"
	"songer/internal/youtube"
)

func Resolve(base string, video youtube.Video) (Metadata, error) {

	if Exists(base, video.ID) {
		return Load(base, video.ID)
	}

	meta := Metadata{
		ID:      video.ID,
		Title:   video.Title,
		Channel: video.Channel,
		URL:     video.URL,
		Path:    filepath.Join(base, "songs", video.ID+".mp3"),
	}

	err := youtube.DownloadAudio(video.URL, meta.Path)
	if err != nil {
		return Metadata{}, err
	}

	err = Save(base, meta)
	if err != nil {
		return Metadata{}, err
	}

	return meta, nil
}
