package cmus

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"

	"songer/pkg/player"
)

// pollInterval is how often we query cmus-remote -Q while a track plays.
const pollInterval = 300 * time.Millisecond

const startTimeout = 8 * time.Second

// Snapshot is the parsed output of `cmus-remote -Q`.
type Snapshot struct {
	State    string // "playing", "paused" or "stopped"
	Position int    // seconds
	Duration int    // seconds
	Volume   int    // 0-100
	Title    string
}

// Command sends a cmus command through cmus-remote. cmd is passed verbatim,
// so it must be a single cmus command line (e.g. "vol 80").
func Command(cmd string) error {
	_, err := remote("-C", cmd)
	return err
}

// Query returns a snapshot of the running cmus instance.
func Query() (Snapshot, error) {
	out, err := remote("-Q")
	if err != nil {
		return Snapshot{}, err
	}
	return parseStatus(out), nil
}

var (
	spawnMu   sync.Mutex
	spawned   *exec.Cmd
	spawnedPT *os.File
)

// EnsureRunning makes sure a cmus instance is reachable through cmus-remote.
// cmus is a curses app that needs a terminal, so when none is running songer
// spawns one on a private pty and leaves it there for cmus-remote to drive.
// It reports whether it started cmus itself.
func EnsureRunning(ctx context.Context) (bool, error) {
	if _, err := Query(); err == nil {
		return false, nil
	}
	spawnMu.Lock()
	defer spawnMu.Unlock()
	if _, err := Query(); err == nil {
		return false, nil
	}
	if _, err := exec.LookPath("cmus"); err != nil {
		return false, fmt.Errorf("cmus not found: %w", err)
	}

	// Clear any stale socket so the fresh cmus binds a clean one and
	// cmus-remote never talks to a dead server.
	if r := os.Getenv("XDG_RUNTIME_DIR"); r != "" {
		_ = os.Remove(filepath.Join(r, "cmus-socket"))
	}

	cmd := exec.Command("cmus")
	f, err := pty.Start(cmd)
	if err != nil {
		return false, fmt.Errorf("start cmus on a pty: %w", err)
	}
	go func() { _, _ = io.Copy(io.Discard, f) }()

	deadline := time.Now().Add(startTimeout)
	for {
		if q, err := Query(); err == nil && q.State != "" {
			spawned = cmd
			spawnedPT = f
			return true, nil
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			_ = f.Close()
			return false, fmt.Errorf("cmus started but its ipc socket never appeared")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// StopSpawned shuts down the cmus instance EnsureRunning started, if any.
func StopSpawned() {
	spawnMu.Lock()
	defer spawnMu.Unlock()
	if spawned != nil && spawned.Process != nil {
		_ = spawned.Process.Kill()
		_, _ = spawned.Process.Wait()
	}
	if spawnedPT != nil {
		_ = spawnedPT.Close()
	}
	if r := os.Getenv("XDG_RUNTIME_DIR"); r != "" {
		_ = os.Remove(filepath.Join(r, "cmus-socket"))
	}
	spawned = nil
	spawnedPT = nil
}

// Download fetches the best audio for url into a temp file cmus can play.
// Many cmus builds (Debian's in particular) ship no HTTP/streaming input
// plugin, so `add <stream-url>` fails; materializing the audio on disk and
// playing the local file works everywhere.
func Download(ctx context.Context, url string) (string, error) {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return "", fmt.Errorf("yt-dlp not found (needed to download audio for cmus): %w", err)
	}
	dir := filepath.Join(os.TempDir(), "songer-cmus")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	out := filepath.Join(dir, strconv.FormatInt(time.Now().UnixNano(), 36)+".%(ext)s")
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--no-playlist",
		"-f", "bestaudio[ext=m4a]/bestaudio/best",
		"-o", out,
		"--print", "after_move:filepath",
		url,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	path := strings.TrimSpace(lines[len(lines)-1])
	if path == "" || !exists(path) {
		return "", fmt.Errorf("yt-dlp produced no playable file")
	}
	return path, nil
}

// setCmd renders a cmus setting assignment. cmus 2.12 requires the "=" form
// (`set opt=value`); the space form errors with "no such option".
func setCmd(name, value string) string {
	return "set " + name + "=" + value
}

// setVolume makes cmus use its software mixer (so volume works without a
// hardware mixer, e.g. on headless boxes) and applies the given volume.
func setVolume(volume int) {
	_ = Command(setCmd("softvol", "true"))
	_ = Command(fmt.Sprintf("vol %d", volume))
}

// PlayFile stops any current playback and plays path directly via
// `cmus-remote -f`, then applies volume. The track plays alone: when it ends
// cmus returns to "stopped" (which the UI uses to auto-advance).
func PlayFile(path string, volume int) error {
	if volume < 1 {
		volume = 80
	}
	// Keep cmus from auto-playing its library or continuing after a track
	// ends: songer needs the "stopped" transition to drive its own queue.
	_ = Command(setCmd("continue", "false"))
	_ = Command(setCmd("play_library", "false"))
	if _, err := remote("-s"); err != nil {
		return fmt.Errorf("cmus-remote stop: %w", err)
	}
	if _, err := remote("-f", path); err != nil {
		return fmt.Errorf("cmus-remote play: %w", err)
	}
	setVolume(volume)
	return nil
}

// Player drives cmus through cmus-remote and exposes the same interface as the
// mpv controller so the TUI can use either backend.
type Player struct {
	ctx         context.Context
	events      chan player.State
	state       player.State
	mu          sync.Mutex
	prev        string
	closed      bool
	loadSeq     uint64
	stop        chan struct{}
	done        chan struct{}
	currentFile string
}

// New verifies cmus is reachable (starting it if needed) and returns a player
// ready to load tracks.
func New(ctx context.Context, volume int) (*Player, error) {
	if _, err := exec.LookPath("cmus-remote"); err != nil {
		return nil, fmt.Errorf("cmus-remote not found: %w", err)
	}
	spawned, err := EnsureRunning(ctx)
	if err != nil {
		return nil, err
	}
	q, err := Query()
	if err != nil {
		return nil, fmt.Errorf("cmus unreachable: %w", err)
	}
	if spawned {
		// a fresh cmus has volume 0; use the configured default so it is audible
		if volume < 1 {
			volume = 80
		}
	} else if q.Volume >= 0 {
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

// Start kicks off downloading url in the background and returns immediately;
// the TUI starts playing as soon as the audio is on disk.
func (p *Player) Start(url string) error {
	p.load(url)
	go p.pollLoop()
	return nil
}

// Load replaces the current media with url (used for "next"). The download
// runs in a background goroutine so the UI never blocks.
func (p *Player) Load(url string) {
	p.load(url)
}

// load resets the state and starts an async download+play of url. A sequence
// number guards against stale downloads racing ahead of newer ones.
func (p *Player) load(url string) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.loadSeq++
	seq := p.loadSeq
	p.state.Ended = false
	p.state.Position = 0
	p.state.Duration = 0
	p.state.Title = ""
	// stopping cmus to play the next track briefly reports "stopped"; treating
	// that as "ended" would make the UI skip a song, so reset the tracked state
	// until the new track plays.
	p.prev = "stopped"
	p.mu.Unlock()

	go p.doLoad(url, seq)
}

func (p *Player) doLoad(url string, seq uint64) {
	path, err := Download(p.ctx, url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmus download error: %v\n", err)
		return
	}
	p.mu.Lock()
	if p.closed || seq != p.loadSeq {
		p.mu.Unlock()
		_ = os.Remove(path)
		return
	}
	old := p.currentFile
	p.currentFile = path
	vol := p.state.Volume
	p.mu.Unlock()
	if old != "" && old != path {
		_ = os.Remove(old)
	}
	_ = PlayFile(path, vol)
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
	setVolume(v)
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
	file := p.currentFile
	p.mu.Unlock()

	close(p.stop)
	<-p.done
	if file != "" {
		_ = os.Remove(file)
	}
	_ = Command("player-stop")
	StopSpawned()
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
		case strings.HasPrefix(line, "position "):
			s.Position = atoi(line[len("position "):])
		case strings.HasPrefix(line, "filepos "):
			s.Position = atoi(line[len("filepos "):])
		case strings.HasPrefix(line, "duration "):
			s.Duration = atoi(line[len("duration "):])
		case strings.HasPrefix(line, "set vol_left "):
			s.Volume = atoi(line[len("set vol_left "):])
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

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
