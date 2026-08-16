//go:build darwin

// Package tuismoke drives the real Codea TUI binary under a PTY and asserts the
// full end-to-end flow: supervised runtime startup, prompt submission,
// reasoning + answer streaming, tool activity, resize, ctrl+t, and ctrl+c
// shutdown with runtime teardown.
//
// It is opt-in (CODEA_TUI_SMOKE=1) because it launches a real terminal program
// and is timing-sensitive; the deterministic unit/contract tests run in the
// normal go test ./... gate.
package tuismoke

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// ansiRe strips CSI/OSC escape sequences so the transcript is human-readable.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]|\x1b\][^\x07\x1b]*(\x07|\x1b\\)|\x1b[()][0-9A-B]`)

func buildBinary(t *testing.T, pkg, out string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", out, pkg)
	cmd.Env = os.Environ()
	if o, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", pkg, err, o)
	}
}

// ptyPair is a master/slave pseudo-terminal pair.
type ptyPair struct {
	master *os.File
	slave  *os.File
}

// winsize mirrors the C struct winsize layout (row, col, xpixel, ypixel).
type winsize struct {
	row, col, xpixel, ypixel uint16
}

func openPty() (*ptyPair, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/ptmx: %w", err)
	}
	// grantpt + unlockpt make the slave accessible to this process.
	for _, req := range []uintptr{uintptr(syscall.TIOCPTYGRANT), uintptr(syscall.TIOCPTYUNLK)} {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), req, 0); errno != 0 {
			master.Close()
			return nil, fmt.Errorf("grantpt/unlockpt ioctl 0x%x: %v", req, errno)
		}
	}
	name := make([]byte, 128)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), uintptr(syscall.TIOCPTYGNAME), uintptr(unsafe.Pointer(&name[0])))
	if errno != 0 {
		master.Close()
		return nil, fmt.Errorf("TIOCPTYGNAME: %v", errno)
	}
	idx := bytes.IndexByte(name, 0)
	if idx < 0 {
		master.Close()
		return nil, fmt.Errorf("TIOCPTYGNAME returned no NUL-terminated name")
	}
	slave, err := os.OpenFile(string(name[:idx]), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return nil, fmt.Errorf("open slave %s: %w", string(name[:idx]), err)
	}
	return &ptyPair{master: master, slave: slave}, nil
}

func (p *ptyPair) setSize(rows, cols uint16) error {
	ws := winsize{row: rows, col: cols}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, p.master.Fd(), uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(&ws)))
	if errno != 0 {
		return errno
	}
	return nil
}

// respondToQueries answers the terminal queries Bubble Tea/termenv emits at
// startup (background/foreground color OSC reports and the cursor-position DSR)
// so the program can proceed without a real terminal emulator on the other end.
func respondToQueries(data []byte, responded map[string]bool, master *os.File) {
	queries := []struct{ q, r string }{
		{"\x1b]11;?\x1b\\", "\x1b]11;rgb:1a1b/1a1b/1a1b\x07"},
		{"\x1b]10;?\x1b\\", "\x1b]10;rgb:ffff/ffff/ffff\x07"},
		{"\x1b[6n", "\x1b[1;1R"},
	}
	for _, q := range queries {
		if responded[q.q] {
			continue
		}
		if bytes.Contains(data, []byte(q.q)) {
			_, _ = master.Write([]byte(q.r))
			responded[q.q] = true
		}
	}
}

func writeKeys(t *testing.T, master *os.File, s string) {
	t.Helper()
	if _, err := master.Write([]byte(s)); err != nil {
		t.Fatalf("write keys: %v", err)
	}
}

func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestRealTUISmoke(t *testing.T) {
	if os.Getenv("CODEA_TUI_SMOKE") != "1" {
		t.Skip("set CODEA_TUI_SMOKE=1 to run the real TUI smoke")
	}

	dir := t.TempDir()
	codeaBin := filepath.Join(dir, "codea")
	fakeBin := filepath.Join(dir, "fakeopencode")
	buildBinary(t, "codea/tui/cmd/codea", codeaBin)
	buildBinary(t, "codea/tui/internal/supervisor/fakeopencode", fakeBin)

	pty, err := openPty()
	if err != nil {
		t.Fatal(err)
	}
	defer pty.master.Close()

	pidFile := filepath.Join(dir, "fake.pid")
	if err := pty.setSize(40, 120); err != nil {
		t.Fatalf("set initial size: %v", err)
	}

	cmd := exec.Command(codeaBin)
	cmd.Env = append(os.Environ(),
		"OPENCODE_BIN="+fakeBin,
		"CODEA_RUNTIME_CONFIG_DIR="+filepath.Join(dir, "config"),
		"FAKE_OPENCODE_PID_FILE="+pidFile,
		"TERM=xterm-256color",
	)
	cmd.Stdin = pty.slave
	cmd.Stdout = pty.slave
	cmd.Stderr = pty.slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pty.slave.Close() // the child owns the slave fd now

	var mu sync.Mutex
	var out bytes.Buffer
	responded := map[string]bool{}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := pty.master.Read(buf)
			if n > 0 {
				mu.Lock()
				out.Write(buf[:n])
				data := out.Bytes()
				mu.Unlock()
				respondToQueries(data, responded, pty.master)
			}
			if err != nil {
				return
			}
		}
	}()

	contains := func(s string) bool {
		mu.Lock()
		defer mu.Unlock()
		return bytes.Contains(out.Bytes(), []byte(s))
	}

	// 1. TUI appears with header + status.
	waitFor(t, 15*time.Second, "TUI header", func() bool { return contains("Codea") })
	waitFor(t, 15*time.Second, "runtime Ready", func() bool { return contains("Ready") })

	// 2. Runtime auto-started and is healthy (supervisor launched fakeopencode).
	waitFor(t, 5*time.Second, "fakeopencode pid file", func() bool {
		_, err := os.Stat(pidFile)
		return err == nil
	})

	// 3. Submit a prompt; expect reasoning + answer + tool activity + finish.
	writeKeys(t, pty.master, "review the code")
	writeKeys(t, pty.master, "\r")

	waitFor(t, 20*time.Second, "answer streaming", func() bool {
		return contains("Here is the review:")
	})
	waitFor(t, 20*time.Second, "complete answer", func() bool {
		return contains("the code looks good.")
	})
	waitFor(t, 20*time.Second, "tool activity", func() bool {
		return contains("read")
	})
	waitFor(t, 20*time.Second, "thinking summary", func() bool {
		return contains("Spent") && contains("thinking")
	})

	// 3b. Approval flow: a second prompt triggers permission.asked, the approval
	// modal renders tool + command + danger warning, and Y replies so the agent
	// continues and completes the step.
	writeKeys(t, pty.master, "delete the build")
	writeKeys(t, pty.master, "\r")

	waitFor(t, 20*time.Second, "approval modal", func() bool {
		return contains("Tool approval required")
	})
	waitFor(t, 20*time.Second, "approval tool + command", func() bool {
		return contains("bash") && contains("rm -rf ./build")
	})
	waitFor(t, 20*time.Second, "danger warning", func() bool {
		return contains("Potentially dangerous command")
	})

	writeKeys(t, pty.master, "y")

	waitFor(t, 20*time.Second, "approval continuation", func() bool {
		return contains("Deleted build directory.")
	})

	// 3c. Session panel: ctrl+s opens the list, arrows move the cursor, Enter
	// resumes another session, and Esc closes the panel.
	writeKeys(t, pty.master, "\x13") // ctrl+s
	waitFor(t, 15*time.Second, "session panel", func() bool {
		return contains("Sessions")
	})
	waitFor(t, 15*time.Second, "session titles", func() bool {
		return contains("Alpha Task") && contains("Beta Task")
	})

	writeKeys(t, pty.master, "\x1b[B") // down
	time.Sleep(100 * time.Millisecond)
	writeKeys(t, pty.master, "\x1b[A") // up
	time.Sleep(100 * time.Millisecond)
	writeKeys(t, pty.master, "\x1b[B") // down
	time.Sleep(100 * time.Millisecond)
	writeKeys(t, pty.master, "\r") // resume (Beta Task -> sess-2)
	waitFor(t, 15*time.Second, "rehydrated history", func() bool {
		return contains("Earlier answer")
	})

	writeKeys(t, pty.master, "\x13") // ctrl+s reopen
	time.Sleep(200 * time.Millisecond)
	writeKeys(t, pty.master, "\x1b") // esc close
	time.Sleep(200 * time.Millisecond)

	// 4. Resize the terminal.
	if err := pty.setSize(50, 130); err != nil {
		t.Fatalf("resize: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// 5. Toggle thinking (ctrl+t).
	writeKeys(t, pty.master, "\x14")
	time.Sleep(300 * time.Millisecond)

	// 6. Quit (ctrl+c).
	writeKeys(t, pty.master, "\x03")

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("codea exited with error: %v", err)
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("codea did not exit within 10s after ctrl+c")
	}

	// 7. Runtime was torn down (supervisor.Stop killed the fakeopencode process).
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read fakeopencode pid: %v", err)
	}
	pid, err := strconv.Atoi(string(bytes.TrimSpace(pidBytes)))
	if err != nil {
		t.Fatalf("parse fakeopencode pid: %v", err)
	}
	waitFor(t, 5*time.Second, "runtime stopped", func() bool {
		return syscall.Kill(pid, 0) == syscall.ESRCH
	})

	// Evidence: write a readable transcript next to the report.
	mu.Lock()
	raw := out.String()
	mu.Unlock()
	if ev := os.Getenv("CODEA_TUI_SMOKE_TRANSCRIPT"); ev != "" {
		if err := os.WriteFile(ev, []byte(raw), 0o644); err != nil {
			t.Fatalf("write raw transcript: %v", err)
		}
		if err := os.WriteFile(ev+".txt", []byte(stripANSI(raw)), 0o644); err != nil {
			t.Fatalf("write stripped transcript: %v", err)
		}
	}
}

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}
