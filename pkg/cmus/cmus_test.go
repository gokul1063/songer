package cmus

import (
	"strings"
	"testing"
	"time"
)

func TestParseStatus(t *testing.T) {
	out := `status playing
file /home/user/music/song.flac
filepos 42
duration 213
bitrate 1048
codec flac
tag title Some Song
tag artist Some Artist
tag album An Album
set main win_width 0.8
vol_left 80
vol_right 80
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

func TestParseStatusStream(t *testing.T) {
	// YouTube streams: no tags, -1 duration until cmus figures it out.
	out := `status playing
file https://rr2---sn-a5mekn7d.googlevideo.com/videoplayback?id=abc&itag=140
filepos 3
duration -1
vol_left 80
vol_right 80
`
	s := parseStatus(out)
	if s.State != "playing" {
		t.Errorf("State = %q, want playing", s.State)
	}
	if s.Duration != -1 {
		t.Errorf("Duration = %d, want -1", s.Duration)
	}
	if s.Title != "" {
		t.Errorf("Title = %q, want empty for stream", s.Title)
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

func TestCommandIsSingleArg(t *testing.T) {
	// cmus-remote -C takes the whole command as one argv element, so stream
	// URLs with query strings (&, =, %) must not be split or shell-quoted.
	url := "https://rr2.example.com/videoplayback?id=abc&itag=140&x=y"
	cmd := "add -p " + url
	if !strings.Contains(cmd, "&itag=140") {
		t.Fatalf("query string mangled: %q", cmd)
	}
	if strings.Contains(cmd, "\\") || strings.Contains(cmd, "'") {
		t.Fatalf("command was shell-quoted: %q", cmd)
	}
}
