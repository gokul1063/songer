package main

import (
	"fmt"
	"time"

	"songer-v3/internal/logger"
	"songer-v3/model"
	"songer-v3/service/queue"
	"songer-v3/service/youtube"
)

func main() {
	err := logger.InitLogger()
	if err != nil {
		fmt.Println("Logger init failed:", err)
		return
	}

	testQueueVerbose()
}

func testQueueVerbose() {
	results, err := youtube.Search("alan walker faded")
	if err != nil {
		fmt.Println("Search error:", err)
		return
	}

	// add first 5 songs
	for i := 0; i < 5 && i < len(results); i++ {
		queue.Add(results[i])
	}

	go queue.StartAutoPlay()

	// live monitor loop
	for {
		printQueueState(queue.GetState())
		time.Sleep(3 * time.Second)
	}
}

func printQueueState(q *model.Queue) {
	fmt.Println("\n==============================")

	// Current
	if q.Current != nil {
		fmt.Println("▶ CURRENT:")
		fmt.Printf("  %s (%s)\n", q.Current.Title, q.Current.VideoID)
	} else {
		fmt.Println("▶ CURRENT: None")
	}

	// Upcoming
	fmt.Println("\n⏭ UPCOMING:")
	if len(q.Upcoming) == 0 {
		fmt.Println("  (empty)")
	} else {
		for i, s := range q.Upcoming {
			fmt.Printf("  %d. %s\n", i+1, s.Title)
		}
	}

	// History
	fmt.Println("\n⏮ HISTORY:")
	if len(q.History) == 0 {
		fmt.Println("  (empty)")
	} else {
		for i, s := range q.History {
			fmt.Printf("  %d. %s\n", i+1, s.Title)
		}
	}

	fmt.Println("==============================")
}
