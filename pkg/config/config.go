package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	UI      UI                 `toml:"ui"`
	Player  PlayerCfg          `toml:"player"`
	Autoplay AutoplayCfg        `toml:"autoplay"`
	Themes  map[string]Theme   `toml:"themes"`
}

type UI struct {
	Theme string `toml:"theme"`
}

type PlayerCfg struct {
	Volume int `toml:"volume"`
}

type AutoplayCfg struct {
	PerNode int `toml:"per_node"`
	Depth   int `toml:"depth"`
}

type Theme struct {
	Background string `toml:"background"`
	Surface    string `toml:"surface"`
	Primary    string `toml:"primary"`
	Secondary  string `toml:"secondary"`
	Accent     string `toml:"accent"`
	Text       string `toml:"text"`
	Muted      string `toml:"muted"`
	Progress   string `toml:"progress"`
	Track      string `toml:"track"`
	Wave       string `toml:"wave"`
	WaveAlt    string `toml:"wave_alt"`
	Thumb      string `toml:"thumb"`
	Selection  string `toml:"selection"`
	Border     string `toml:"border"`
	Header     string `toml:"header"`
	HeaderBg   string `toml:"header_bg"`
	Footer     string `toml:"footer"`
	FooterBg   string `toml:"footer_bg"`
}

func Default() Config {
	return Config{
		UI:      UI{Theme: "opencode"},
		Player:  PlayerCfg{Volume: 80},
		Autoplay: AutoplayCfg{PerNode: 2, Depth: 2},
		Themes: map[string]Theme{
			"opencode": {
				Background: "#0f0f1a",
				Surface:    "#1a1a2e",
				Primary:    "#00ff87",
				Secondary:  "#00e5ff",
				Accent:     "#ffcc66",
				Text:       "#e8e8f0",
				Muted:      "#6b7280",
				Progress:   "#00ff87",
				Track:      "#33334d",
				Wave:       "#00e5ff",
				WaveAlt:    "#00ff87",
				Thumb:      "#ffcc66",
				Selection:  "#00ff87",
				Border:     "#2a2a40",
				Header:     "#00ff87",
				HeaderBg:   "#16162a",
				Footer:     "#9aa0b4",
				FooterBg:   "#16162a",
			},
		},
	}
}

func (c Config) ThemeFor(name string) Theme {
	if t, ok := c.Themes[name]; ok {
		return t
	}
	return c.Themes["opencode"]
}

// Load reads the first config file that exists, falling back to defaults.
// Order: local ./config.toml, then ~/.config/songer/config.toml.
func Load() (Config, error) {
	cfg := Default()
	for _, p := range paths() {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if _, err := toml.DecodeFile(p, &cfg); err != nil {
			return cfg, fmt.Errorf("parse %s: %w", p, err)
		}
		if cfg.Themes == nil {
			cfg.Themes = Default().Themes
		}
		if cfg.UI.Theme == "" {
			cfg.UI.Theme = "opencode"
		}
		return cfg, nil
	}
	return cfg, nil
}

func paths() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"config.toml",
		filepath.Join(home, ".config", "songer", "config.toml"),
	}
}
