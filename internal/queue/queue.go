package queue

import (
	"errors"
	"songer/internal/player"
	"songer/internal/resolver"
	"songer/internal/youtube"
)

type Queue struct {
	items   []youtube.Video
	index   int
	player  *player.Player
	baseDir string
}

func New(p *player.Player, baseDir string) *Queue {

	return &Queue{
		items:   []youtube.Video{},
		index:   -1,
		player:  p,
		baseDir: baseDir,
	}
}

func (q *Queue) Add(video youtube.Video) {
	q.items = append(q.items, video)
}

func (q *Queue) PlayCurrent() error {

	if q.index < 0 || q.index >= len(q.items) {
		return errors.New("queue empty")
	}

	video := q.items[q.index]

	meta, err := resolver.Resolve(q.baseDir, video)
	if err != nil {
		return err
	}

	return q.player.Play(meta.Path)
}

func (q *Queue) Next() error {

	if len(q.items) == 0 {
		return errors.New("queue empty")
	}

	q.index++

	if q.index >= len(q.items) {
		q.index = len(q.items) - 1
		return errors.New("end of queue")
	}

	return q.PlayCurrent()
}

func (q *Queue) Prev() error {

	if len(q.items) == 0 {
		return errors.New("queue empty")
	}

	q.index--

	if q.index < 0 {
		q.index = 0
		return errors.New("start of queue")
	}

	return q.PlayCurrent()
}

func (q *Queue) Start() error {

	if len(q.items) == 0 {
		return errors.New("queue empty")
	}

	q.index = 0

	return q.PlayCurrent()
}

func (q *Queue) Current() (youtube.Video, error) {

	if q.index < 0 || q.index >= len(q.items) {
		return youtube.Video{}, errors.New("no current song")
	}

	return q.items[q.index], nil
}
