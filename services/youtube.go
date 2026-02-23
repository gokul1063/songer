package services

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"

	"github.com/gokul1063/songer/configs"
	"github.com/gokul1063/songer/internal"
)

func fetchList(songName string) (string, error) {
	searchURL := "https://www.youtube.com/results?search_query=" +
		url.QueryEscape(songName) +
		"&sp=EgIQAQ%3D%3D"

	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	return processList(string(body))
}

func processList(htmlString string) (string, error) {
	reg := regexp.MustCompile(`"videoId":"([^"]+)"`)

	match := reg.FindStringSubmatch(htmlString)
	if len(match) < 2 {
		return "", fmt.Errorf("no video found")
	}

	videoID := match[1]
	url := fmt.Sprintf("https://youtube.com/watch?v=%s", videoID)
	return url, nil
}

func downloadSong(songLink string, songName string) bool {
	cmd := exec.Command(
		"yt-dlp",
		"-x",
		"--audio-format", "mp3",
		"-o", configs.DataPath+songName+".%(ext)s",
		songLink,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		internal.WriteLog("youtueb.go", err)
		return false
	}

	return true
}
