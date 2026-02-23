package services

/*

here I'll create a playLocalSong(path string) { } which plays the song using the file name

and playOnlineSong(link string) {} this pays the song using the https link


and PlaySong(songName string) error {} this usese the internal package and uses a specific funtion to play online or offiline

*/

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/gokul1063/songer/configs"
	"github.com/gokul1063/songer/internal"
)

func playOfflineSong(songName string) {
	fmt.Println("file exist localy so lets play the local file")
	fullName := configs.DataPath + songName + ".mp3"
	cmd := exec.Command(
		"mpv",
		"--no-video",
		fullName,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		internal.WriteLog("play.go", err)
	}

}

func playOnlineSong(songName string) {
	fmt.Println("file does not exist so lets play via youtube.com")
	songLink, err := fetchList(songName)
	if err != nil {
		internal.WriteLog("play.go", err)
	}

	if saved := downloadSong(songLink, songName); !saved {
		fmt.Println("saved failed")
	}

	playOfflineSong(songName)

}
func PlaySong(songName string) error {
	if exist := internal.IsFileExist(songName); exist {
		playOfflineSong(songName)
		return nil
	} else {
		playOnlineSong(songName)
		return nil
	}

}
