package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	MusicDirectory  string   `json:"music_directory"`
	ScanDirectories []string `json:"scan_directories"`
	DownloadQuality string   `json:"download_quality"`
	LogFile         string   `json:"log_file"`
}

var AppConfig Config

func LoadConfig(path string) (Config, error) {

	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}

	defer file.Close()

	decoder := json.NewDecoder(file)

	var cfg Config
	err = decoder.Decode(&cfg)

	if err != nil {
		return Config{}, err
	}

	AppConfig = cfg

	return cfg, nil
}

func GetConfig() Config {
	return AppConfig
}
