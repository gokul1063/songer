package queue

import (
	"songer-v3/internal/workflow"
	"songer-v3/service/youtube"
)

func PreloadNext() {
	workflow.Enter("QueuePreloader")
	defer workflow.Exit("QueuePreloader", "done")

	state := GetState()

	limit := 3
	if len(state.Upcoming) < limit {
		limit = len(state.Upcoming)
	}

	for i := 0; i < limit; i++ {
		videoID := state.Upcoming[i].VideoID
		println("video currently downloading:", videoID)
		youtube.DownloadAsync(videoID)
	}
}


func StartPreloader() {
	PreloadNext()
}
