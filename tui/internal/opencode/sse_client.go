package opencode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SSERawEvent is a raw SSE event with its connection-level sequence number.
type SSERawEvent struct {
	Data     []byte
	Sequence int64
}

// SSEClient subscribes to the OpenCode /global/event SSE stream.
type SSEClient struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

// NewSSEClient creates an SSE client for the given OpenCode server.
func NewSSEClient(baseURL, username, password string) *SSEClient {
	return &SSEClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		client:   &http.Client{},
	}
}

// Subscribe opens a long-lived SSE connection and returns a channel of raw events.
// The caller owns the context lifecycle; cancelling ctx closes the channel cleanly.
// Pre-subscription HTTP/auth errors are returned directly.
// Post-connection Scanner errors emit a final runtime_error before closing.
func (c *SSEClient) Subscribe(ctx context.Context) (<-chan SSERawEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/global/event", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return nil, fmt.Errorf("SSE subscribe returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	ch := make(chan SSERawEvent, 16)
	go c.readLoop(ctx, resp, ch)
	return ch, nil
}

// runtimeErrorEvent builds a properly structured runtime_error SSE event
// using the payload envelope format that MapEvent expects.
type runtimeErrorEvent struct {
	Directory string              `json:"directory"`
	Payload   runtimeErrorPayload `json:"payload"`
}

type runtimeErrorPayload struct {
	Type       string                  `json:"type"`
	Properties runtimeErrorProperties `json:"properties"`
}

type runtimeErrorProperties struct {
	Error        string `json:"error"`
	Code         string `json:"code"`
	Partial      string `json:"partial,omitempty"`
	OriginalSize int    `json:"originalSize,omitempty"`
}

func newRuntimeErrorEvent(errMsg, code string) []byte {
	return newRuntimeErrorEventWithPartial(errMsg, code, "", 0)
}

func newRuntimeErrorEventWithPartial(errMsg, code, partial string, originalSize int) []byte {
	if len(partial) > maxRawSize {
		partial = partial[:maxRawSize]
	}
	evt := runtimeErrorEvent{
		Directory: "",
		Payload: runtimeErrorPayload{
			Type: "runtime_error",
			Properties: runtimeErrorProperties{
				Error:        errMsg,
				Code:         code,
				Partial:      partial,
				OriginalSize: originalSize,
			},
		},
	}
	data, _ := json.Marshal(evt)
	return data
}

func (c *SSEClient) readLoop(ctx context.Context, resp *http.Response, ch chan SSERawEvent) {
	defer resp.Body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 128*1024), 2*1024*1024)

	var seq int64
	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line = event separator
			if len(dataLines) > 0 {
				seq++
				event := SSERawEvent{
					Data:     []byte(strings.Join(dataLines, "\n")),
					Sequence: seq,
				}
				select {
				case ch <- event:
				case <-ctx.Done():
					return
				}
				dataLines = dataLines[:0]
			}
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimPrefix(data, " ")
			dataLines = append(dataLines, data)
		}
	}

	if scanner.Err() != nil && ctx.Err() == nil {
		seq++
		select {
		case ch <- SSERawEvent{
			Data:     newRuntimeErrorEvent(scanner.Err().Error(), "SCANNER_ERROR"),
			Sequence: seq,
		}:
		case <-ctx.Done():
		}
	}

	// Emit residual dataLines on clean EOF — preserve partial content within 16KB.
	if len(dataLines) > 0 && ctx.Err() == nil {
		seq++
		partial := strings.Join(dataLines, "\n")
		originalSize := len(partial)
		select {
		case ch <- SSERawEvent{
			Data:     newRuntimeErrorEventWithPartial("truncated stream: incomplete event", "TRUNCATED_STREAM", partial, originalSize),
			Sequence: seq,
		}:
		case <-ctx.Done():
		}
	}
}
