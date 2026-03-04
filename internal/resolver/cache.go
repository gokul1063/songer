package resolver

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func metadataPath(base string, id string) string {
	return filepath.Join(base, "metadata", id+".json")
}

func songPath(base string, id string) string {
	return filepath.Join(base, "songs", id+".mp3")
}

func Exists(base string, id string) bool {

	meta := metadataPath(base, id)

	_, err := os.Stat(meta)

	return err == nil
}

func Load(base string, id string) (Metadata, error) {

	path := metadataPath(base, id)

	data, err := os.ReadFile(path)
	if err != nil {
		return Metadata{}, err
	}

	var meta Metadata

	err = json.Unmarshal(data, &meta)

	return meta, err
}

func Save(base string, meta Metadata) error {

	dir := filepath.Join(base, "metadata")

	os.MkdirAll(dir, 0755)

	data, _ := json.MarshalIndent(meta, "", "  ")

	return os.WriteFile(metadataPath(base, meta.ID), data, 0644)
}
