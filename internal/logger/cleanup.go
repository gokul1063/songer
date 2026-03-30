package logger

import (
	"os"
	"path/filepath"
	"sort"
)

func cleanupOldLogs(dir string, maxFiles int) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	type fileInfo struct {
		name string
		time int64
	}

	var logFiles []fileInfo

	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			continue
		}

		logFiles = append(logFiles, fileInfo{
			name: f.Name(),
			time: info.ModTime().Unix(),
		})
	}

	// sort oldest first
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].time < logFiles[j].time
	})

	// delete extra files
	for len(logFiles) > maxFiles {
		old := logFiles[0]
		_ = os.Remove(filepath.Join(dir, old.name))
		logFiles = logFiles[1:]
	}
}
