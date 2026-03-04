package service

import (
	"songer/internal/history"
	"songer/internal/player"
	"songer/internal/queue"
	"songer/internal/resolver"
	"songer/internal/youtube"
)

type Service struct {
	Player      *player.Player
	Queue       *queue.Queue
	BaseDir     string
	HistoryFile string
}

func New(socket string, baseDir string) (*Service, error) {

	p := player.NewPlayer(socket)

	err := p.Start()
	if err != nil {
		return nil, err
	}

	q := queue.New(p, baseDir)

	s := &Service{
		Player:      p,
		Queue:       q,
		BaseDir:     baseDir,
		HistoryFile: baseDir + "/history.json",
	}

	return s, nil
}

func (s *Service) Search(query string) ([]youtube.Video, error) {
	return youtube.Search(query)
}

func (s *Service) Add(video youtube.Video) {
	s.Queue.Add(video)
}

func (s *Service) Start() error {
	return s.Queue.Start()
}

func (s *Service) Next() error {
	return s.Queue.Next()
}

func (s *Service) Prev() error {
	return s.Queue.Prev()
}

func (s *Service) preloadNext() {

	next, err := s.Queue.PeekNext()
	if err != nil {
		return
	}

	go func() {

		_, err := resolver.Resolve(s.BaseDir, next)
		if err != nil {
			return
		}

	}()

}

func (s *Service) Listen() {

	s.Player.ListenEvents(func(e player.Event) {

		switch e.Event {

		case "file-loaded":

			video, err := s.Queue.Current()
			if err == nil {

				history.Append(
					s.HistoryFile,
					video.ID,
					video.Title,
				)

				s.preloadNext()
			}

		case "end-file":

			s.Queue.Next()

		}

	})
}
