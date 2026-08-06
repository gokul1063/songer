package library

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"songer/pkg/search"
)

const (
	historyCap = 50
	fileName   = "library.json"
)

// Entry is a saved video with the time it was added/played.
type Entry struct {
	Video search.Video `json:"video"`
	At    time.Time    `json:"at"`
}

// Library is the persisted collection: favorites, liked, playlists, history.
type Library struct {
	Favorites []Entry            `json:"favorites"`
	Liked     []Entry            `json:"liked"`
	Playlists map[string][]Entry `json:"playlists"`
	History   []Entry            `json:"history"`

	path string
}

// DefaultPath returns ~/.config/songer/library.json.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "songer", fileName), nil
}

// Load reads the library from the default path, starting empty if missing.
func Load() (*Library, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return Open(path)
}

// Open reads the library from an explicit path.
func Open(path string) (*Library, error) {
	l := &Library{path: path, Playlists: map[string][]Entry{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return l, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, l); err != nil {
		return nil, err
	}
	if l.Playlists == nil {
		l.Playlists = map[string][]Entry{}
	}
	return l, nil
}

func (l *Library) Save() error {
	if l.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, data, 0o644)
}

func (l *Library) AddFavorite(v search.Video) error {
	l.Favorites = upsert(l.Favorites, v)
	return l.Save()
}

func (l *Library) RemoveFavorite(id string) error {
	l.Favorites = remove(l.Favorites, id)
	return l.Save()
}

func (l *Library) IsFavorite(id string) bool {
	return contains(l.Favorites, id)
}

func (l *Library) AddLiked(v search.Video) error {
	l.Liked = upsert(l.Liked, v)
	return l.Save()
}

func (l *Library) RemoveLiked(id string) error {
	l.Liked = remove(l.Liked, id)
	return l.Save()
}

func (l *Library) IsLiked(id string) bool {
	return contains(l.Liked, id)
}

func (l *Library) AddToPlaylist(name string, v search.Video) error {
	l.Playlists[name] = upsert(l.Playlists[name], v)
	return l.Save()
}

func (l *Library) PlaylistNames() []string {
	names := make([]string, 0, len(l.Playlists))
	for n := range l.Playlists {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (l *Library) Playlist(name string) []Entry {
	return l.Playlists[name]
}

// RecordPlay moves v to the front of history and caps the list.
func (l *Library) RecordPlay(v search.Video) error {
	if v.ID == "" {
		return nil
	}
	l.History = upsert(l.History, v)
	if len(l.History) > historyCap {
		l.History = l.History[:historyCap]
	}
	return l.Save()
}

func upsert(entries []Entry, v search.Video) []Entry {
	for i, e := range entries {
		if e.Video.ID == v.ID {
			entries[i].Video = v
			return entries
		}
	}
	return append([]Entry{{Video: v, At: time.Now()}}, entries...)
}

func remove(entries []Entry, id string) []Entry {
	out := entries[:0]
	for _, e := range entries {
		if e.Video.ID != id {
			out = append(out, e)
		}
	}
	return out
}

func contains(entries []Entry, id string) bool {
	for _, e := range entries {
		if e.Video.ID == id {
			return true
		}
	}
	return false
}
