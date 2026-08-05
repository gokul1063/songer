package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const waveInterval = 90 * time.Millisecond

// truncate shortens s to at most width terminal cells.
func truncate(s string, width int) string {
	if width < 1 {
		return ""
	}
	if runewidth.StringWidth(s) <= width {
		return s
	}
	return runewidth.Truncate(s, width, "…")
}

// nextWave generates bar heights (0..1) for a lively equalizer strip.
func nextWave(n int) []float64 {
	if n < 1 {
		n = 1
	}
	now := float64(time.Now().UnixMilli())
	heights := make([]float64, n)
	for i := range heights {
		v := 0.5 + 0.5*math.Sin(now/140+float64(i)*0.55)
		v += 0.3 * math.Sin(now/60+float64(i)*1.3)
		v += 0.15 * math.Sin(now/17+float64(i)*3.1)
		h := 0.25 + 0.75*clamp(v, 0, 1)
		heights[i] = h
	}
	return heights
}

func waveView(heights []float64, rows int, color string) string {
	if len(heights) == 0 || rows < 1 {
		return ""
	}
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("█")
	var b strings.Builder
	for r := rows; r >= 1; r-- {
		for _, h := range heights {
			if int(math.Round(h*float64(rows))) >= r {
				b.WriteString(bar)
			} else {
				b.WriteByte(' ')
			}
		}
		if r > 1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func progressBar(pos, total time.Duration, width int, doneColor, trackColor string) string {
	if width < 1 {
		return ""
	}
	pct := 0.0
	if total > 0 {
		pct = pos.Seconds() / total.Seconds()
	}
	if pct > 1 {
		pct = 1
	}
	if pct < 0 {
		pct = 0
	}
	filled := int(math.Round(pct * float64(width)))
	done := lipgloss.NewStyle().Foreground(lipgloss.Color(doneColor)).Render(strings.Repeat("█", filled))
	track := lipgloss.NewStyle().Foreground(lipgloss.Color(trackColor)).Render(strings.Repeat("░", width-filled))
	return done + track
}

func fmtDur(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	s := int(d.Seconds())
	h := s / 3600
	m := (s % 3600) / 60
	sec := s % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%d:%02d", m, sec)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
