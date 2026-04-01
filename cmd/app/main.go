package main

import (
	"fmt"
	"time"

	"songer-v3/internal/logger"
	"songer-v3/internal/ui"
	"songer-v3/service/player"
	"songer-v3/service/queue"
	"songer-v3/service/youtube"
)

func main() {
	err := logger.InitLogger()
	if err != nil {
		fmt.Println("Logger init failed:", err)
		return
	}

	setup()

	go ui.StartUI()

	// keep backend alive
	for {
		time.Sleep(time.Second)
	}
}

func setup() {
	results, err := youtube.Search("alan walker faded")
	if err != nil {
		fmt.Println("Search error:", err)
		return
	}

	for i := 0; i < 5 && i < len(results); i++ {
		queue.Add(results[i])
	}

	player.SetOnEndCallback(func() {
		queue.PlayNext()
		queue.PreloadNext()
	})

	go queue.StartAutoPlay()
	go queue.StartPreloader()
	go player.StartEventListener()
}
