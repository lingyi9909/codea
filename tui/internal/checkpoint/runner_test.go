package checkpoint

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunnerHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_CHECKPOINT_HELPER") != "1" {
		return
	}
	args := os.Args
	sep := -1
	for i, arg := range args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if sep < 0 {
		os.Exit(2)
	}
	helperArgs := args[sep+1:]
	if len(helperArgs) > 0 && helperArgs[0] == "sleep" {
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}
	in, _ := io.ReadAll(os.Stdin)
	fmt.Printf("ARGS=%s\n", strings.Join(helperArgs, "|"))
	fmt.Printf("STDIN=%x\n", in)
	if len(helperArgs) > 0 && helperArgs[0] == "fail" {
		fmt.Fprint(os.Stderr, strings.Repeat("x", 2048))
		os.Exit(7)
	}
	os.Exit(0)
}

func helperRunner() *ExecRunner {
	r := NewExecRunner(os.Args[0])
	r.PrefixArgs = []string{"-test.run=TestRunnerHelperProcess", "--"}
	r.Env = append(os.Environ(), "GO_WANT_CHECKPOINT_HELPER=1")
	r.StderrLimit = 128
	return r
}

func TestRunnerPassesArgvAndNULStdinDirectly(t *testing.T) {
	r := helperRunner()
	payload := []byte("a.go\x00目录/b.go\x00")
	got, err := r.Run(context.Background(), []string{"stdin", "--pathspec-from-file=-", "--pathspec-file-nul"}, payload)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got.Stdout, []byte("ARGS=stdin|--pathspec-from-file=-|--pathspec-file-nul")) {
		t.Fatalf("argv not preserved: %s", got.Stdout)
	}
	if !bytes.Contains(got.Stdout, []byte(fmt.Sprintf("STDIN=%x", payload))) {
		t.Fatalf("stdin not preserved: %s", got.Stdout)
	}
}

func TestRunnerCancellationStopsProcess(t *testing.T) {
	r := helperRunner()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := r.Run(ctx, []string{"sleep"}, nil)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("expected deadline error, got %v", err)
	}
}

func TestRunnerBoundsStderrOnFailure(t *testing.T) {
	r := helperRunner()
	result, err := r.Run(context.Background(), []string{"fail"}, nil)
	if err == nil {
		t.Fatal("expected command failure")
	}
	if len(result.Stderr) > 128 {
		t.Fatalf("stderr not bounded: %d", len(result.Stderr))
	}
	if len(err.Error()) > 512 {
		t.Fatalf("error should stay bounded: %d", len(err.Error()))
	}
}
