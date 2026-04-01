package ui

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"songer-v3/internal/logger"
	"songer-v3/internal/workflow"
	"songer-v3/model"
	"songer-v3/service/player"
	"songer-v3/service/queue"
)

func StartUI() {
	workflow.Enter("UIStart")
	defer workflow.Exit("UIStart", "done")

	enterAltScreen()
	hideCursor()
	setupExitHandler()

	go listenInput()

	for {
		render(queue.GetState())
		time.Sleep(200 * time.Millisecond)
	}
}

func enterAltScreen() {
	workflow.Enter("UIEnterAltScreen")
	defer workflow.Exit("UIEnterAltScreen", "done")

	fmt.Print("\033[?1049h")
}

func exitAltScreen() {
	workflow.Enter("UIExitAltScreen")
	defer workflow.Exit("UIExitAltScreen", "done")

	fmt.Print("\033[?1049l")
}

func hideCursor() {
	workflow.Enter("UIHideCursor")
	defer workflow.Exit("UIHideCursor", "done")

	fmt.Print("\033[?25l")
}

func showCursor() {
	workflow.Enter("UIShowCursor")
	defer workflow.Exit("UIShowCursor", "done")

	fmt.Print("\033[?25h")
}

func setupExitHandler() {
	workflow.Enter("UISetupExitHandler")
	defer workflow.Exit("UISetupExitHandler", "done")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		cleanup()
		os.Exit(0)
	}()
}

func cleanup() {
	workflow.Enter("UICleanup")
	defer workflow.Exit("UICleanup", "done")

	showCursor()
	exitAltScreen()
}

func render(q *model.Queue) {
	workflow.Enter("UIRender")
	defer workflow.Exit("UIRender", "done")

	clear()

	width := 60

	center := func(s string) string {
		padding := (width - len(s)) / 2
		if padding < 0 {
			padding = 0
		}
		return fmt.Sprintf("%*s%s", padding, "", s)
	}

	fmt.Println(center("♪ Now Playing ♪"))
	fmt.Println()

	if q.Current != nil {
		fmt.Println(center("─── " + q.Current.Title + " ───"))
	} else {
		fmt.Println(center("─── Nothing Playing ───"))
	}

	fmt.Println()
	fmt.Println(center("⏮    ⏯    ⏭"))
	fmt.Println()

	fmt.Println(center("────────────●────────────"))
	fmt.Println(center("00:00 / 00:00"))

	fmt.Println()
	fmt.Println(center("🔊 ███████░░░░░░░ 50%"))
}

func listenInput() {
	workflow.Enter("UIListenInput")
	defer workflow.Exit("UIListenInput", "done")

	buf := make([]byte, 1)

	for {
		_, err := os.Stdin.Read(buf)
		if err != nil {
			logger.LogError(err)
			continue
		}

		switch buf[0] {
		case 'q':
			cleanup()
			os.Exit(0)

		case ' ':
			err := player.Pause()
			if err != nil {
				logger.LogError(err)
			}

		case 'n':
			queue.PlayNext()

		case 'p':
			err := player.Resume()
			if err != nil {
				logger.LogError(err)
			}
		}
	}
}

func clear() {
	fmt.Print("\033[H\033[2J")
}
