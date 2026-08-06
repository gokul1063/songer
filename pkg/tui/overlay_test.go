package tui

import (
	"strings"
	"testing"

	"github.com/muesli/reflow/ansi"
)

func TestCutWidthPlain(t *testing.T) {
	before, after := cutWidth("hello world", 5)
	if before != "hello" || after != " world" {
		t.Fatalf("cut = (%q, %q)", before, after)
	}
	before, after = cutWidth("ab", 10)
	if before != "ab" || after != "" {
		t.Fatalf("short cut = (%q, %q)", before, after)
	}
}

func TestCutWidthWide(t *testing.T) {
	before, after := cutWidth("a日b", 2)
	// 'a' (1) + 'b' (1) fit in 2 cells; the wide 日 is pushed whole to after.
	if before != "ab" || after != "日" {
		t.Fatalf("wide cut = (%q, %q)", before, after)
	}
	// a wide rune must never be split: cut at 2 leaves before at width 1.
	before, after = cutWidth("a日", 2)
	if before != "a" || after != "日" {
		t.Fatalf("wide cut 2 = (%q, %q)", before, after)
	}
}

func TestCutWidthANSI(t *testing.T) {
	s := "\x1b[38;2;0;255;135m" + "abc" + "\x1b[0m"
	before, after := cutWidth(s, 2)
	if ansi.PrintableRuneWidth(before) != 2 {
		t.Fatalf("before width = %d", ansi.PrintableRuneWidth(before))
	}
	if !strings.Contains(before, "\x1b[38;2;0;255;135m") {
		t.Fatalf("before lost color code: %q", before)
	}
	if !strings.Contains(after, "\x1b[0m") {
		t.Fatalf("after lost reset code: %q", after)
	}
	if ansi.PrintableRuneWidth(after) != 1 {
		t.Fatalf("after width = %d", ansi.PrintableRuneWidth(after))
	}
}

func TestLeadingSGR(t *testing.T) {
	got := leadingSGR("\x1b[1m\x1b[38;2;1;2;3m" + "text")
	if got != "\x1b[1m\x1b[38;2;1;2;3m" {
		t.Fatalf("leading = %q", got)
	}
	if leadingSGR("plain") != "" {
		t.Fatal("plain should have no leading codes")
	}
}

func TestOverlayWindow(t *testing.T) {
	bg := strings.Repeat(strings.Repeat(".", 10)+"\n", 5)
	bg = strings.TrimRight(bg, "\n")
	fg := "┌──┐\n│XX│\n└──┘"

	out := overlayWindow(bg, fg, 4, 1)
	lines := strings.Split(out, "\n")
	if len(lines) != 5 {
		t.Fatalf("height changed: %d lines", len(lines))
	}
	if !strings.HasPrefix(lines[1], "....┌──┐") {
		t.Fatalf("box top row wrong: %q", lines[1])
	}
	if !strings.Contains(lines[2], "│XX│") {
		t.Fatalf("box middle wrong: %q", lines[2])
	}
	if !strings.HasPrefix(lines[3], "....└──┘") {
		t.Fatalf("box bottom wrong: %q", lines[3])
	}
	// rows outside the box unchanged
	if lines[0] != ".........." {
		t.Fatalf("row 0 changed: %q", lines[0])
	}
}
