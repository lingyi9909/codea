package opencode

import (
	"bufio"
	"context"
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
			Data:     []byte(fmt.Sprintf(`{"type":"runtime_error","error":"%s"}`, scanner.Err().Error())),
			Sequence: seq,
		}:
		case <-ctx.Done():
		}
	}
}
