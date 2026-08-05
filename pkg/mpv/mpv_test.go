package mpv

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestNewMissingMpv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	_, err := New(context.Background(), "https://youtu.be/abc", filepath.Join(dir, "x.sock"), 80)
	if err == nil {
		t.Fatal("expected error when mpv is missing")
	}
}

func TestItoa(t *testing.T) {
	if got := itoa(80); got != "80" {
		t.Fatalf("itoa(80) = %q", got)
	}
}

// TestEndFileEvent plays a real 1s local video through mpv and asserts the
// Ended (end-of-file) event reaches the Events channel. This guards the
// auto-advance path: --idle=yes keeps mpv alive and ended pushes are not
// throttled.
func TestEndFileEvent(t *testing.T) {
	if _, err := exec.LookPath("mpv"); err != nil {
		t.Skip("mpv not installed")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}

	dir := t.TempDir()
	file := filepath.Join(dir, "tone.mp4")
	cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "anullsrc=r=8000:cl=mono", "-t", "1", file)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg failed: %v %s", err, out)
	}

	player, err := New(context.Background(), file, filepath.Join(dir, "s.sock"), 50)
	if err != nil {
		t.Fatal(err)
	}
	if err := player.Start(); err != nil {
		t.Fatal(err)
	}
	defer player.Close()

	deadline := time.After(20 * time.Second)
	for {
		select {
		case st := <-player.Events():
			if st.Ended {
				return
			}
		case <-deadline:
			t.Fatal("no end-file event within 20s")
		}
	}
}
