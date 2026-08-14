// Command fakeopencode is a minimal stand-in for `opencode serve`, used by
// supervisor lifecycle tests to exercise real process start/stop/readiness
// without depending on the real OpenCode binary.
//
// Behaviour is selected via FAKE_OPENCODE_* env vars:
//
//	FAKE_OPENCODE_MODE            healthy (default) | unhealthy (500) | never-ready | exit-immediately
//	FAKE_OPENCODE_REQUIRE_AUTH    1 -> all endpoints require Basic Auth
//	FAKE_OPENCODE_IGNORE_SIGTERM  1 -> ignore SIGTERM (forces kill fallback)
//	FAKE_OPENCODE_SPAWN_CHILD     1 -> spawn a child process in the same group
//
// Beyond readiness, it serves just enough of the OpenCode HTTP API for an
// end-to-end TUI smoke: POST /session creates a session, POST
// /session/{id}/prompt_async triggers a scripted SSE event sequence
// (reasoning + answer + tool + step.finished), and GET /global/event streams
// that sequence to subscribers. This keeps the smoke deterministic and offline.
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
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
	if pf := os.Getenv("FAKE_OPENCODE_PID_FILE"); pf != "" {
		_ = os.WriteFile(pf, []byte(strconv.Itoa(os.Getpid())), 0o644)
	}

	hub := newEventHub()
	mode := os.Getenv("FAKE_OPENCODE_MODE")
	mux := http.NewServeMux()

	// promptCount distinguishes the first prompt (review flow) from the second
	// (approval flow) so the smoke can drive both scripted sequences.
	var promptMu sync.Mutex
	promptCount := 0

	mux.HandleFunc("/global/health", func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if mode == "unhealthy" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"not ready"}`))
			return
		}
		if mode == "healthy-then-exit" {
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

	mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "sess-1"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]any{
			map[string]any{
				"id":    "sess-1",
				"title": "Alpha Task",
				"time":  map[string]any{"created": 1000, "updated": 2000},
			},
			map[string]any{
				"id":    "sess-2",
				"title": "Beta Task",
				"time":  map[string]any{"created": 1000, "updated": 1500},
			},
		})
	})

	mux.HandleFunc("/session/", func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/prompt_async") && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusNoContent)
			promptMu.Lock()
			n := promptCount
			promptCount++
			promptMu.Unlock()
			if n == 0 {
				go hub.emitScript()
			} else {
				go hub.emitApprovalScript()
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	mux.HandleFunc("/permission/", func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/reply") && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(true)
			go hub.emitApprovalContinuation()
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	mux.HandleFunc("/global/event", func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}

		ch := hub.subscribe()
		defer hub.unsubscribe(ch)

		// Announce connectivity so the TUI sees the runtime is alive.
		_, _ = fmt.Fprintf(w, "data: %s\n\n", sseEnvelope("server.connected", map[string]any{}))
		flusher.Flush()

		for {
			select {
			case <-r.Context().Done():
				return
			case payload, ok := <-ch:
				if !ok {
					return
				}
				_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
				flusher.Flush()
			}
		}
	})

	_ = http.Serve(ln, mux)
}

// authorized reports whether the request satisfies the (optional) Basic Auth
// gate. When REQUIRE_AUTH is unset every request is allowed.
func authorized(r *http.Request) bool {
	if os.Getenv("FAKE_OPENCODE_REQUIRE_AUTH") != "1" {
		return true
	}
	u, p, ok := r.BasicAuth()
	return ok && u == os.Getenv("OPENCODE_SERVER_USERNAME") && p == os.Getenv("OPENCODE_SERVER_PASSWORD")
}

// eventHub fans out scripted SSE payloads to all active /global/event
// subscribers. Each subscriber gets its own buffered channel.
type eventHub struct {
	mu   sync.Mutex
	subs map[chan string]struct{}
}

func newEventHub() *eventHub {
	return &eventHub{subs: make(map[chan string]struct{})}
}

func (h *eventHub) subscribe() chan string {
	ch := make(chan string, 64)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *eventHub) unsubscribe(ch chan string) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
}

func (h *eventHub) broadcast(payload string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- payload:
		default:
		}
	}
}

// emitScript emits the deterministic prompt lifecycle: step-start, reasoning
// deltas, answer deltas, a tool call, then step-finished. Small inter-event
// delays keep the stream observable (and exercise the TUI coalescing path).
func (h *eventHub) emitScript() {
	step := func(typ string, props map[string]any) {
		h.broadcast(sseEnvelope(typ, props))
		time.Sleep(10 * time.Millisecond)
	}

	step("message.part.updated", map[string]any{
		"part": map[string]any{"id": "part-step", "type": "step-start"},
	})
	step("message.part.delta", map[string]any{
		"sessionID": "sess-1", "messageID": "msg-1", "partID": "part-reason",
		"field": "reasoning", "delta": "Let me analyze the request.",
	})
	step("message.part.delta", map[string]any{
		"sessionID": "sess-1", "messageID": "msg-1", "partID": "part-reason",
		"field": "reasoning", "delta": " I should review the target file first.",
	})
	// Leave the reasoning block open long enough to register a non-zero
	// duration before the answer begins.
	time.Sleep(150 * time.Millisecond)
	step("message.part.delta", map[string]any{
		"sessionID": "sess-1", "messageID": "msg-1", "partID": "part-answer",
		"field": "text", "delta": "Here is the review:",
	})
	step("message.part.delta", map[string]any{
		"sessionID": "sess-1", "messageID": "msg-1", "partID": "part-answer",
		"field": "text", "delta": " the code looks good.",
	})
	step("message.part.updated", map[string]any{
		"part": map[string]any{"id": "part-tool", "type": "tool", "tool": "read", "callID": "c1"},
	})
	step("message.part.updated", map[string]any{
		"part": map[string]any{"id": "part-finish", "type": "step-finish"},
	})
}

// emitApprovalScript emits a permission.asked flow: the agent starts a step and
// immediately requests approval for a dangerous shell command. It blocks there;
// the continuation is emitted by emitApprovalContinuation once the TUI replies.
func (h *eventHub) emitApprovalScript() {
	step := func(typ string, props map[string]any) {
		h.broadcast(sseEnvelope(typ, props))
		time.Sleep(10 * time.Millisecond)
	}
	step("message.part.updated", map[string]any{
		"part": map[string]any{"id": "part-step", "type": "step-start"},
	})
	step("permission.asked", map[string]any{
		"id":         "per_1",
		"permission": "bash",
		"sessionID":  "sess-1",
		"patterns":   []string{"rm", "-rf", "./build"},
		"metadata":   map[string]any{"command": "rm -rf ./build"},
	})
}

// emitApprovalContinuation is broadcast after the TUI replies to the approval,
// simulating the agent resuming and completing the step.
func (h *eventHub) emitApprovalContinuation() {
	step := func(typ string, props map[string]any) {
		h.broadcast(sseEnvelope(typ, props))
		time.Sleep(10 * time.Millisecond)
	}
	step("message.part.delta", map[string]any{
		"sessionID": "sess-1", "messageID": "msg-2", "partID": "part-answer",
		"field": "text", "delta": "Deleted build directory.",
	})
	step("message.part.updated", map[string]any{
		"part": map[string]any{"id": "part-finish", "type": "step-finish"},
	})
}

// sseEnvelope wraps a vendor event type + properties in the OpenCode SSE
// envelope shape that MapEvent expects.
func sseEnvelope(typ string, props map[string]any) string {
	env := map[string]any{
		"directory": "",
		"payload": map[string]any{
			"type":       typ,
			"properties": props,
		},
	}
	data, _ := json.Marshal(env)
	return string(data)
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
