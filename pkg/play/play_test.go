package play

import (
	"testing"
	"time"

	"songer/pkg/search"
)

func TestDetectStart(t *testing.T) {
	cases := []struct {
		line string
		want time.Duration
		ok   bool
	}{
		{"SONGER_TIME 00:00:00", 0, false},
		{"SONGER_TIME 00:00:01", time.Second, true},
		{"SONGER_TIME 00:03:34", 214 * time.Second, true},
		{"SONGER_TIME 01:02:03", 3723 * time.Second, true},
		{"[status] SONGER_TIME 00:00:02\r", 2 * time.Second, true},
		{"no marker here", 0, false},
		{"SONGER_TIME garbage", 0, false},
	}
	for _, c := range cases {
		got, ok := detectStart(c.line)
		if ok != c.ok || got != c.want {
			t.Errorf("detectStart(%q) = (%v, %v), want (%v, %v)", c.line, got, ok, c.want, c.ok)
		}
	}
}

func TestBuildArgs(t *testing.T) {
	v := search.Video{ID: "abc", URL: "https://youtu.be/abc"}

	args := buildArgs(v, Options{})
	if args[0] != "--no-video" || args[len(args)-1] != v.URL {
		t.Fatalf("unexpected audio args: %v", args)
	}
	seenStatus := false
	for _, a := range args {
		if a == "--term-status-msg="+statusPrefix+"${playback-time}" {
			seenStatus = true
		}
	}
	if !seenStatus {
		t.Fatalf("args missing status msg: %v", args)
	}

	vargs := buildArgs(v, Options{Video: true})
	for _, a := range vargs {
		if a == "--no-video" {
			t.Fatalf("video mode should not pass --no-video: %v", vargs)
		}
	}
}
