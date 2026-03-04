package youtube

import (
	"os/exec"
)

func DownloadAudio(url string, outputDir string) error {

	cmd := exec.Command(
		"yt-dlp",
		"-x",
		"--audio-format",
		"mp3",
		"-o",
		outputDir+"/%(title)s.%(ext)s",
		url,
	)

	return cmd.Run()
}
