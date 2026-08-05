package tui

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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
	if len(nm.upcoming) != 1 {
		t.Fatalf("expected 1 left in queue, got %d", len(nm.upcoming))
	}
	if fp.loaded != "u2" {
		t.Fatalf("expected mpv to load u2, got %q", fp.loaded)
	}
	if nm.state.Ended {
		t.Fatal("Ended should be cleared after advancing")
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
