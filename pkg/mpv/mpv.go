package mpv

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"songer/pkg/player"
)

type State = player.State

// audioClientName is the name songer's mpv reports to the audio server
// (--audio-client-name). It gives the stream a stable, unique identity so
// we can find it with pactl and so it never shares WirePlumber's per-client
// saved volume with a manually-launched "mpv".
const audioClientName = "songer"

type Player struct {
	ctx      context.Context
	cmd      *exec.Cmd
	socket   string
	conn     net.Conn
	mu       sync.Mutex
	state    State
	events   chan State
	closed   bool
	lastPush time.Time
}

// New returns a player that drives mpv over its JSON IPC socket.
func New(ctx context.Context, url, socket string, volume int) (*Player, error) {
	if _, err := exec.LookPath("mpv"); err != nil {
		return nil, fmt.Errorf("mpv not found: %w", err)
	}
	args := []string{
		"--no-video",
		"--idle=yes",
		"--terminal=no",
		"--no-input-default-bindings",
		"--input-ipc-server=" + socket,
		"--audio-client-name=" + audioClientName,
		"--volume=" + itoa(volume),
		url,
	}
	cmd := exec.CommandContext(ctx, "mpv", args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	p := &Player{
		ctx:      ctx,
		cmd:      cmd,
		socket:   socket,
		events:   make(chan State, 32),
		lastPush: time.Now(),
	}
	p.state = State{Volume: volume}
	return p, nil
}

func (p *Player) Events() <-chan State { return p.events }

func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *Player) Start() error {
	if err := p.cmd.Start(); err != nil {
		return err
	}

	// mpv creates the socket shortly after starting.
	deadline := time.Now().Add(8 * time.Second)
	for {
		if _, err := os.Stat(p.socket); err == nil {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("mpv ipc socket never appeared")
		}
		time.Sleep(50 * time.Millisecond)
	}

	conn, err := net.DialTimeout("unix", p.socket, 5*time.Second)
	if err != nil {
		return fmt.Errorf("dial mpv ipc: %w", err)
	}
	p.conn = conn

	for _, cmd := range []string{
		`{"command":["observe_property", 1, "pause"]}`,
		`{"command":["observe_property", 2, "time-pos"]}`,
		`{"command":["observe_property", 3, "duration"]}`,
		`{"command":["observe_property", 4, "volume"]}`,
		`{"command":["observe_property", 5, "media-title"]}`,
	} {
		if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
			return err
		}
	}

	go p.readLoop()

	// Some mpv builds (0.40 with ao_pipewire) keep the `volume` property as a
	// decoupled software value and let the audio server's per-client restore
	// own the real stream volume; if that restore is 0 the song plays silently
	// even though mpv reports a healthy volume. Force it to match below.
	p.mu.Lock()
	vol := p.state.Volume
	p.mu.Unlock()
	p.ensureStreamVolume(vol)
	return nil
}

func (p *Player) readLoop() {
	defer p.conn.Close()
	sc := bufio.NewScanner(p.conn)
	sc.Buffer(make([]byte, 64*1024), 64*1024)
	for sc.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			continue
		}
		switch msg["event"] {
		case "property-change":
			p.handleProperty(msg)
		case "end-file":
			if msg["reason"] == "eof" {
				p.mu.Lock()
				p.state.Ended = true
				p.mu.Unlock()
				p.push()
			}
		}
	}
}

func (p *Player) handleProperty(msg map[string]any) {
	name, _ := msg["name"].(string)
	var changed bool
	p.mu.Lock()
	switch name {
	case "pause":
		if b, ok := msg["data"].(bool); ok && b != p.state.Paused {
			p.state.Paused = b
			changed = true
		}
	case "time-pos":
		if f, ok := msg["data"].(float64); ok {
			d := time.Duration(f * float64(time.Second))
			if d != p.state.Position {
				p.state.Position = d
				changed = true
			}
		}
	case "duration":
		if f, ok := msg["data"].(float64); ok {
			d := time.Duration(f * float64(time.Second))
			if d != p.state.Duration {
				p.state.Duration = d
				changed = true
			}
		}
	case "volume":
		if f, ok := msg["data"].(float64); ok {
			v := int(f)
			if v != p.state.Volume {
				p.state.Volume = v
				changed = true
			}
		}
	case "media-title":
		if s, ok := msg["data"].(string); ok && s != p.state.Title {
			p.state.Title = s
			changed = true
		}
	}
	p.mu.Unlock()
	if changed {
		p.push()
	}
}

// push notifies the TUI, throttling position-only updates to ~4/sec.
// Ended (end-of-file) is never throttled so auto-advance is never lost.
func (p *Player) push() {
	now := time.Now()
	p.mu.Lock()
	ended := p.state.Ended
	throttle := !ended && now.Sub(p.lastPush) < 250*time.Millisecond
	p.mu.Unlock()
	if throttle {
		return
	}
	p.mu.Lock()
	p.lastPush = now
	s := p.state
	p.mu.Unlock()

	select {
	case p.events <- s:
	default:
	}
}

func (p *Player) send(cmd map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn == nil {
		return fmt.Errorf("mpv ipc not connected")
	}
	b, _ := json.Marshal(cmd)
	_, err := p.conn.Write(append(b, '\n'))
	return err
}

func (p *Player) Seek(d time.Duration) {
	_ = p.send(map[string]any{"command": []any{"seek", d.Seconds(), "relative", "exact"}})
}

func (p *Player) SetPause(paused bool) {
	_ = p.send(map[string]any{"command": []any{"set_property", "pause", paused}})
}

func (p *Player) TogglePause() {
	_ = p.send(map[string]any{"command": []any{"cycle", "pause"}})
}

func (p *Player) SetVolume(v int) {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	_ = p.send(map[string]any{"command": []any{"set_property", "volume", v}})
	p.applyStreamVolume(v)
	p.mu.Lock()
	p.state.Volume = v
	p.mu.Unlock()
}

// Load replaces the current media with url (used for "next").
func (p *Player) Load(url string) {
	p.mu.Lock()
	p.state.Ended = false
	p.mu.Unlock()
	_ = p.send(map[string]any{"command": []any{"loadfile", url, "replace"}})
}

// EnsureStreamVolume retries until songer's mpv stream shows up in the audio
// server, then sets its volume. Best-effort: returns when pactl is missing or
// no matching stream appears within a few seconds.
func EnsureStreamVolume(v int) {
	deadline := time.Now().Add(3 * time.Second)
	for {
		if ApplyStreamVolume(v) {
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
}

// ApplyStreamVolume sets the volume of songer's PulseAudio/PipeWire stream so
// the audio that reaches the speakers matches the requested level. mpv's own
// `volume` property is best-effort only: on some builds (ao_pipewire) it is a
// decoupled software value overridden by the audio server's per-client
// restore, and a restored 0% means literal silence. No-op (and harmless) when
// pactl or a matching stream is absent, e.g. on a plain ALSA setup.
func ApplyStreamVolume(v int) bool {
	if _, err := exec.LookPath("pactl"); err != nil {
		return true
	}
	id, ok := findSinkInput()
	if !ok {
		return false
	}
	_ = exec.Command("pactl", "set-sink-input-volume", strconv.Itoa(id), fmt.Sprintf("%d%%", v)).Run()
	return true
}

// ensureStreamVolume retries until mpv's audio-server stream appears, then
// sets its volume. Called once at startup (the sink-input may not exist yet
// the instant the IPC socket is up).
func (p *Player) ensureStreamVolume(v int) {
	EnsureStreamVolume(v)
}

// applyStreamVolume sets the volume of mpv's PulseAudio/PipeWire sink-input.
func (p *Player) applyStreamVolume(v int) bool {
	return ApplyStreamVolume(v)
}

// findSinkInput locates the sink-input id of songer's mpv stream by scanning
// `pactl list sink-inputs` for application.name == songer (set via
// --audio-client-name). mpv does not advertise its pid to PulseAudio, so the
// client name is the only stable handle.
func findSinkInput() (int, bool) {
	out, err := exec.Command("pactl", "list", "sink-inputs").Output()
	if err != nil {
		return 0, false
	}
	cur := -1
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if id, ok := parseSinkInputID(line); ok {
			cur = id
			continue
		}
		if cur >= 0 && strings.HasPrefix(line, "application.name") && strings.Contains(line, audioClientName) {
			return cur, true
		}
	}
	return 0, false
}

func parseSinkInputID(line string) (int, bool) {
	const pre = "Sink Input #"
	if !strings.HasPrefix(line, pre) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(line[len(pre):]))
	return n, err == nil
}

func (p *Player) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	if p.conn != nil {
		p.conn.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		_, _ = p.cmd.Process.Wait()
	}
	_ = os.Remove(p.socket)
	close(p.events)
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
