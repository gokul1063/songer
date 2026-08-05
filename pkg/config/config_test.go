package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLocalConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	toml := `[ui]
theme = "mono"

[player]
volume = 55

[autoplay]
per_node = 3
depth = 1

[themes.mono]
background = "#000000"
primary    = "#ffffff"
`
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(cwd)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UI.Theme != "mono" {
		t.Errorf("theme = %q, want mono", cfg.UI.Theme)
	}
	if cfg.Player.Volume != 55 {
		t.Errorf("volume = %d, want 55", cfg.Player.Volume)
	}
	if cfg.Autoplay.PerNode != 3 || cfg.Autoplay.Depth != 1 {
		t.Errorf("autoplay = %+v", cfg.Autoplay)
	}
	th := cfg.ThemeFor("mono")
	if th.Background != "#000000" || th.Primary != "#ffffff" {
		t.Errorf("theme decode wrong: %+v", th)
	}
}

func TestDefaults(t *testing.T) {
	cfg := Default()
	th := cfg.ThemeFor("opencode")
	if th.Primary != "#00ff87" {
		t.Errorf("default opencode primary wrong: %q", th.Primary)
	}
	fallback := cfg.ThemeFor("does-not-exist")
	if fallback != th {
		t.Errorf("fallback should return opencode theme")
	}
}
