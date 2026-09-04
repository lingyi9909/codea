package checkpoint

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

const defaultStderrLimit = 16 * 1024

type ExecRunner struct {
	Binary      string
	Dir         string
	PrefixArgs  []string
	Env         []string
	StderrLimit int
}

func NewExecRunner(binary string) *ExecRunner {
	return &ExecRunner{Binary: binary, StderrLimit: defaultStderrLimit}
}

func NewGitRunner() *ExecRunner { return NewExecRunner("git") }

func (r *ExecRunner) Run(ctx context.Context, args []string, stdin []byte) (Result, error) {
	binary := r.Binary
	if binary == "" {
		binary = "git"
	}
	argv := make([]string, 0, len(r.PrefixArgs)+len(args))
	argv = append(argv, r.PrefixArgs...)
	argv = append(argv, args...)
	cmd := exec.CommandContext(ctx, binary, argv...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	if r.Env != nil {
		cmd.Env = append([]string(nil), r.Env...)
	}
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	limit := r.StderrLimit
	if limit <= 0 {
		limit = defaultStderrLimit
	}
	bounded := boundTail(stderr.Bytes(), limit)
	result := Result{Stdout: stdout.Bytes(), Stderr: bounded, ExitCode: 0}
	if err == nil {
		return result, nil
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return result, &Error{Code: CodeCheckpointUnavailable, Message: "git executable is unavailable", Cause: err}
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.ExitCode = -1
	}
	msg := fmt.Sprintf("git command failed (exit=%d)", result.ExitCode)
	if len(bounded) > 0 {
		msg += ": " + string(bounded)
	}
	return result, fmt.Errorf("%s: %w", msg, err)
}

func boundTail(in []byte, limit int) []byte {
	if len(in) <= limit {
		return append([]byte(nil), in...)
	}
	return append([]byte(nil), in[len(in)-limit:]...)
}
