package logger

import (
	"log/slog"
	"runtime"
)

func LogError(err error) {
	_, file, line, _ := runtime.Caller(1)

	ErrorLogger.Error("ERROR",
		"file", file,
		"line", line,
		"error", err.Error(),
	)
}
