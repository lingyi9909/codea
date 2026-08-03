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

type reasoningResult struct {
	Reasoning    string `json:"reasoning"`
	Answer       string `json:"answer"`
	HasThinkTags bool   `json:"hasThinkTags"`
}

type partUpdated struct {
	Part struct {
		SessionID string `json:"sessionID"`
		Type      string `json:"type"`
		Text      string `json:"text"`
	} `json:"part"`
}

type sessionStatus struct {
	SessionID string `json:"sessionID"`
	Status    struct {
		Type string `json:"type"`
	} `json:"status"`
}

func readReasoningAndAnswer(r io.Reader, targetSessionID string) (reasoningResult, []eventEnvelope, error) {
	var result reasoningResult
	events := make([]eventEnvelope, 0, 32)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
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
			return result, events, fmt.Errorf("decode SSE event: %w", err)
		}
		events = append(events, event)
		switch event.Payload.Type {
		case "message.part.updated":
			var properties partUpdated
			if err := json.Unmarshal(event.Payload.Properties, &properties); err != nil {
				return result, events, err
			}
			if properties.Part.SessionID == targetSessionID {
				switch properties.Part.Type {
				case "reasoning":
					result.Reasoning = properties.Part.Text
				case "text":
					result.Answer = properties.Part.Text
				}
			}
		case "session.status":
			var properties sessionStatus
			if err := json.Unmarshal(event.Payload.Properties, &properties); err != nil {
				return result, events, err
			}
			if properties.SessionID == targetSessionID && properties.Status.Type == "idle" {
				result.HasThinkTags = strings.Contains(result.Reasoning, "<think>") || strings.Contains(result.Answer, "<think>")
				if result.Reasoning == "" || result.Answer == "" {
					return result, events, errors.New("session became idle without both reasoning and answer parts")
				}
				return result, events, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return result, events, err
	}
	return result, events, errors.New("SSE ended before target session became idle")
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

func run(ctx context.Context, client *http.Client, baseURL, directory string) error {
	query := "?directory=" + url.QueryEscape(directory)
	var session struct {
		ID string `json:"id"`
	}
	if _, err := postJSON(ctx, client, baseURL+"/session"+query, map[string]string{"title": "Codea S4 reasoning spike"}, &session); err != nil {
		return err
	}
	eventReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/global/event", nil)
	if err != nil {
		return err
	}
	eventResp, err := client.Do(eventReq)
	if err != nil {
		return err
	}
	defer eventResp.Body.Close()
	status, err := postJSON(ctx, client, baseURL+"/session/"+session.ID+"/prompt_async"+query, map[string]any{
		"model": map[string]string{"providerID": "codea-s4", "modelID": "fake-reasoning"},
		"parts": []map[string]string{{"type": "text", "text": "Reason briefly, then answer."}},
	}, nil)
	if err != nil || status != http.StatusNoContent {
		return fmt.Errorf("prompt: HTTP %d: %w", status, err)
	}
	result, events, err := readReasoningAndAnswer(eventResp.Body, session.ID)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "session %s reasoning=%q answer=%q think_tags=%t events=%d\n", session.ID, result.Reasoning, result.Answer, result.HasThinkTags, len(events))
	return nil
}

func main() {
	baseURL := flag.String("base-url", "http://127.0.0.1:49325", "OpenCode server URL")
	directory := flag.String("directory", ".", "project directory")
	timeout := flag.Duration("timeout", 60*time.Second, "end-to-end timeout")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := run(ctx, &http.Client{}, strings.TrimRight(*baseURL, "/"), *directory); err != nil {
		fmt.Fprintln(os.Stderr, "S4 FAIL:", err)
		os.Exit(1)
	}
}
