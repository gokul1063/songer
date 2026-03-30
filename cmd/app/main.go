package main

import (
	"time"
	"fmt"
	"songer-v3/internal/executor"
	"songer-v3/internal/logger"
)

func main() {
	err := logger.InitLogger()
	if err != nil {
		fmt.Println("Logger init failed:", err)
		return
	}

	testExecutor()
}

func testExecutor() {
	cfg := executor.ExecConfig{
		Timeout: 5 * time.Second,
		Retries: 1,
	}

	res := executor.RunCommand("echo", []string{"Hello from executor"}, cfg)

	fmt.Println("STDOUT:", res.Stdout)
	fmt.Println("STDERR:", res.Stderr)
	fmt.Println("ERROR:", res.Err)
}
