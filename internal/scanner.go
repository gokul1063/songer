package internal

import (
	"github.com/gokul1063/songer/configs"
	"os"
	"path/filepath"
)

func scanMusic(root string) ([]configs.SongFile, error) {
	var songs []configs.SongFile

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {

		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)

		if configs.IsSupported(ext) {
			songs = append(songs, configs.SongFile{
				Path: path,
				Name: d.Name(),
			})
		}

		return nil
	})

	return songs, err

}
