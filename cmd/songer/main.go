package main

import (
	"fmt"
	"time"

	"songer/internal/player"
)

func main() {

	fmt.Println("Starting Songer test")

	p := player.NewPlayer("/tmp/songer-mpv.sock")

	err := p.Start()
	if err != nil {
		fmt.Println("mpv start failed:", err)
		return
	}

	p.ListenEvents(func(e player.Event) {
		fmt.Println("event:", e.Event)
	})
	fmt.Println("mpv started")

	err = p.Play("/home/coder/Desktop/m_pho/songs/Usure.mpga")
	if err != nil {
		fmt.Println("play failed:", err)
		return
	}

	fmt.Println("playing song")

	time.Sleep(35 * time.Second)

	err = p.Pause()
	if err != nil {
		fmt.Println("pause failed:", err)
		return
	}

	fmt.Println("paused")

	time.Sleep(3 * time.Second)

	err = p.Pause()
	if err != nil {
		fmt.Println("resume failed:", err)
		return
	}

	fmt.Println("resumed")

	time.Sleep(5 * time.Second)

	err = p.Stop()
	if err != nil {
		fmt.Println("stop failed:", err)
		return
	}

	fmt.Println("stopped")
}
