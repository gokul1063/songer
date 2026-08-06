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

// waveView renders the equalizer with a horizontal gradient from->to color.
func waveView(heights []float64, rows int, from, to string) string {
	if len(heights) == 0 || rows < 1 {
		return ""
	}
	var b strings.Builder
	for r := rows; r >= 1; r-- {
		for i, h := range heights {
			if int(math.Round(h*float64(rows))) >= r {
				t := 0.0
				if len(heights) > 1 {
					t = float64(i) / float64(len(heights)-1)
				}
				c := lerpColor(from, to, t)
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render("█"))
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

// progressBar draws a seekbar with a moving thumb: ━━━━●────
func progressBar(pos, total time.Duration, width int, doneColor, trackColor, thumbColor string) string {
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
	if filled > width-1 {
		filled = width - 1
	}
	track := width - filled - 1

	done := lipgloss.NewStyle().Foreground(lipgloss.Color(doneColor)).Render(strings.Repeat("━", filled))
	thumb := lipgloss.NewStyle().Foreground(lipgloss.Color(thumbColor)).Render("●")
	rest := lipgloss.NewStyle().Foreground(lipgloss.Color(trackColor)).Render(strings.Repeat("─", track))
	return done + thumb + rest
}

// lerpColor interpolates between two hex colors at t in [0,1].
func lerpColor(a, b string, t float64) string {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	ar, ag, ab := parseHex(a)
	br, bg, bb := parseHex(b)
	r := int(float64(ar) + (float64(br)-float64(ar))*t)
	g := int(float64(ag) + (float64(bg)-float64(ag))*t)
	bl := int(float64(ab) + (float64(bb)-float64(ab))*t)
	return fmt.Sprintf("#%02x%02x%02x", clampInt(r, 0, 255), clampInt(g, 0, 255), clampInt(bl, 0, 255))
}

func parseHex(c string) (int, int, int) {
	c = strings.TrimPrefix(strings.TrimSpace(c), "#")
	if len(c) != 6 {
		return 255, 255, 255
	}
	var v uint64
	if _, err := fmt.Sscanf(c, "%x", &v); err != nil {
		return 255, 255, 255
	}
	return int(v >> 16 & 0xff), int(v >> 8 & 0xff), int(v & 0xff)
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

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
