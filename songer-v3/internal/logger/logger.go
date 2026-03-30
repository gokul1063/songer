package logger

import (
	"log/slog"
	"os"
)

var WorkflowLogger *slog.Logger
var ErrorLogger *slog.Logger

func InitLogger() error {
	// workflow log file
	wfFile, err := os.OpenFile("logs/workflow/workflow.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	// error log file
	errFile, err := os.OpenFile("logs/error/error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	WorkflowLogger = slog.New(slog.NewTextHandler(wfFile, nil))
	ErrorLogger = slog.New(slog.NewTextHandler(errFile, nil))

	return nil
}
