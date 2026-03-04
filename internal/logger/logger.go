package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var logger *log.Logger

func InitLogger(path string) error {

	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	logger = log.New(file, "", 0)
	return nil
}

func logMessage(level string, function string, message string) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	entry := fmt.Sprintf("%s %s %s %s", timestamp, level, function, message)

	if logger != nil {
		logger.Println(entry)
	} else {
		fmt.Println(entry)
	}
}

func Info(function string, message string) {
	logMessage("INFO", function, message)
}

func Error(function string, message string) {
	logMessage("ERROR", function, message)
}

func Debug(function string, message string) {
	logMessage("DEBUG", function, message)
}
