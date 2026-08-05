package download

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"songer/pkg/search"
)

func writeFakeBin(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "yt-dlp")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

const fakeOK = `echo "[download] 10.0% of 1.2MiB at 500KiB/s ETA 00:01"
echo "[download] 100% of 1.2MiB"
echo "FILE=/tmp/songer-test.mp3"
exit 0
`

const fakeFail = `echo "[download] downloading something"
exit 1
`

func TestProgressParser(t *testing.T) {
	cases := []struct {
		line string
		want float64
		ok   bool
	}{
		{"[download] 45.2% of 3.2MiB at 1.1MiB/s ETA 00:02", 45.2, true},
		{"[download] 100% of 3.2MiB", 100, true},
		{"[info] foo", 0, false},
		{"something else", 0, false},
	}
	for _, c := range cases {
		m := progressRe.FindStringSubmatch(c.line)
		if c.ok && len(m) != 2 {
			t.Errorf("expected match for %q", c.line)
			continue
		}
		if !c.ok {
			if len(m) != 0 {
				t.Errorf("unexpected match for %q", c.line)
			}
			continue
		}
	}
}

func TestDownload(t *testing.T) {
	bin := writeFakeBin(t, fakeOK)

	var pcts []float64
	video := search.Video{ID: "abc", Title: "T", URL: "https://youtu.be/abc"}
	path, err := Download(context.Background(), video, Options{
		Dir: t.TempDir(),
		Bin: bin,
		OnProgress: func(v search.Video, pct float64) {
			pcts = append(pcts, pct)
		},
	})
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if path != "/tmp/songer-test.mp3" {
		t.Fatalf("unexpected path: %q", path)
	}
	if len(pcts) != 2 || pcts[0] != 10 || pcts[1] != 100 {
		t.Fatalf("unexpected progress: %v", pcts)
	}
}

func TestDownloadError(t *testing.T) {
	bin := writeFakeBin(t, fakeFail)
	_, err := Download(context.Background(), search.Video{ID: "x"}, Options{Bin: bin, Dir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error from failing yt-dlp")
	}
}

func TestDownloadMissingBin(t *testing.T) {
	_, err := Download(context.Background(), search.Video{ID: "x"}, Options{Bin: "/nonexistent/yt-dlp"})
	if err == nil {
		t.Fatal("expected error when binary missing")
	}
}

func TestJSRuntimeDetection(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "node")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	got := jsRuntimeFlag()
	if len(got) != 2 || got[1] != "node" {
		t.Fatalf("expected node runtime flag, got %v", got)
	}
}

func TestDownloadMany(t *testing.T) {
	bin := writeFakeBin(t, fakeOK)

	videos := []search.Video{
		{ID: "a", Title: "A"},
		{ID: "b", Title: "B"},
		{ID: "c", Title: "C"},
		{ID: "d", Title: "D"},
	}

	results := DownloadMany(context.Background(), videos, 4, Options{Bin: bin, Dir: t.TempDir()})
	got := map[string]bool{}
	for res := range results {
		if res.Err != nil {
			t.Fatalf("unexpected error: %v", res.Err)
		}
		if res.Path == "" {
			t.Fatalf("empty path for %s", res.Video.ID)
		}
		got[res.Video.ID] = true
	}
	for _, v := range videos {
		if !got[v.ID] {
			t.Fatalf("missing result for %s", v.ID)
		}
	}
}
