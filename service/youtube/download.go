package youtube

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"songer-v3/config"
	"songer-v3/database/library"
	"songer-v3/internal/executor"
	"songer-v3/internal/logger"
	"songer-v3/internal/workflow"
)

func isExist(filePath string) (bool, error){
	if _, err := os.Stat(filePath); err == nil {
		return true, nil
	}else {
		return false, err
	}


}

func Download(videoID string) (string, error) {
	workflow.Enter("YouTubeDownload")
	defer workflow.Exit("YouTubeDownload", "done")

	err := os.MkdirAll(config.AppPaths.DownloadDir, os.ModePerm)
	if err != nil {
		logger.LogError(err)
		return "", err
	}

	filePath := filepath.Join(config.AppPaths.DownloadDir, videoID+".mp3")

	if _, err := os.Stat(filePath); err == nil {
		return filePath, nil
	}

	url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

	cfg := executor.ExecConfig{
		Timeout: 2 * time.Minute,
		Retries: 3,
	}

	res := executor.RunCommand("yt-dlp", []string{
		"-f", "bestaudio",
		"-x",
		"--audio-format", "mp3",
		"-o", filePath,
		url,
	}, cfg)

	if res.Err != nil {
		logger.LogError(fmt.Errorf("yt-dlp failed: %v, stderr: %s", res.Err, res.Stderr))
		return "", res.Err
	}

	absPath, _ := filepath.Abs(filePath)

	_ = library.Add(videoID, absPath)

	return absPath, nil
}
