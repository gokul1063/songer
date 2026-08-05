package play

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"songer/pkg/search"
)

const statusPrefix = "SONGER_TIME "

type Options struct {
	Video   bool
	OnStart func(time.Duration)
}

type Session struct {
	Video     search.Video
	StartedIn time.Duration
}

func Play(ctx context.Context, video search.Video, opts Options) (*Session, error) {
	if _, err := exec.LookPath("mpv"); err != nil {
		return nil, fmt.Errorf("mpv not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, "mpv", buildArgs(video, opts)...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	sess := &Session{Video: video}

	var once sync.Once
	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 64*1024), 64*1024)
		for sc.Scan() {
			if _, ok := detectStart(sc.Text()); !ok {
				continue
			}
			once.Do(func() {
				started := time.Since(start)
				sess.StartedIn = started
				if opts.OnStart != nil {
					opts.OnStart(started)
				}
			})
		}
	}()
	cmd.Wait()
	<-done

	if err := ctx.Err(); err != nil {
		return sess, err
	}
	return sess, nil
}

func buildArgs(video search.Video, opts Options) []string {
	args := []string{}
	if !opts.Video {
		args = append(args, "--no-video")
	}
	args = append(args, "--term-status-msg="+statusPrefix+"${playback-time}")
	return append(args, video.URL)
}

func detectStart(line string) (time.Duration, bool) {
	idx := strings.Index(line, statusPrefix)
	if idx < 0 {
		return 0, false
	}
	rest := line[idx+len(statusPrefix):]
	rest = strings.TrimRight(rest, "\r ")
	parts := strings.Split(rest, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	total := 0
	mult := 1
	for i := len(parts) - 1; i >= 0; i-- {
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil {
			return 0, false
		}
		total += n * mult
		mult *= 60
	}
	if total <= 0 {
		return 0, false
	}
	return time.Duration(total) * time.Second, true
}
