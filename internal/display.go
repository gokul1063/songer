package internal

import (
	"fmt"
	"os"
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
