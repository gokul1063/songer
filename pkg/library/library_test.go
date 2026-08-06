package library

import (
	"path/filepath"
	"strconv"
	"testing"

	"songer/pkg/search"
)

func video(id string) search.Video {
	return search.Video{ID: id, Title: "Title " + id, URL: "https://youtu.be/" + id}
}

func TestFavoriteLikedPlaylist(t *testing.T) {
	dir := t.TempDir()
	l, err := Open(filepath.Join(dir, "library.json"))
	if err != nil {
		t.Fatal(err)
	}

	v1, v2 := video("a"), video("b")
	if err := l.AddFavorite(v1); err != nil {
		t.Fatal(err)
	}
	if err := l.AddFavorite(v2); err != nil {
		t.Fatal(err)
	}
	if !l.IsFavorite("a") || !l.IsFavorite("b") {
		t.Fatal("favorites not persisted")
	}
	// duplicate is fine, no growth
	if err := l.AddFavorite(v1); err != nil {
		t.Fatal(err)
	}
	if len(l.Favorites) != 2 {
		t.Fatalf("favorites dup: got %d", len(l.Favorites))
	}
	if err := l.RemoveFavorite("a"); err != nil {
		t.Fatal(err)
	}
	if l.IsFavorite("a") {
		t.Fatal("remove failed")
	}

	if err := l.AddLiked(v1); err != nil {
		t.Fatal(err)
	}
	if !l.IsLiked("a") {
		t.Fatal("liked not persisted")
	}

	if err := l.AddToPlaylist("chill", v1); err != nil {
		t.Fatal(err)
	}
	if err := l.AddToPlaylist("chill", v2); err != nil {
		t.Fatal(err)
	}
	if err := l.AddToPlaylist("focus", v2); err != nil {
		t.Fatal(err)
	}
	names := l.PlaylistNames()
	if len(names) != 2 || names[0] != "chill" || names[1] != "focus" {
		t.Fatalf("playlist names: %v", names)
	}
	if got := len(l.Playlist("chill")); got != 2 {
		t.Fatalf("chill has %d entries, want 2", got)
	}
}

func TestHistoryCappedAndPersisted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "library.json")
	l, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < historyCap+5; i++ {
		id := string(rune('a'+i%26)) + "_" + strconv.Itoa(i)
		if err := l.RecordPlay(video(id)); err != nil {
			t.Fatal(err)
		}
	}
	if len(l.History) != historyCap {
		t.Fatalf("history len %d, want %d", len(l.History), historyCap)
	}
	// most recent first
	i := historyCap + 4
	wantID := string(rune('a'+i%26)) + "_" + strconv.Itoa(i)
	if l.History[0].Video.ID != wantID {
		t.Fatalf("history head %q, want %q", l.History[0].Video.ID, wantID)
	}

	// re-open and confirm persistence
	l2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(l2.Favorites) != 0 || len(l2.History) != historyCap {
		t.Fatal("persistence failed")
	}
}
