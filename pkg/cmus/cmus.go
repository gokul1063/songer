package cmus

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"songer/pkg/player"
)

// pollInterval is how often we query cmus-remote -Q while a stream plays.
const pollInterval = 300 * time.Millisecond

// Snapshot is the parsed output of `cmus-remote -Q`.
type Snapshot struct {
	State    string // "playing", "paused" or "stopped"
	Position int    // seconds
	Duration int    // seconds
	Volume   int    // 0-100
	Title    string
}

// Command sends a cmus command through cmus-remote. cmd is passed verbatim,
// so it must be a single cmus command line (e.g. "add -p <url>").
func Command(cmd string) error {
	_, err := remote("C", cmd)
	return err
}

// Query returns a snapshot of the running cmus instance.
func Query() (Snapshot, error) {
	out, err := remote("Q")
	if err != nil {
		return Snapshot{}, err
	}
	return parseStatus(out), nil
}

// ResolveStream returns a direct audio URL for a YouTube video using yt-dlp.
// cmus has no native YouTube support, so the URL is resolved here and handed
// to cmus as a plain HTTP stream.
func ResolveStream(ctx context.Context, url string) (string, error) {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return "", fmt.Errorf("yt-dlp not found (needed to resolve streams for cmus): %w", err)
	}
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--no-playlist",
		"-f", "bestaudio[ext=m4a]/bestaudio",
		"-g",
		url,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	line := strings.TrimSpace(stdout.String())
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	if line == "" {
		return "", fmt.Errorf("yt-dlp returned no stream URL")
	}
	return line, nil
}

// Player drives an already-running cmus instance through cmus-remote and
// exposes the same interface as the mpv controller so the TUI can use either.
type Player struct {
	ctx    context.Context
	events chan player.State
	state  player.State
	mu     sync.Mutex
	prev   string
	closed bool
	stop   chan struct{}
	done   chan struct{}
}

// New verifies cmus is reachable and returns a player ready to load streams.
func New(ctx context.Context, volume int) (*Player, error) {
	if _, err := exec.LookPath("cmus-remote"); err != nil {
		return nil, fmt.Errorf("cmus-remote not found: %w", err)
	}
	q, err := Query()
	if err != nil {
		return nil, fmt.Errorf("cmus is not running (start `cmus` first): %w", err)
	}
	// Respect the volume of the running cmus instance; fall back to the
	// configured default only when cmus has nothing to report.
	if q.Volume >= 0 {
		volume = q.Volume
	}
	return &Player{
		ctx:    ctx,
		events: make(chan player.State, 32),
		state:  player.State{Volume: volume},
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}, nil
}

func (p *Player) Events() <-chan player.State { return p.events }

func (p *Player) State() player.State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

// Start queues url in cmus and begins polling for state changes.
func (p *Player) Start(url string) error {
	if err := p.load(url); err != nil {
		return err
	}
	go p.pollLoop()
	return nil
}

// Load replaces the current media with url (used for "next"). cmus has no
// idle mode, so we clear the queue, add the stream and play it.
func (p *Player) Load(url string) {
	_ = p.load(url)
}

// load implements Load, returning the first error so Start can fail fast.
func (p *Player) load(url string) error {
	p.mu.Lock()
	p.state.Ended = false
	p.state.Position = 0
	p.state.Duration = 0
	p.state.Title = ""
	// The clear command below momentarily stops cmus; treating that as "ended"
	// would make the UI skip a song. Reset the tracked state so the brief
	// stopped status is ignored until the new stream starts playing.
	p.prev = "stopped"
	p.mu.Unlock()

	stream, err := ResolveStream(p.ctx, url)
	if err != nil {
		return err
	}
	for _, c := range []string{"clear", "add -p " + stream, "player-play"} {
		if err := Command(c); err != nil {
			return fmt.Errorf("cmus-remote: %w", err)
		}
	}
	return nil
}

func (p *Player) pollLoop() {
	defer close(p.done)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			p.pollOnce()
		}
	}
}

func (p *Player) pollOnce() {
	q, err := Query()
	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		// cmus went away (e.g. the user quit it): mark ended so the UI reacts.
		if p.prev == "playing" || p.prev == "paused" {
			p.prev = "stopped"
			p.state.Ended = true
			p.pushLocked()
		}
		return
	}
	if q.State == "" {
		return
	}

	changed := false
	switch q.State {
	case "playing":
		if p.prev != "playing" {
			p.state.Paused = false
			changed = true
		}
		p.prev = "playing"
	case "paused":
		if !p.state.Paused {
			p.state.Paused = true
			changed = true
		}
		p.prev = "paused"
	case "stopped":
		if p.prev == "playing" || p.prev == "paused" {
			p.state.Ended = true
			changed = true
		}
		p.prev = "stopped"
	}

	if q.Position >= 0 {
		if d := time.Duration(q.Position) * time.Second; d != p.state.Position {
			p.state.Position = d
			changed = true
		}
	}
	if q.Duration > 0 {
		if d := time.Duration(q.Duration) * time.Second; d != p.state.Duration {
			p.state.Duration = d
			changed = true
		}
	}
	if q.Volume >= 0 && q.Volume != p.state.Volume {
		p.state.Volume = q.Volume
		changed = true
	}
	if q.Title != "" && q.Title != p.state.Title {
		p.state.Title = q.Title
		changed = true
	}

	if changed {
		p.pushLocked()
	}
}

// pushLocked sends the current state; callers must hold p.mu.
func (p *Player) pushLocked() {
	s := p.state
	select {
	case p.events <- s:
	default:
	}
}

func (p *Player) Seek(d time.Duration) {
	_ = Command(seekCmd(d))
}

// seekCmd renders a cmus seek command. cmus relative seeks require an explicit
// + or - sign; a bare number means "seek to absolute position N seconds".
func seekCmd(d time.Duration) string {
	secs := int64(d.Seconds())
	if secs == 0 {
		return "seek 0"
	}
	sign := "+"
	if secs < 0 {
		sign = "-"
		secs = -secs
	}
	return "seek " + sign + strconv.FormatInt(secs, 10)
}

func (p *Player) TogglePause() {
	_ = Command("player-pause")
}

func (p *Player) SetVolume(v int) {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	_ = Command(fmt.Sprintf("set vol %d", v))
	p.mu.Lock()
	p.state.Volume = v
	p.mu.Unlock()
}

func (p *Player) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	close(p.stop)
	<-p.done
	_ = Command("player-stop")
	close(p.events)
}

// remote runs cmus-remote with the given arguments. For commands (-C) the
// whole command string is one argv element, so no shell quoting is needed.
func remote(args ...string) (string, error) {
	cmd := exec.Command("cmus-remote", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stderr.String(), err
	}
	return stdout.String(), nil
}

func parseStatus(out string) Snapshot {
	var s Snapshot
	s.Position = -1
	s.Duration = -1
	s.Volume = -1
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "status "):
			s.State = strings.TrimSpace(line[len("status "):])
		case strings.HasPrefix(line, "filepos "):
			s.Position = atoi(line[len("filepos "):])
		case strings.HasPrefix(line, "duration "):
			s.Duration = atoi(line[len("duration "):])
		case strings.HasPrefix(line, "vol_left "):
			s.Volume = atoi(line[len("vol_left "):])
		case strings.HasPrefix(line, "tag title "):
			s.Title = strings.TrimSpace(line[len("tag title "):])
		}
	}
	return s
}

func atoi(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return -1
	}
	return n
}
