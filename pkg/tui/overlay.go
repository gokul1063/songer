package tui

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/ansi"
)

// overlayWindow paints fg (a small window) over bg, with its top-left at
// (x, y). It is ANSI-aware: escape sequences consume no width and are carried
// with the side they belong to, and the uncovered tail keeps the base color.
func overlayWindow(bg, fg string, x, y int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")
	fgW := 0
	for _, l := range fgLines {
		if w := ansi.PrintableRuneWidth(l); w > fgW {
			fgW = w
		}
	}
	if x < 0 {
		x = 0
	}

	for i, fl := range fgLines {
		bi := y + i
		if bi < 0 || bi >= len(bgLines) {
			continue
		}
		base := bgLines[bi]
		left, rest := cutWidth(base, x)
		_, right := cutWidth(rest, fgW)
		out := left + fl
		if right != "" {
			out += leadingSGR(base) + right
		}
		bgLines[bi] = out
	}
	return strings.Join(bgLines, "\n")
}

// cutWidth splits s into (before, after) at `width` printable cells,
// keeping escape sequences with whichever side they belong to.
func cutWidth(s string, width int) (string, string) {
	var before, after strings.Builder
	w := 0
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\x1b' {
			seq, n := readEscape(string(runes[i:]))
			if w < width {
				before.WriteString(seq)
			} else {
				after.WriteString(seq)
			}
			i += n - 1
			continue
		}
		ww := runewidth.RuneWidth(r)
		if w < width {
			if w+ww > width {
				after.WriteRune(r)
			} else {
				before.WriteRune(r)
				w += ww
			}
		} else {
			after.WriteRune(r)
		}
	}
	return before.String(), after.String()
}

// leadingSGR returns the escape sequences at the very start of s (e.g. the
// opening color codes of a styled line).
func leadingSGR(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); {
		if runes[i] != '\x1b' {
			break
		}
		seq, n := readEscape(string(runes[i:]))
		b.WriteString(seq)
		i += n
	}
	return b.String()
}

// readEscape consumes one ANSI escape sequence starting at s[0] (an ESC byte)
// and returns it with its length in runes.
func readEscape(s string) (string, int) {
	if len(s) < 2 {
		return s, len(s)
	}
	switch s[1] {
	case '[': // CSI ... final byte
		i := 2
		for i < len(s) && !(s[i] >= 0x40 && s[i] <= 0x7e) {
			i++
		}
		if i < len(s) {
			i++
		}
		return s[:i], i
	case ']': // OSC ... BEL or ESC\
		i := 2
		for i < len(s) {
			if s[i] == '\x07' {
				i++
				break
			}
			if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\' {
				i += 2
				break
			}
			i++
		}
		return s[:i], i
	default:
		if len(s) >= 2 {
			return s[:2], 2
		}
		return s, len(s)
	}
}
