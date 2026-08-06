package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"songer/pkg/autoplay"
	"songer/pkg/config"
	"songer/pkg/download"
	"songer/pkg/library"
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
	favorite := flag.Bool("favorite", false, "add the selected song to favorites")
	liked := flag.Bool("liked", false, "add the selected song to liked")
	playlistName := flag.String("playlist", "", "add the selected song to a playlist (creates it if needed)")
	view := flag.String("view", "", "list a collection: favorites, liked, history, playlists, playlist:<name>")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "songer - play songs from YouTube\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  songer --source \"song name\" [flags]\n  songer --view favorites|liked|history|playlists|playlist:<name> [--rank N]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *source == "" && *view == "" {
		flag.Usage()
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var lib *library.Library
	if l, err := library.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: library: %v\n", err)
	} else {
		lib = l
	}

	if *view != "" {
		runView(ctx, lib, *view, *rank, *video, *noPlay)
		return
	}

	if *source == "" {
		flag.Usage()
		os.Exit(1)
	}

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

	if *rank < 1 || *rank > len(videos) {
		fmt.Fprintf(os.Stderr, "error: --rank %d out of range (1-%d)\n", *rank, len(videos))
		os.Exit(1)
	}
	target := videos[*rank-1]

	if lib != nil {
		if *favorite {
			if err := lib.AddFavorite(target); err != nil {
				fmt.Fprintf(os.Stderr, "favorite error: %v\n", err)
			} else {
				fmt.Println("♥ added to favorites")
			}
		}
		if *liked {
			if err := lib.AddLiked(target); err != nil {
				fmt.Fprintf(os.Stderr, "liked error: %v\n", err)
			} else {
				fmt.Println("★ added to liked")
			}
		}
		if *playlistName != "" {
			if err := lib.AddToPlaylist(*playlistName, target); err != nil {
				fmt.Fprintf(os.Stderr, "playlist error: %v\n", err)
			} else {
				fmt.Printf("♺ added to playlist %q\n", *playlistName)
			}
		}
	}

	if *noPlay {
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v (using defaults)\n", err)
		cfg = config.Default()
	}

	if *tuiMode {
		if err := runTUI(ctx, cfg, target, lib); err != nil {
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

	if lib != nil {
		_ = lib.RecordPlay(target)
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

func runView(ctx context.Context, lib *library.Library, view string, rank int, video, noPlay bool) {
	if lib == nil {
		fmt.Fprintln(os.Stderr, "library unavailable")
		os.Exit(1)
	}

	var entries []library.Entry
	switch {
	case view == "favorites":
		entries = lib.Favorites
	case view == "liked":
		entries = lib.Liked
	case view == "history":
		entries = lib.History
	case view == "playlists":
		names := lib.PlaylistNames()
		if len(names) == 0 {
			fmt.Println("no playlists yet")
			return
		}
		for _, n := range names {
			fmt.Printf("♺ %s (%d)\n", n, len(lib.Playlist(n)))
		}
		return
	case strings.HasPrefix(view, "playlist:"):
		entries = lib.Playlist(strings.TrimPrefix(view, "playlist:"))
	default:
		fmt.Fprintf(os.Stderr, "unknown view %q\n", view)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("nothing here yet")
		return
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for i, e := range entries {
		fmt.Fprintf(tw, "[%d]\t%s\t%s\t%s\n", i+1, e.Video.Title, e.Video.Channel, e.Video.URL)
	}
	tw.Flush()

	if noPlay || rank < 1 || rank > len(entries) {
		return
	}
	target := entries[rank-1].Video
	_ = lib.RecordPlay(target)

	fmt.Printf("\n▶ Playing [%d] %s\n", rank, target.Title)
	if _, err := play.Play(ctx, target, play.Options{Video: video}); err != nil {
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

func runTUI(ctx context.Context, cfg config.Config, target search.Video, lib *library.Library) error {
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
	return tui.Run(ctx, player, target, cfg.ThemeFor(cfg.UI.Theme), cfg.Autoplay.PerNode, cfg.Autoplay.Depth, lib)
}
