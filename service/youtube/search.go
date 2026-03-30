package youtube

import (
	"fmt"
	"net/url"
	"time"

	"songer-v3/internal/executor"
	"songer-v3/internal/logger"
	"songer-v3/internal/workflow"
	"songer-v3/model"
)

func Search(query string) ([]model.Song, error) {
	workflow.Enter("YouTubeSearch")
	defer workflow.Exit("YouTubeSearch", "done")

	encoded := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://www.youtube.com/results?search_query=%s", encoded)

	cfg := executor.ExecConfig{
		Timeout: 10 * time.Second,
		Retries: 2,
	}

	res := executor.RunCommand("curl", []string{
		"-s",
		"-L",
		"-A", "Mozilla/5.0",
		searchURL,
	}, cfg)


	if res.Err != nil {
		logger.LogError(fmt.Errorf("cmd error: %v, stderr: %s", res.Err, res.Stderr))
		return nil, res.Err
	}

	if res.Stdout == "" {
		err := fmt.Errorf("empty response from youtube")
		logger.LogError(err)
		return nil, err
	}

	return parseHTML(res.Stdout), nil
}
