package tui

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/ansi"

	"songer/pkg/config"
	"songer/pkg/library"
	"songer/pkg/mpv"
	"songer/pkg/search"
)

type fakePlayer struct {
	loaded string
	seeked []time.Duration
}

func (f *fakePlayer) Load(url string)       { f.loaded = url }
func (f *fakePlayer) TogglePause()          {}
func (f *fakePlayer) Seek(d time.Duration)  { f.seeked = append(f.seeked, d) }
func (f *fakePlayer) SetVolume(v int)       {}

func testModel(upcoming []search.Video, idx int) Model {
	return Model{
		player:   &fakePlayer{},
		ctx:      context.Background(),
		upcoming: upcoming,
		queueIdx: idx,
		focus:    focusQueue,
		perNode:  2,
		depth:    2,
	}
}

func TestDeleteFocused(t *testing.T) {
	m := testModel([]search.Video{{ID: "a"}, {ID: "b"}, {ID: "c"}}, 1)
	updated, _ := m.deleteFocused()
	nm := updated.(Model)
	if len(nm.upcoming) != 2 || nm.upcoming[0].ID != "a" || nm.upcoming[1].ID != "c" {
		t.Fatalf("bad delete: %+v", nm.upcoming)
	}
	if nm.queueIdx != 1 {
		t.Fatalf("queueIdx = %d, want 1", nm.queueIdx)
	}
}

func TestDeleteLastFocused(t *testing.T) {
	m := testModel([]search.Video{{ID: "a"}, {ID: "b"}}, 1)
	updated, _ := m.deleteFocused()
	nm := updated.(Model)
	if len(nm.upcoming) != 1 || nm.queueIdx != 0 {
		t.Fatalf("deleting last item should clamp idx: %+v idx=%d", nm.upcoming, nm.queueIdx)
	}
}

func TestMoveQueue(t *testing.T) {
	m := testModel([]search.Video{{ID: "a"}, {ID: "b"}, {ID: "c"}}, 0)
	updated, _ := m.moveQueue(+1)
	nm := updated.(Model)
	if nm.upcoming[0].ID != "b" || nm.upcoming[1].ID != "a" {
		t.Fatalf("move down failed: %+v", nm.upcoming)
	}
	if nm.queueIdx != 1 {
		t.Fatalf("queueIdx = %d, want 1", nm.queueIdx)
	}

	updated, _ = nm.moveQueue(-1)
	nm = updated.(Model)
	if nm.upcoming[0].ID != "a" || nm.queueIdx != 0 {
		t.Fatalf("move up failed: %+v", nm.upcoming)
	}

	// boundary: can't move first item up
	updated, _ = nm.moveQueue(-1)
	nm = updated.(Model)
	if nm.upcoming[0].ID != "a" {
		t.Fatalf("moved past start: %+v", nm.upcoming)
	}
}

func TestAdvanceOnEnd(t *testing.T) {
	fp := &fakePlayer{}
	m := testModel([]search.Video{{ID: "b", Title: "B", URL: "u2"}, {ID: "c", Title: "C", URL: "u3"}}, 0)
	m.player = fp
	m.current = search.Video{ID: "a", Title: "A"}

	updated, _ := m.Update(mpv.State{Ended: true})
	nm := updated.(Model)
	if nm.current.ID != "b" {
		t.Fatalf("expected current B, got %s", nm.current.ID)
	}
	// the played song is removed from the list entirely
	if len(nm.upcoming) != 1 || nm.upcoming[0].ID != "c" {
		t.Fatalf("unexpected queue after advance: %+v", nm.upcoming)
	}
	if fp.loaded != "u2" {
		t.Fatalf("expected mpv to load u2, got %q", fp.loaded)
	}
	if nm.state.Ended {
		t.Fatal("Ended should be cleared after advancing")
	}
}

func TestPlayFromRemovesFromList(t *testing.T) {
	fp := &fakePlayer{}
	m := testModel([]search.Video{{ID: "a", Title: "A"}, {ID: "b", Title: "B"}, {ID: "c", Title: "C"}, {ID: "d", Title: "D"}}, 2)
	m.player = fp

	updated, _ := m.playFrom(2)
	nm := updated.(Model)
	got := []string{}
	for _, v := range nm.upcoming {
		got = append(got, v.ID)
	}
	want := []string{"a", "b", "d"}
	if !slices.Equal(got, want) {
		t.Fatalf("playFrom should remove c: got %v want %v", got, want)
	}
	if nm.current.ID != "c" {
		t.Fatalf("current = %s, want c", nm.current.ID)
	}
	if nm.queueIdx != 2 {
		t.Fatalf("queueIdx = %d, want 2 (item d shifted into place)", nm.queueIdx)
	}
}

func TestClampScroll(t *testing.T) {
	cases := []struct {
		scroll, idx, vis, total, want int
	}{
		{0, 0, 5, 60, 0},
		{0, 8, 5, 60, 4},  // idx 8 must fit in window of 5 -> scroll 4
		{10, 2, 5, 60, 2}, // idx above window -> scroll down to 2
		{0, 0, 5, 3, 0},   // total smaller than window
		{50, 59, 5, 60, 55},
		{0, 0, 5, 0, 0}, // empty
	}
	for _, c := range cases {
		if got := clampScroll(c.scroll, c.idx, c.vis, c.total); got != c.want {
			t.Errorf("clampScroll(%d,%d,%d,%d) = %d, want %d", c.scroll, c.idx, c.vis, c.total, got, c.want)
		}
	}
}

func TestListVisible(t *testing.T) {
	if v := listVisible(24); v != 5 {
		t.Errorf("listVisible(24) = %d, want 5", v)
	}
	if v := listVisible(8); v != 1 {
		t.Errorf("listVisible(8) = %d, want 1", v)
	}
	if v := listVisible(100); v > 24 {
		t.Errorf("listVisible(100) = %d, want capped at 24", v)
	}
}

func TestQueueCap(t *testing.T) {
	m := testModel(nil, 0)
	m.current = search.Video{ID: "current"}
	up := make([]search.Video, maxQueue)
	for i := range up {
		up[i] = search.Video{ID: fmt.Sprintf("id%d", i)}
	}
	m.upcoming = up

	more := make([]search.Video, 6)
	for i := range more {
		more[i] = search.Video{ID: fmt.Sprintf("new%d", i)}
	}
	updated, _ := m.Update(queueLoadedMsg{videos: more})
	if len(updated.(Model).upcoming) != maxQueue {
		t.Fatalf("expected cap at %d, got %d", maxQueue, len(updated.(Model).upcoming))
	}
}

func TestQueueKeysGatedByFocus(t *testing.T) {
	m := testModel([]search.Video{{ID: "a"}, {ID: "b"}}, 0)
	m.focus = focusMain
	d := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updated, _ := m.handleKey(d)
	if len(updated.(Model).upcoming) != 2 {
		t.Fatal("d should not delete when focus is main")
	}
	j := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ = m.handleKey(j)
	if updated.(Model).queueIdx != 0 {
		t.Fatal("j should not navigate when focus is main")
	}
}

func TestSeekKeysInMain(t *testing.T) {
	fp := &fakePlayer{}
	m := testModel(nil, 0)
	m.player = fp
	m.focus = focusMain

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if len(fp.seeked) != 1 || fp.seeked[0] != 5*time.Second {
		t.Fatalf("l in main should seek +5s, got %v", fp.seeked)
	}

	updated, _ = updated.(Model).handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if len(fp.seeked) != 2 || fp.seeked[1] != -5*time.Second {
		t.Fatalf("h in main should seek -5s, got %v", fp.seeked)
	}
}

func TestSeekKeysNotInQueue(t *testing.T) {
	fp := &fakePlayer{}
	m := testModel(nil, 0)
	m.player = fp
	m.focus = focusQueue

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if len(fp.seeked) != 0 {
		t.Fatalf("h/l should not seek when queue focused, got %v", fp.seeked)
	}
}

func TestToggleFavorite(t *testing.T) {
	lib, err := library.Open(filepath.Join(t.TempDir(), "lib.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := testModel(nil, 0)
	m.lib = lib
	m.current = search.Video{ID: "x", Title: "X"}

	updated, _ := m.toggleFavorite()
	nm := updated.(Model)
	if !nm.fav || !nm.lib.IsFavorite("x") {
		t.Fatal("favorite not added")
	}

	updated, _ = nm.toggleFavorite()
	nm = updated.(Model)
	if nm.fav || nm.lib.IsFavorite("x") {
		t.Fatal("favorite not removed")
	}
}

func TestPageToggle(t *testing.T) {
	m := testModel(nil, 0)
	if m.page != pageMain {
		t.Fatal("default page should be main")
	}
	// tab -> next
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
	if updated.(Model).page != pagePlaylist {
		t.Fatal("tab should go to playlist page")
	}
	// tab again -> wraps to main
	updated, _ = updated.(Model).handleKey(tea.KeyMsg{Type: tea.KeyTab})
	if updated.(Model).page != pageMain {
		t.Fatal("tab should wrap back to main")
	}
	// shift+tab -> previous
	updated, _ = updated.(Model).handleKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	if updated.(Model).page != pagePlaylist {
		t.Fatal("shift+tab should go to previous page")
	}
	// alt+tab -> previous
	updated, _ = updated.(Model).handleKey(tea.KeyMsg{Type: tea.KeyTab, Alt: true})
	if updated.(Model).page != pageMain {
		t.Fatal("alt+tab should go to previous page")
	}
}

func TestPlaylistView(t *testing.T) {
	lib, err := library.Open(filepath.Join(t.TempDir(), "lib.json"))
	if err != nil {
		t.Fatal(err)
	}
	_ = lib.AddToPlaylist("chill", search.Video{ID: "x", Title: "X"})
	m := testModel(nil, 0)
	m.lib = lib
	m.page = pagePlaylist

	out := m.playlistView(60, 18)
	for _, want := range []string{"★", "♥", "＋", "chill"} {
		if !strings.Contains(out, want) {
			t.Errorf("playlist view missing %q\n%s", want, out)
		}
	}
}

func TestPlaylistBoxSquare(t *testing.T) {
	th := config.Theme{Border: "#2a2a40", Selection: "#00ff87", Primary: "#00ff87", Accent: "#ffcc66", Secondary: "#00e5ff", Muted: "#6b7280"}
	box := playlistBox(playlistBoxData{symbol: "★", name: "Favorites", color: "#fff"}, true, th)
	lines := strings.Split(box, "\n")
	if len(lines) != ansi.PrintableRuneWidth(lines[0]) {
		t.Fatalf("box not square: %d lines x %d wide", len(lines), ansi.PrintableRuneWidth(lines[0]))
	}
}

func TestPlaylistNav(t *testing.T) {
	m := testModel(nil, 0)
	m.page = pagePlaylist
	m.width = 100

	// first square highlighted by default
	if m.plFocus != 0 {
		t.Fatalf("plFocus = %d, want 0", m.plFocus)
	}
	// l -> second square
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if updated.(Model).plFocus != 1 {
		t.Fatalf("plFocus after l = %d, want 1", updated.(Model).plFocus)
	}
	// enter the liked box
	updated, _ = updated.(Model).handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	nm := updated.(Model)
	if !nm.plList {
		t.Fatal("enter should open the list")
	}
	// backspace exits, focus stays on that square
	updated, _ = nm.handleKey(tea.KeyMsg{Type: tea.KeyBackspace})
	nm = updated.(Model)
	if nm.plList {
		t.Fatal("backspace should exit the list")
	}
	if nm.plFocus != 1 {
		t.Fatalf("plFocus after backspace = %d, want 1", nm.plFocus)
	}
}

func TestPlaylistEnterPlusNoop(t *testing.T) {
	m := testModel(nil, 0)
	m.page = pagePlaylist
	m.width = 100
	// move focus to the plus (last box)
	last := len(m.buildBoxes()) - 1
	m.plFocus = last
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.(Model).plList {
		t.Fatal("enter on + must do nothing")
	}
}

func TestTabName(t *testing.T) {
	m := testModel(nil, 0)
	if got := m.tabName(); got != "main" {
		t.Fatalf("main page tab name = %q", got)
	}
	m.page = pagePlaylist
	if got := m.tabName(); got != "playlists" {
		t.Fatalf("playlist page tab name = %q", got)
	}
	m.plList = true
	m.plFocus = 0
	if got := m.tabName(); got != "Favorites" {
		t.Fatalf("list tab name = %q", got)
	}

	// header actually renders the tab name top-right
	m.width = 100
	m.height = 24
	h := m.headerView(m.width)
	if !strings.Contains(h, "Favorites") {
		t.Fatalf("header missing tab name:\n%q", h)
	}
}

// TestHeaderSurvivesPlaylistPage guards against the header scrolling off
// screen: switching pages must never make the base view taller than the
// terminal.
func TestHeaderSurvivesPlaylistPage(t *testing.T) {
	for _, page := range []page{pageMain, pagePlaylist} {
		m := testModel(nil, 0)
		m.width = 100
		m.height = 24
		m.page = page
		out := m.baseView()
		lines := strings.Split(out, "\n")
		if len(lines) > m.height {
			t.Errorf("page %d: base view is %d rows, terminal is %d (header scrolled off)",
				page, len(lines), m.height)
		}
		if !strings.Contains(out, "SONGER") {
			t.Errorf("page %d: header missing", page)
		}
	}
}

// TestHeaderSurvivesAnySize renders the whole UI at a range of terminal sizes
// and asserts the header is always present and nothing overflows the screen.
func TestHeaderSurvivesAnySize(t *testing.T) {
	for _, w := range []int{60, 80, 100, 140, 200} {
		for _, h := range []int{8, 10, 12, 14, 18, 24, 32, 48} {
			for _, page := range []page{pageMain, pagePlaylist} {
				m := testModel(
					[]search.Video{{ID: "a", Title: "A"}, {ID: "b", Title: "B"}, {ID: "c", Title: "C"}},
					1,
				)
				m.width = w
				m.height = h
				m.page = page
				m.focus = focusMain
				out := m.View()
				lines := strings.Split(out, "\n")
				if len(lines) > h {
					t.Errorf("w=%d h=%d page=%d: view is %d rows (> terminal)", w, h, page, len(lines))
				}
				if !strings.Contains(out, "SONGER") {
					t.Errorf("w=%d h=%d page=%d: header missing", w, h, page)
				}
			}
		}
	}
}

func TestFocusSwitchKeys(t *testing.T) {
	m := testModel(nil, 0)
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlH})
	if updated.(Model).focus != focusMain {
		t.Fatal("ctrl+h should focus main")
	}
	updated, _ = updated.(Model).handleKey(tea.KeyMsg{Type: tea.KeyCtrlL})
	if updated.(Model).focus != focusQueue {
		t.Fatal("ctrl+l should focus queue")
	}
}

// TestRealPlayerAutoAdvance is an end-to-end check of the auto-play path:
// a real mpv instance playing a 1s local video emits end-of-file, the event
// loop forwards it, and the Model advances to the next queued song.
func TestRealPlayerAutoAdvance(t *testing.T) {
	if _, err := exec.LookPath("mpv"); err != nil {
		t.Skip("mpv not installed")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}

	dir := t.TempDir()
	gen := func(name string, dur string) string {
		p := filepath.Join(dir, name)
		cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "anullsrc=r=8000:cl=mono", "-t", dur, p)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("ffmpeg failed: %v %s", err, out)
		}
		return p
	}
	fileA := gen("a.mp4", "1")
	fileB := gen("b.mp4", "3")

	ctx := context.Background()
	player, err := mpv.New(ctx, fileA, filepath.Join(dir, "s.sock"), 50)
	if err != nil {
		t.Fatal(err)
	}
	if err := player.Start(); err != nil {
		t.Fatal(err)
	}
	defer player.Close()

	m := Model{
		player: player,
		ctx:    ctx,
		current: search.Video{ID: "a", Title: "A", URL: fileA},
		upcoming: []search.Video{{ID: "b", Title: "B", URL: fileB}},
		perNode: 2,
		depth:   2,
	}

	deadline := time.After(25 * time.Second)
	for {
		select {
		case st := <-player.Events():
			updated, _ := m.Update(st)
			m = updated.(Model)
			if m.current.ID == "b" {
				return
			}
		case <-deadline:
			t.Fatal("model never advanced to the next song")
		}
	}
}
