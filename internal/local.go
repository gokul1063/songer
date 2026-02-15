package internal

import (
	"os"
	"strings"

	"github.com/gokul1063/songer/configs"
)

const songPath string = configs.DataPath

func IsFileExist(songName string) bool {
	entire, err := os.ReadDir(songPath)

	if err != nil {
		WriteLog("local.go", err)
	}

	for _, entry := range entire {
		if strings.Contains(entry.Name(), ".") {
			if songName == entry.Name()[:len(entry.Name())-4] {
				return true
			}
		}
	}

	return false

}

func helperProcess(songName string) string {
	data := strings.Split(songName, " ")

	for ind, ele := range data {
		data[ind] = strings.ToLower(ele)
	}

	var result string = strings.Join(data, "-")
	return result
}

func ProcessSongName(songName string) string {
	if !strings.Contains(songName, " ") {
		return songName
	}

	return helperProcess(songName)
}

func TisFileExist(songName string) bool {
	return IsFileExist(songName)
}
