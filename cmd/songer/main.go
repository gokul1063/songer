package main

import (
	"fmt"
	"time"

	"songer/internal/player"
	"songer/internal/queue"
	"songer/internal/youtube"
)

func main() {

	socket := "/tmp/songer-mpv.sock"
	baseDir := "/home/coder/.local/share/songer"

	fmt.Println("starting player")

	p := player.NewPlayer(socket)

	err := p.Start()
	if err != nil {
		fmt.Println("player start error:", err)
		return
	}

	fmt.Println("player started")

	q := queue.New(p, baseDir)

	fmt.Println("searching youtube...")

	results, err := youtube.Search("lofi hip hop")
	if err != nil {
		fmt.Println("search error:", err)
		return
	}

	if len(results) == 0 {
		fmt.Println("no results found")
		return
	}

	fmt.Println("results found:", len(results))

	for i := 0; i < 5 && i < len(results); i++ {

		v := results[i]

		fmt.Println("adding:", v.Title)

		q.Add(v)
	}

	fmt.Println("starting queue")

	err = q.Start()
	if err != nil {
		fmt.Println("queue error:", err)
		return
	}

	fmt.Println("playing first song")

	// simple runtime loop to inspect player state

	for {

		title, _ := p.GetTitle()
		cur, _ := p.GetCurrentTime()
		dur, _ := p.GetDuration()

		fmt.Println("title:", title)
		fmt.Println("time:", cur, "/", dur)

		time.Sleep(5 * time.Second)
	}
}
