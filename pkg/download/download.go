package download

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"

	"songer/pkg/search"
)

var progressRe = regexp.MustCompile(`\[download\]\s+([0-9]+(?:\.[0-9]+)?)%`)

const filePrefix = "FILE="

type Options struct {
	Dir        string
	MP3        bool
	Bin        string
	OnProgress func(video search.Video, percent float64)
}

type Result struct {
	Index int
	Video search.Video
	Path  string
	Err   error
}

// Download fetches the best audio of a video via yt-dlp and returns the saved path.
func Download(ctx context.Context, video search.Video, opts Options) (string, error) {
	bin := opts.Bin
	if bin == "" {
		bin = "yt-dlp"
	}
	if _, err := exec.LookPath(bin); err != nil {
		return "", fmt.Errorf("%s not found: %w", bin, err)
	}

	dir := opts.Dir
	if dir == "" {
		dir = "downloads"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	url := video.URL
	if url == "" {
		url = "https://youtu.be/" + video.ID
	}

	args := []string{
		"--no-playlist",
		"--newline",
		"--print", "after_move:" + filePrefix + "%(filepath)s",
	}
	args = append(args, jsRuntimeFlag()...)
	if opts.MP3 {
		args = append(args, "-x", "--audio-format", "mp3", "--audio-quality", "0")
	} else {
		args = append(args, "-f", "bestaudio")
	}
	args = append(args, "-o", filepath.Join(dir, "%(title)s.%(ext)s"), url)

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}

	var path string
	var mu sync.Mutex
	setPath := func(p string) {
		mu.Lock()
		if path == "" {
			path = p
		}
		mu.Unlock()
	}

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 64*1024), 64*1024)
	for sc.Scan() {
		line := sc.Text()
		if hasPrefix(line, filePrefix) {
			setPath(line[len(filePrefix):])
			continue
		}
		if m := progressRe.FindStringSubmatch(line); len(m) == 2 && opts.OnProgress != nil {
			pct, _ := strconv.ParseFloat(m[1], 64)
			opts.OnProgress(video, pct)
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", err
	}

	mu.Lock()
	defer mu.Unlock()
	return path, nil
}

// DownloadMany downloads a list of videos concurrently, streaming one Result
// per video as it finishes. Order is completion order, not input order.
func DownloadMany(ctx context.Context, videos []search.Video, workers int, opts Options) <-chan Result {
	results := make(chan Result)
	if workers <= 0 {
		workers = 3
	}

	jobs := make(chan Result)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				path, err := Download(ctx, job.Video, opts)
				job.Path, job.Err = path, err
				select {
				case results <- job:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(results)
		go func() {
			defer close(jobs)
			for i, v := range videos {
				select {
				case jobs <- Result{Index: i, Video: v}:
				case <-ctx.Done():
					return
				}
			}
		}()
		wg.Wait()
	}()

	return results
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func jsRuntimeFlag() []string {
	for _, name := range []string{"deno", "bun", "node"} {
		if _, err := exec.LookPath(name); err == nil {
			return []string{"--js-runtimes", name}
		}
	}
	return nil
}
