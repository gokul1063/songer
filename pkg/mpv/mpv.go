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
	"sync"
	"time"
)

type State struct {
	Paused   bool
	Position time.Duration
	Duration time.Duration
	Volume   int
	Title    string
	Ended    bool
}

type Player struct {
	ctx       context.Context
	cmd       *exec.Cmd
	socket    string
	conn      net.Conn
	mu        sync.Mutex
	state     State
	events    chan State
	closed    bool
	lastPush  time.Time
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
