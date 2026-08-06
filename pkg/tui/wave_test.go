package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/mattn/go-runewidth"
)

func TestFmtDur(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0:00"},
		{73 * time.Second, "1:13"},
		{148 * time.Second, "2:28"},
		{3723 * time.Second, "1:02:03"},
		{-5 * time.Second, "0:00"},
	}
	for _, c := range cases {
		if got := fmtDur(c.d); got != c.want {
			t.Errorf("fmtDur(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestNextWaveBounds(t *testing.T) {
	w := nextWave(30)
	if len(w) != 30 {
		t.Fatalf("expected 30 bars, got %d", len(w))
	}
	for _, h := range w {
		if h < 0 || h > 1 {
			t.Fatalf("height %f out of range", h)
		}
	}
}

func TestWaveViewDims(t *testing.T) {
	v := waveView(nextWave(20), 5, "#00e5ff", "#00ff87")
	lines := strings.Split(v, "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 rows, got %d", len(lines))
	}
	for _, l := range lines {
		if w := runewidth.StringWidth(l); w != 20 {
			t.Fatalf("expected 20 visual cols, got %d (%q)", w, l)
		}
	}
}

func TestProgressBarLength(t *testing.T) {
	b := progressBar(30*time.Second, 120*time.Second, 20, "#0f0", "#333", "#ff0")
	if w := runewidth.StringWidth(b); w != 20 {
		t.Fatalf("expected 20 visual chars, got %d (%q)", w, b)
	}
	empty := progressBar(0, 0, 10, "#0f0", "#333", "#ff0")
	if w := runewidth.StringWidth(empty); w != 10 {
		t.Fatalf("expected 10 visual chars for no duration, got %d", w)
	}
}

func TestLerpColor(t *testing.T) {
	if got := lerpColor("#000000", "#ffffff", 0.5); got != "#7f7f7f" {
		t.Fatalf("lerp mid = %q, want #7f7f7f", got)
	}
	if got := lerpColor("#00ff87", "#00e5ff", 0); got != "#00ff87" {
		t.Fatalf("lerp 0 = %q, want start", got)
	}
	if got := lerpColor("#00ff87", "#00e5ff", 1); got != "#00e5ff" {
		t.Fatalf("lerp 1 = %q, want end", got)
	}
}

func TestWaveCount(t *testing.T) {
	if c := waveCount(56); c != 48 {
		t.Errorf("waveCount(56) = %d, want 48", c)
	}
	if c := waveCount(10); c < 4 {
		t.Errorf("waveCount(10) = %d, want >= 4", c)
	}
}
