package player

import (
	"encoding/json"
	"errors"
	"net"
	"os/exec"
	"sync"
	"time"
)

type Player struct {
	socketPath string
	conn       net.Conn
	cmd        *exec.Cmd
	mu         sync.Mutex
}

func NewPlayer(socket string) *Player {
	return &Player{
		socketPath: socket,
	}
}

func (p *Player) Start() error {
	p.cmd = exec.Command(
		"mpv",
		"--idle=yes",
		"--no-terminal",
		"--force-window=no",
		"--vo=null",
		"--input-ipc-server="+p.socketPath,
	)

	err := p.cmd.Start()
	if err != nil {
		return err
	}

	for i := 0; i < 10; i++ {
		conn, err := net.Dial("unix", p.socketPath)
		if err == nil {
			p.conn = conn
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	conn, err := net.Dial("unix", p.socketPath)
	if err != nil {
		return err
	}

	p.conn = conn
	return nil
}

func (p *Player) send(command []interface{}) error {

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return errors.New("player not connected")
	}

	req := map[string]interface{}{
		"command": command,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	_, err = p.conn.Write(data)
	return err
}

func (p *Player) Play(path string) error {
	return p.send([]interface{}{"loadfile", path, "replace"})
}

func (p *Player) Pause() error {
	return p.send([]interface{}{"cycle", "pause"})
}

func (p *Player) Stop() error {
	return p.send([]interface{}{"stop"})
}

func (p *Player) Seek(seconds int) error {
	return p.send([]interface{}{"seek", seconds, "relative"})
}
