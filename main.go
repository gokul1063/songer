package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"songer/pkg/autoplay"
	"songer/pkg/play"
	"songer/pkg/search"
)

func main() {
	source := flag.String("source", "", "song name or video to search on YouTube")
	limit := flag.Int("limit", 10, "max number of results to return")
	rank := flag.Int("rank", 1, "play the Nth search result (1-based)")
	video := flag.Bool("video", false, "play with video instead of audio-only")
	noPlay := flag.Bool("no-play", false, "search only, do not start playback")
	doAutoplay := flag.Bool("autoplay", false, "build a suggested queue (2 + 4 = 6 videos) for the played song")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "songer - play songs from YouTube\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  songer --source \"song name\" [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *source == "" {
		flag.Usage()
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := search.NewClient()
	searchStart := time.Now()
	videos, err := client.Search(ctx, *source, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	searchTime := time.Since(searchStart)

	if len(videos) == 0 {
		fmt.Fprintln(os.Stderr, "no results found")
		os.Exit(1)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for i, v := range videos {
		fmt.Fprintf(tw, "[%d]\t%s\t%s\t%s\t%s\t%s\n",
			i+1, v.Title, v.Duration, v.Channel, v.Views, v.URL)
	}
	tw.Flush()

	if *noPlay {
		return
	}

	if *rank < 1 || *rank > len(videos) {
		fmt.Fprintf(os.Stderr, "error: --rank %d out of range (1-%d)\n", *rank, len(videos))
		os.Exit(1)
	}
	target := videos[*rank-1]

	if *doAutoplay {
		queueStart := time.Now()
		queue, err := autoplay.NewClient().BuildQueue(ctx, target.ID, 2, 2)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoplay error: %v\n", err)
		} else {
			fmt.Printf("\n◆ Up next (built in %s):\n", time.Since(queueStart).Round(10*time.Millisecond))
			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			for i, v := range queue {
				fmt.Fprintf(tw, "  %d.\t%s\t%s\t%s\n", i+1, v.Title, v.Channel, v.URL)
			}
			tw.Flush()
		}
	}

	fmt.Printf("\n▶ Playing [%d] %s\n", *rank, target.Title)
	_, err = play.Play(ctx, target, play.Options{
		Video: *video,
		OnStart: func(d time.Duration) {
			fmt.Printf("✔ started in %s (search %s, total %s)\n",
				d.Round(100*time.Millisecond), searchTime.Round(10*time.Millisecond),
				(d+searchTime).Round(100*time.Millisecond))
		},
	})
	if err != nil {
		if ctx.Err() != nil {
			fmt.Fprintln(os.Stderr, "stopped")
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}

