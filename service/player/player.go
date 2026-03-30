package player

import (
	"os"
	"os/exec"
	"time"
	"net"

	"songer-v3/internal/logger"
	"songer-v3/internal/workflow"
)

var socketPath = "/tmp/mpv_socket"

func Play(filePath string) error {
	workflow.Enter("PlayerPlay")
	defer workflow.Exit("PlayerPlay", "done")

	// remove old socket if exists
	_ = os.Remove(socketPath)

	cmd := exec.Command("mpv",
		"--no-video",
		"--input-ipc-server="+socketPath,
		filePath,
	)

	err := cmd.Start()
	if err != nil {
		logger.LogError(err)
		return err
	}

	time.Sleep(500 * time.Millisecond)

	return nil
}

func sendCommand(command string) error {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		logger.LogError(err)
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(command + "\n"))
	if err != nil {
		logger.LogError(err)
		return err
	}

	return nil
}

func Pause() error {
	workflow.Enter("PlayerPause")
	defer workflow.Exit("PlayerPause", "done")

	cmd := `{"command": ["set_property", "pause", true]}`
	return sendCommand(cmd)
}

func Resume() error {
	workflow.Enter("PlayerResume")
	defer workflow.Exit("PlayerResume", "done")

	cmd := `{"command": ["set_property", "pause", false]}`
	return sendCommand(cmd)
}

func Stop() error {
	workflow.Enter("PlayerStop")
	defer workflow.Exit("PlayerStop", "done")

	cmd := `{"command": ["quit"]}`
	return sendCommand(cmd)
}
