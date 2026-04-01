package queue

import (
	"time"
	"sync"

	"songer-v3/internal/logger"
	"songer-v3/internal/workflow"
	"songer-v3/model"
	"songer-v3/service/player"
	"songer-v3/service/youtube"
)

var q = &model.Queue{}
var mu sync.Mutex

func Add(song model.Song) {
	workflow.Enter("QueueAdd")
	defer workflow.Exit("QueueAdd", "done")

	q.Upcoming = append(q.Upcoming, song)
}

func PlayNext() {
	workflow.Enter("QueuePlayNext")
	defer workflow.Exit("QueuePlayNext", "done")

	mu.Lock()

	if len(q.Upcoming) == 0 {
		mu.Unlock()
		return
	}

	next := q.Upcoming[0]
	q.Upcoming = q.Upcoming[1:]

	if q.Current != nil {
		q.History = append(q.History, *q.Current)
		if len(q.History) > 30 {
			q.History = q.History[1:]
		}
	}

	q.Current = &next

	mu.Unlock()

	path, err := youtube.Download(next.VideoID)
	if err != nil {
		logger.LogError(err)
		return
	}

	err = player.Play(path)
	if err != nil {
		logger.LogError(err)
		return
	}
}
func StartAutoPlay() {
	workflow.Enter("QueueAutoPlay")
	defer workflow.Exit("QueueAutoPlay", "done")

	for {
		if q.Current == nil && len(q.Upcoming) > 0 {
			PlayNext()
		}

		time.Sleep(1 * time.Second)
	}
}

func GetState() *model.Queue {
	return q
}
