// Command fakeopencode is a minimal stand-in for `opencode serve`, used by
// supervisor lifecycle tests to exercise real process start/stop/readiness
// without depending on the real OpenCode binary.
//
// Behaviour is selected via FAKE_OPENCODE_* env vars:
//
//	FAKE_OPENCODE_MODE            healthy (default) | unhealthy (500) | never-ready | exit-immediately
//	FAKE_OPENCODE_REQUIRE_AUTH    1 -> /global/health requires Basic Auth
//	FAKE_OPENCODE_IGNORE_SIGTERM  1 -> ignore SIGTERM (forces kill fallback)
//	FAKE_OPENCODE_SPAWN_CHILD     1 -> spawn a child process in the same group
package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {
	if os.Getenv("FAKE_OPENCODE_CHILD") == "1" {
		// Child process that simply blocks; killed together with the group.
		select {}
	}

	hostname, port := parseArgs(os.Args[1:])

	switch os.Getenv("FAKE_OPENCODE_MODE") {
	case "exit-immediately":
		os.Exit(3)
	case "never-ready":
		select {}
	}

	if os.Getenv("FAKE_OPENCODE_SPAWN_CHILD") == "1" {
		spawnChild()
	}

	if os.Getenv("FAKE_OPENCODE_IGNORE_SIGTERM") == "1" {
		signal.Ignore(syscall.SIGTERM)
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(hostname, strconv.Itoa(port)))
	if err != nil {
		os.Exit(1)
	}

	mode := os.Getenv("FAKE_OPENCODE_MODE")
	mux := http.NewServeMux()
	mux.HandleFunc("/global/health", func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("FAKE_OPENCODE_REQUIRE_AUTH") == "1" {
			u, p, ok := r.BasicAuth()
			if !ok || u != os.Getenv("OPENCODE_SERVER_USERNAME") || p != os.Getenv("OPENCODE_SERVER_PASSWORD") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if mode == "unhealthy" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"not ready"}`))
			return
		}
		if mode == "healthy-then-exit" {
			// Deliver a complete (Content-Length) healthy response, then exit
			// immediately — reproducing "health succeeded then process died".
			// Content-Length avoids the truncated chunked-body problem when the
			// process dies before the handler returns.
			const body = `{"healthy":true,"version":"fake"}`
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			os.Exit(0)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"healthy": true, "version": "fake"})
	})

	_ = http.Serve(ln, mux)
}

func parseArgs(args []string) (string, int) {
	hostname := "127.0.0.1"
	port := 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--hostname":
			if i+1 < len(args) {
				hostname = args[i+1]
				i++
			}
		case "--port":
			if i+1 < len(args) {
				port, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
	}
	return hostname, port
}

func spawnChild() {
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "FAKE_OPENCODE_CHILD=1")
	if err := cmd.Start(); err != nil {
		os.Exit(1)
	}
	if pf := os.Getenv("FAKE_OPENCODE_CHILD_PID_FILE"); pf != "" {
		_ = os.WriteFile(pf, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)
	}
}
