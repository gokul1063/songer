package services

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/gokul1063/songer/configs"
	"github.com/gokul1063/songer/internal"
)

func startMVP() (*exec.Cmd, error) {
	const socketPath string = configs.SocketPath
	_ = os.Remove(socketPath)

	cmd := exec.Command(
		"mpv",
		"--no-video",
		"--input-ipc-server="+socketPath,
		"--idle=yes",
	)

	err := cmd.Start()
	if err != nil {
		internal.WriteLog("player.go ", err)
		return nil, err
	}

	for i := 0; i < 20; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return cmd, nil
}

func sendCommands(cmd string) error {
	conn, err := net.Dial("unix", "/tmp/mpv.sock")
	if err != nil {
		internal.WriteLog("sendCommands", err)
		return err
	}
	defer conn.Close()
	_, err = conn.Write([]byte(cmd + "\n"))
	if err != nil {
		internal.WriteLog("sendCommands", err)
		return err
	}
	return nil
}

func PlaySongTest1(songPath string) {
	_, err := startMVP()

	if err != nil {
		return
	}

	time.Sleep(100 * time.Millisecond)
	cmd := fmt.Sprintf(`{ "command": ["loadfile", "%s", "replace"] }`, songPath)
	sendCommands(cmd)

}
