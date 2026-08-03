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

type sessionStatusProperties struct {
	SessionID string `json:"sessionID"`
	Status    struct {
		Type string `json:"type"`
	} `json:"status"`
}

type sessionIdleProperties struct {
	SessionID string `json:"sessionID"`
}

func readEvents(r io.Reader, targetSessionID string) ([]eventEnvelope, error) {
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

		switch event.Payload.Type {
		case "session.status":
			var properties sessionStatusProperties
			if err := json.Unmarshal(event.Payload.Properties, &properties); err != nil {
				return nil, fmt.Errorf("decode session.status properties: %w", err)
			}
			if properties.SessionID == targetSessionID && properties.Status.Type == "idle" {
				return events, nil
			}
		case "session.idle":
			var properties sessionIdleProperties
			if err := json.Unmarshal(event.Payload.Properties, &properties); err != nil {
				return nil, fmt.Errorf("decode session.idle properties: %w", err)
			}
			if properties.SessionID == targetSessionID {
				return events, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read SSE stream: %w", err)
	}
	return nil, errors.New("SSE stream ended before target session became idle")
}

func postJSON(ctx context.Context, client *http.Client, endpoint string, body any, out any) (int, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
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
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return resp.StatusCode, fmt.Errorf("POST %s returned %s: %s", endpoint, resp.Status, strings.TrimSpace(string(payload)))
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

func run(ctx context.Context, client *http.Client, baseURL, directory, providerID, modelID, prompt string) error {
	query := "?directory=" + url.QueryEscape(directory)
	var session struct {
		ID string `json:"id"`
	}
	status, err := postJSON(ctx, client, baseURL+"/session"+query, map[string]any{
		"title": "Codea S2 protocol spike",
	}, &session)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	if session.ID == "" {
		return errors.New("create session response did not contain id")
	}
	fmt.Fprintf(os.Stderr, "created session %s (HTTP %d)\n", session.ID, status)

	eventReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/global/event", nil)
	if err != nil {
		return err
	}
	eventResp, err := client.Do(eventReq)
	if err != nil {
		return fmt.Errorf("subscribe SSE: %w", err)
	}
	defer eventResp.Body.Close()
	if eventResp.StatusCode != http.StatusOK {
		return fmt.Errorf("subscribe SSE returned %s", eventResp.Status)
	}

	status, err = postJSON(ctx, client, baseURL+"/session/"+session.ID+"/prompt_async"+query, map[string]any{
		"model": map[string]string{
			"providerID": providerID,
			"modelID":    modelID,
		},
		"parts": []map[string]string{{"type": "text", "text": prompt}},
	}, nil)
	if err != nil {
		return fmt.Errorf("send prompt: %w", err)
	}
	if status != http.StatusNoContent {
		return fmt.Errorf("send prompt returned HTTP %d, want 204", status)
	}
	fmt.Fprintf(os.Stderr, "prompt accepted (HTTP %d)\n", status)

	events, err := readEvents(eventResp.Body, session.ID)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "session %s completed after %d SSE events\n", session.ID, len(events))
	return nil
}

func main() {
	baseURL := flag.String("base-url", "http://127.0.0.1:49321", "OpenCode server URL")
	directory := flag.String("directory", ".", "project directory sent to OpenCode")
	providerID := flag.String("provider", "codea-s2", "provider ID")
	modelID := flag.String("model", "fake-chat", "model ID")
	prompt := flag.String("prompt", "Reply with exactly: hello from s2", "prompt text")
	timeout := flag.Duration("timeout", 60*time.Second, "end-to-end timeout")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	client := &http.Client{}
	if err := run(ctx, client, strings.TrimRight(*baseURL, "/"), *directory, *providerID, *modelID, *prompt); err != nil {
		fmt.Fprintln(os.Stderr, "S2 FAIL:", err)
		os.Exit(1)
	}
}
