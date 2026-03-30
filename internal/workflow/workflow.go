package workflow

import "songer-v3/internal/logger"

func Enter(funcName string) {
	logger.WorkflowLogger.Info("ENTER", "function", funcName)
}

func Exit(funcName string, status string) {
	logger.WorkflowLogger.Info("EXIT", "function", funcName, "status", status)
}
