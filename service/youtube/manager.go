package youtube

import (
	"sync"

	"songer-v3/internal/workflow"
	"songer-v3/internal/logger"
)

var (
	maxWorkers = 3
	sem        = make(chan struct{}, maxWorkers)
	wg         sync.WaitGroup

	downloading = make(map[string]bool)
	mu          sync.Mutex
)

func DownloadAsync(videoID string) {
	mu.Lock()
	if downloading[videoID] {
		mu.Unlock()
		return
	}
	downloading[videoID] = true
	mu.Unlock()

	wg.Add(1)

	go func(id string) {
		defer wg.Done()

		sem <- struct{}{}
		defer func() { <-sem }()

		workflow.Enter("DownloadAsync:" + id)
		defer workflow.Exit("DownloadAsync:"+id, "done")

		println("starting download:", id)
		workflow.Enter("YTDownloadInner:" + id)
		_, err := Download(id)
		workflow.Exit("YTDownloadInner:"+id, "done")
		println("finished download:", id)

		if err != nil {
			logger.LogError(err)
			return
		}
		
		mu.Lock()
		delete(downloading, id)
		mu.Unlock()
	}(videoID)
}

func WaitAll() {
	wg.Wait()
}
