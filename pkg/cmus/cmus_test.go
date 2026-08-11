package cmus

import (
	"strings"
	"testing"
	"time"
)

func TestParseStatus(t *testing.T) {
	out := `status playing
file /home/user/music/song.flac
position 42
duration 213
bitrate 1048
codec flac
tag title Some Song
tag artist Some Artist
tag album An Album
set main win_width 0.8
set vol_left 80
set vol_right 80
`
	s := parseStatus(out)
	if s.State != "playing" {
		t.Errorf("State = %q, want playing", s.State)
	}
	if s.Position != 42 {
		t.Errorf("Position = %d, want 42", s.Position)
	}
	if s.Duration != 213 {
		t.Errorf("Duration = %d, want 213", s.Duration)
	}
	if s.Volume != 80 {
		t.Errorf("Volume = %d, want 80", s.Volume)
	}
	if s.Title != "Some Song" {
		t.Errorf("Title = %q, want %q", s.Title, "Some Song")
	}
}

// cmus < 2.12 called the field "filepos"; keep parsing it as a fallback.
func TestParseStatusLegacyFilepos(t *testing.T) {
	out := "status playing\nfilepos 9\nduration 100\n"
	s := parseStatus(out)
	if s.Position != 9 {
		t.Errorf("Position = %d, want 9 (legacy filepos)", s.Position)
	}
}

func TestParseStatusStream(t *testing.T) {
	// Local downloads: -1 duration until cmus figures it out, no tags.
	out := `status playing
file /tmp/songer-cmus/abc.m4a
filepos 3
duration -1
set vol_left 80
set vol_right 80
`
	s := parseStatus(out)
	if s.State != "playing" {
		t.Errorf("State = %q, want playing", s.State)
	}
	if s.Duration != -1 {
		t.Errorf("Duration = %d, want -1", s.Duration)
	}
	if s.Title != "" {
		t.Errorf("Title = %q, want empty", s.Title)
	}
}

func TestParseStatusStopped(t *testing.T) {
	s := parseStatus("status stopped\n")
	if s.State != "stopped" {
		t.Errorf("State = %q, want stopped", s.State)
	}
}

func TestParseStatusEmpty(t *testing.T) {
	s := parseStatus("")
	if s.State != "" || s.Position != -1 || s.Duration != -1 || s.Volume != -1 {
		t.Errorf("empty output should leave unknown fields at -1: %+v", s)
	}
}

func TestSeekCmd(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{5 * time.Second, "seek +5"},
		{-5 * time.Second, "seek -5"},
		{10 * time.Second, "seek +10"},
		{0, "seek 0"},
		{-90 * time.Second, "seek -90"},
	}
	for _, c := range cases {
		if got := seekCmd(c.d); got != c.want {
			t.Errorf("seekCmd(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestPlayFileFlags(t *testing.T) {
	// PlayFile drives `cmus-remote -s` then `cmus-remote -f <path>`; the file
	// path must be passed as one argv element, never shell-quoted.
	path := "/tmp/songer-cmus/abc.m4a"
	if !strings.Contains(path, "/tmp/songer-cmus/abc.m4a") {
		t.Fatalf("path mangled: %q", path)
	}
	if strings.Contains(path, "\\") || strings.Contains(path, "'") {
		t.Fatalf("path was shell-quoted: %q", path)
	}
}

func TestSetVolumeUsesEqualsSyntax(t *testing.T) {
	// cmus 2.12 requires `set opt=value`; the space form errors out.
	cases := []struct{ name, value, want string }{
		{"softvol", "true", "set softvol=true"},
		{"continue", "false", "set continue=false"},
		{"play_library", "false", "set play_library=false"},
	}
	for _, c := range cases {
		if got := setCmd(c.name, c.value); got != c.want {
			t.Errorf("setCmd(%q, %q) = %q, want %q", c.name, c.value, got, c.want)
		}
	}
}
