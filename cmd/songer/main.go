package main

import (
	"fmt"
	"time"

	"songer/internal/service"
)

func main() {

	socket := "/tmp/songer-mpv.sock"
	baseDir := "/home/coder/.local/share/songer"

	s, err := service.New(socket, baseDir)
	if err != nil {
		fmt.Println("service start error:", err)
		return
	}

	s.Listen()

	results, err := s.Search("Konjam")
	if err != nil {
		fmt.Println("search error:", err)
		return
	}

	for i := 0; i < 5 && i < len(results); i++ {
		fmt.Println("adding:", results[i].Title)
		s.Add(results[i])
	}

	err = s.Start()
	if err != nil {
		fmt.Println("queue start error:", err)
		return
	}

	for {

		title, _ := s.Player.GetTitle()
		cur, _ := s.Player.GetCurrentTime()
		dur, _ := s.Player.GetDuration()

		fmt.Println("title:", title)
		fmt.Println("time:", cur, "/", dur)

		time.Sleep(5 * time.Second)
	}
}
