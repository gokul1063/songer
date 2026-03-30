package library

import (
	"encoding/json"
	"os"

	"songer-v3/internal/logger"
	"songer-v3/internal/workflow"
	"songer-v3/model"
)

var dbPath = "database/library/library.json"

func Load() (*model.Library, error) {
	workflow.Enter("LibraryLoad")
	defer workflow.Exit("LibraryLoad", "done")

	file, err := os.Open(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.Library{
				Songs: make(map[string]string),
			}, nil
		}
		logger.LogError(err)
		return nil, err
	}
	defer file.Close()

	var lib model.Library
	err = json.NewDecoder(file).Decode(&lib)
	if err != nil {
		logger.LogError(err)
		return nil, err
	}

	// 🔥 FIX: ensure map is initialized
	if lib.Songs == nil {
		lib.Songs = make(map[string]string)
	}

	return &lib, nil
}

func Save(lib *model.Library) error {
	workflow.Enter("LibrarySave")
	defer workflow.Exit("LibrarySave", "done")

	file, err := os.Create(dbPath)
	if err != nil {
		logger.LogError(err)
		return err
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(lib)
	if err != nil {
		logger.LogError(err)
		return err
	}

	return nil
}

func Add(videoID string, filePath string) error {
	workflow.Enter("LibraryAdd")
	defer workflow.Exit("LibraryAdd", "done")

	lib, err := Load()
	if err != nil {
		return err
	}

	lib.Songs[videoID] = filePath

	return Save(lib)
}

func Exists(videoID string) (string, bool) {
	lib, err := Load()
	if err != nil {
		return "", false
	}

	path, ok := lib.Songs[videoID]
	return path, ok
}
