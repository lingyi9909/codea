package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type eventEnvelope struct {
	Directory string `json:"directory"`
	Payload   struct {
		Type       string          `json:"type"`
		Properties json.RawMessage `json:"properties"`
	} `json:"payload"`
}

type permissionRequest struct {
	ID         string         `json:"id"`
	SessionID  string         `json:"sessionID"`
	Permission string         `json:"permission"`
	Patterns   []string       `json:"patterns"`
	Metadata   map[string]any `json:"metadata"`
	Always     []string       `json:"always"`
}

type sessionStatus struct {
	SessionID string `json:"sessionID"`
	Status    struct {
		Type string `json:"type"`
	} `json:"status"`
}

type sessionError struct {
	SessionID string          `json:"sessionID"`
	Error     json.RawMessage `json:"error"`
}

func scanEvents(r io.Reader, visit func(eventEnvelope) (bool, error)) ([]eventEnvelope, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	events := make([]eventEnvelope, 0, 16)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		var event eventEnvelope
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return nil, fmt.Errorf("decode SSE event: %w", err)
		}
		events = append(events, event)
		done, err := visit(event)
		if err != nil {
			return events, err
		}
		if done {
			return events, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return events, fmt.Errorf("read SSE stream: %w", err)
	}
	return events, errors.New("SSE stream ended before expected event")
}

func readUntilPermission(r io.Reader, targetSessionID string) (permissionRequest, []eventEnvelope, error) {
	var request permissionRequest
	events, err := scanEvents(r, func(event eventEnvelope) (bool, error) {
		if event.Payload.Type != "permission.asked" {
			return false, nil
		}
		var candidate permissionRequest
		if err := json.Unmarshal(event.Payload.Properties, &candidate); err != nil {
			return false, fmt.Errorf("decode permission.asked: %w", err)
		}
		if candidate.SessionID != targetSessionID {
			return false, nil
		}
		request = candidate
		return true, nil
	})
	return request, events, err
}

func readUntilIdle(r io.Reader, targetSessionID string) ([]eventEnvelope, error) {
	return scanEvents(r, func(event eventEnvelope) (bool, error) {
		switch event.Payload.Type {
		case "session.error":
			var properties sessionError
			if err := json.Unmarshal(event.Payload.Properties, &properties); err != nil {
				return false, fmt.Errorf("decode session.error: %w", err)
			}
			if properties.SessionID == targetSessionID {
				return false, fmt.Errorf("session.error: %s", properties.Error)
			}
		case "session.status":
			var properties sessionStatus
			if err := json.Unmarshal(event.Payload.Properties, &properties); err != nil {
				return false, fmt.Errorf("decode session.status: %w", err)
			}
			return properties.SessionID == targetSessionID && properties.Status.Type == "idle", nil
		}
		return false, nil
	})
}

func postJSON(ctx context.Context, client *http.Client, endpoint string, body any, out any) (int, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return resp.StatusCode, fmt.Errorf("POST %s returned %s: %s", endpoint, resp.Status, strings.TrimSpace(string(data)))
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

func run(ctx context.Context, client *http.Client, baseURL, directory, reply string) error {
	query := "?directory=" + url.QueryEscape(directory)
	var session struct {
		ID string `json:"id"`
	}
	if _, err := postJSON(ctx, client, baseURL+"/session"+query, map[string]string{"title": "Codea S3 permission spike"}, &session); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	if session.ID == "" {
		return errors.New("create session response did not contain id")
	}

	eventReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/global/event", nil)
	if err != nil {
		return err
	}
	eventResp, err := client.Do(eventReq)
	if err != nil {
		return fmt.Errorf("subscribe SSE: %w", err)
	}
	defer eventResp.Body.Close()

	if status, err := postJSON(ctx, client, baseURL+"/session/"+session.ID+"/prompt_async"+query, map[string]any{
		"model": map[string]string{"providerID": "codea-s3", "modelID": "fake-tool"},
		"parts": []map[string]string{{"type": "text", "text": "Create the requested marker file using a tool."}},
	}, nil); err != nil || status != http.StatusNoContent {
		return fmt.Errorf("send prompt: HTTP %d: %w", status, err)
	}

	encoder := json.NewEncoder(os.Stdout)
	var permission permissionRequest
	events, err := scanEvents(eventResp.Body, func(event eventEnvelope) (bool, error) {
		if err := encoder.Encode(event); err != nil {
			return false, err
		}
		switch event.Payload.Type {
		case "permission.asked":
			if err := json.Unmarshal(event.Payload.Properties, &permission); err != nil {
				return false, err
			}
			if permission.SessionID != session.ID {
				return false, nil
			}
			status, err := postJSON(ctx, client, baseURL+"/permission/"+permission.ID+"/reply"+query, map[string]string{"reply": reply}, nil)
			if err != nil {
				return false, fmt.Errorf("reply to permission: %w", err)
			}
			fmt.Fprintf(os.Stderr, "permission %s (%s) replied %s (HTTP %d)\n", permission.ID, permission.Permission, reply, status)
		case "session.error":
			var properties sessionError
			if err := json.Unmarshal(event.Payload.Properties, &properties); err != nil {
				return false, err
			}
			if properties.SessionID == session.ID {
				return false, fmt.Errorf("session.error: %s", properties.Error)
			}
		case "session.status":
			var properties sessionStatus
			if err := json.Unmarshal(event.Payload.Properties, &properties); err != nil {
				return false, err
			}
			if properties.SessionID == session.ID && properties.Status.Type == "idle" && permission.ID != "" {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "session %s completed after permission %s and %d SSE events\n", session.ID, permission.ID, len(events))
	return nil
}

func main() {
	baseURL := flag.String("base-url", "http://127.0.0.1:49324", "OpenCode server URL")
	directory := flag.String("directory", ".", "project directory")
	reply := flag.String("reply", "once", "permission reply: once, always, or reject")
	timeout := flag.Duration("timeout", 60*time.Second, "end-to-end timeout")
	flag.Parse()
	if *reply != "once" && *reply != "always" && *reply != "reject" {
		fmt.Fprintln(os.Stderr, "S3 FAIL: invalid reply", *reply)
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := run(ctx, &http.Client{}, strings.TrimRight(*baseURL, "/"), *directory, *reply); err != nil {
		fmt.Fprintln(os.Stderr, "S3 FAIL:", err)
		os.Exit(1)
	}
}
