package models

type Song struct {
	ID       string
	Title    string
	Author   string
	Path     string
	Duration int
	Tags     []string
}

type Playlist struct {
	ID    string
	Name  string
	Songs []Song
}

type History struct {
	SongID    string
	Timestamp int64
}

type SearchResult struct {
	Title  string
	Author string
	URL    string
}
