package player

import (
	"bufio"
	"encoding/json"
)

type Event struct {
	Event string `json:"event"`
}

func (p *Player) ListenEvents(handler func(Event)) {

	go func() {

		reader := bufio.NewReader(p.conn)

		for {

			line, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}

			var ev Event

			err = json.Unmarshal(line, &ev)
			if err != nil {
				continue
			}

			if ev.Event != "" {
				handler(ev)
			}

		}

	}()
}
