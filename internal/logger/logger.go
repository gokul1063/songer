package logger

import (
	"fmt"
	"log/slog"
	"os"
	"time"
)

var WorkflowLogger *slog.Logger
var ErrorLogger *slog.Logger

func InitLogger() error {
	// create timestamped workflow log
	ts := time.Now().Format("20060102_150405")
	workflowPath := fmt.Sprintf("logs/workflow/run_%s.log", ts)

	wfFile, err := os.OpenFile(workflowPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	errFile, err := os.OpenFile("logs/error/error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	WorkflowLogger = slog.New(slog.NewTextHandler(wfFile, nil))
	ErrorLogger = slog.New(slog.NewTextHandler(errFile, nil))

	go cleanupOldLogs("logs/workflow", 10)

	return nil
}
