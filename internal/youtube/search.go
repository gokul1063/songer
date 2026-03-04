package youtube

import (
	"net/url"
	"os/exec"
)

func FetchSearchPage(query string) ([]byte, error) {

	q := url.QueryEscape(query)

	cmd := exec.Command(
		"curl",
		"-s",
		"https://www.youtube.com/results?search_query="+q,
	)

	return cmd.Output()
}

func Search(query string) ([]Video, error) {

	html, err := FetchSearchPage(query)
	if err != nil {
		return nil, err
	}

	jsonData, err := ExtractInitialData(html)
	if err != nil {
		return nil, err
	}

	return ParseVideos(jsonData)
}
