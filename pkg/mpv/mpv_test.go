package mpv

import (
	"context"
	"path/filepath"
	"testing"
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
