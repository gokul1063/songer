package main

import (
	"fmt"
	"songer-v3/internal/logger"
	"songer-v3/internal/workflow"
)

func main() {
	err := logger.InitLogger()
	if err != nil {
		fmt.Println("Logger init failed:", err)
		return
	}

	testFunction()
}

func testFunction() {
	workflow.Enter("testFunction")

	// simulate work

	workflow.Exit("testFunction", "success")
}
