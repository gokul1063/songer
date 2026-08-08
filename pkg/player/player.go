package player

import "time"

// State is a snapshot of a media backend (mpv, cmus, ...). The TUI renders
// from these snapshots and reacts to Ended to auto-advance.
type State struct {
	Paused   bool
	Position time.Duration
	Duration time.Duration
	Volume   int
	Title    string
	Ended    bool
}

// Controller is the control surface the TUI drives on a backend.
type Controller interface {
	Load(url string)
	TogglePause()
	Seek(d time.Duration)
	SetVolume(v int)
}

// Player is everything the TUI needs from a backend: state events plus controls.
type Player interface {
	Controller
	Events() <-chan State
	State() State
}
