package player

import (
	"bufio"
	"encoding/json"
	"net"

	"songer-v3/internal/workflow"
)

type MpvEvent struct {
	Event string `json:"event"`
}

var onEnd func()

func StartEventListener() {
	workflow.Enter("MPVListener")
	defer workflow.Exit("MPVListener", "done")

	for {
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			continue
		}

		reader := bufio.NewReader(conn)

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				break
			}

			var evt MpvEvent
			err = json.Unmarshal(line, &evt)
			if err != nil {
				continue
			}

			if evt.Event == "end-file" {
				handleEndOfFile()
			}
		}

		conn.Close()
	}

}

func handleEndOfFile() {
	workflow.Enter("MPVEndFile")
	defer workflow.Exit("MPVEndFile", "done")

	if onEnd != nil {
		onEnd()
	}
}


func SetOnEndCallback(f func()) {
	onEnd = f
}
