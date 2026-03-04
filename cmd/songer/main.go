package main

import (
	"fmt"
	"os"

	"songer/internal/config"
	"songer/internal/logger"
)

func main() {
	path := os.ExpandEnv("$HOME/.config/songer/config.json")
	cfg, err := config.LoadConfig(path)

	if err != nil {
		fmt.Println("failed to load config:", err)
		os.Exit(1)
	}

	err = logger.InitLogger(cfg.LogFile)

	if err != nil {
		fmt.Println("failed to initialize logger:", err)
		os.Exit(1)
	}

	logger.Info("main", "application started")

	fmt.Println("Songer initialized")
}
