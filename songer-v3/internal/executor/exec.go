package executor

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

type ExecResult struct {
	Stdout string
	Stderr string
	Err    error
}

type ExecConfig struct {
	Timeout time.Duration
	Retries int
}

func RunCommand(cmdName string, args []string, cfg ExecConfig) ExecResult {
	var result ExecResult

	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	if cfg.Retries <= 0 {
		cfg.Retries = 1
	}

	for i := 0; i < cfg.Retries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, cmdName, args...)

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		result = ExecResult{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
			Err:    err,
		}

		// success
		if err == nil {
			return result
		}

		// if timeout
		if ctx.Err() == context.DeadlineExceeded {
			result.Err = errors.New("command timed out")
		}
	}

	return result
}
