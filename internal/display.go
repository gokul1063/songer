package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func DisplaySong() error {
	home := os.Getenv("HOME")

	data, err := scanMusic(home)

	if err != nil {
		WriteLog("scanner.go at ScanMusic", err)
		return err
	}

	for _, values := range data {
		fmt.Printf("Current files in the device : %s\n", values.Name)
	}

	return nil

}

func SearchSong(songName string) string {
	home := os.Getenv("HOME")

	data, err := scanMusic(home)

	if err != nil {
		return ""
	}

	for _, values := range data {

		substr := strings.TrimSuffix(values.Name, filepath.Ext(values.Name))
		fmt.Printf("current song : %s\nsong Path :%s\n", values.Name, values.Path)
		if strings.ToLower(substr) == strings.ToLower(songName) {

			return values.Path
		}

	}

	return ""
}
