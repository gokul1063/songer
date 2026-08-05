package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"text/tabwriter"
	"time"

	"songer/pkg/autoplay"
	"songer/pkg/config"
	"songer/pkg/download"
	"songer/pkg/mpv"
	"songer/pkg/play"
	"songer/pkg/search"
	"songer/pkg/tui"
)

func main() {
	source := flag.String("source", "", "song name or video to search on YouTube")
	limit := flag.Int("limit", 10, "max number of results to return")
	rank := flag.Int("rank", 1, "play the Nth search result (1-based)")
	video := flag.Bool("video", false, "play with video instead of audio-only")
	noPlay := flag.Bool("no-play", false, "search only, do not start playback")
	doAutoplay := flag.Bool("autoplay", false, "build a suggested queue (2 + 4 = 6 videos) for the played song")
	download := flag.Bool("download", false, "download the selected result and exit")
	downloadAll := flag.Bool("download-all", false, "download all search results concurrently and exit")
	downloadDir := flag.String("download-dir", "downloads", "directory to save downloads")
	workers := flag.Int("workers", 3, "number of concurrent downloads")
	mp3 := flag.Bool("mp3", false, "convert downloads to mp3")
	tuiMode := flag.Bool("tui", false, "launch the keyboard-driven TUI")
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

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v (using defaults)\n", err)
		cfg = config.Default()
	}

	if *tuiMode {
		if err := runTUI(ctx, cfg, target); err != nil {
			if ctx.Err() != nil {
				fmt.Fprintln(os.Stderr, "stopped")
			} else {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		}
		return
	}

	if *download || *downloadAll {
		doDownloads(ctx, videos, target, *rank, *downloadAll, *downloadDir, *workers, *mp3)
		return
	}

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

func doDownloads(ctx context.Context, videos []search.Video, target search.Video, rank int, all bool, dir string, workers int, mp3 bool) {
	if all {
		fmt.Printf("▸ downloading %d videos to %s (%d workers)...\n", len(videos), dir, workers)
		results := download.DownloadMany(ctx, videos, workers, download.Options{Dir: dir, MP3: mp3})
		for res := range results {
			if res.Err != nil {
				fmt.Printf("✗ [%d] %s: %v\n", res.Index+1, res.Video.Title, res.Err)
			} else {
				fmt.Printf("✓ [%d] %s → %s\n", res.Index+1, res.Video.Title, res.Path)
			}
		}
		return
	}

	fmt.Printf("▸ downloading [%d] %s...\n", rank, target.Title)
	path, err := download.Download(ctx, target, download.Options{
		Dir: dir, MP3: mp3,
		OnProgress: func(v search.Video, pct float64) {
			fmt.Printf("\r  %3.0f%%", pct)
		},
	})
	if err != nil {
		if ctx.Err() != nil {
			fmt.Fprintln(os.Stderr, "\nstopped")
		} else {
			fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
		}
		os.Exit(1)
	}
	if path == "" {
		path = dir
	}
	fmt.Printf("\r")
	fmt.Printf("✓ saved → %s\n", path)
}

func runTUI(ctx context.Context, cfg config.Config, target search.Video) error {
	socket := filepath.Join(os.TempDir(), fmt.Sprintf("songer-%d.sock", os.Getpid()))
	player, err := mpv.New(ctx, target.URL, socket, cfg.Player.Volume)
	if err != nil {
		return err
	}
	if err := player.Start(); err != nil {
		return err
	}
	defer player.Close()

	fmt.Fprintf(os.Stderr, "♫ Now playing: %s\n", target.Title)
	return tui.Run(ctx, player, target, cfg.ThemeFor(cfg.UI.Theme), cfg.Autoplay.PerNode, cfg.Autoplay.Depth)
}

