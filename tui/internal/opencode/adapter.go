package opencode

import (
	"context"

	"codea/tui/internal/runtime"
)

// OpenCodeAdapter implements runtime.AgentRuntime using the OpenCode HTTP/SSE vendor client.
type OpenCodeAdapter struct {
	httpClient *HTTPClient
	sseClient  *SSEClient
}

// NewOpenCodeAdapter creates an adapter backed by an OpenCode server at baseURL.
func NewOpenCodeAdapter(baseURL, username, password string) *OpenCodeAdapter {
	return &OpenCodeAdapter{
		httpClient: NewHTTPClient(baseURL, username, password),
		sseClient:  NewSSEClient(baseURL, username, password),
	}
}

func (a *OpenCodeAdapter) Health(ctx context.Context) (runtime.HealthInfo, error) {
	resp, err := a.httpClient.Health(ctx)
	if err != nil {
		return runtime.HealthInfo{}, err
	}
	return runtime.HealthInfo{
		Healthy: resp.Healthy,
		Version: resp.Version,
	}, nil
}

func (a *OpenCodeAdapter) CreateSession(ctx context.Context, req runtime.CreateSessionRequest) (runtime.Session, error) {
	input := MapCreateSessionRequest(req)
	session, err := a.httpClient.CreateSession(ctx, &input)
	if err != nil {
		return runtime.Session{}, err
	}
	return runtime.Session{ID: session.ID}, nil
}

func (a *OpenCodeAdapter) Prompt(ctx context.Context, sessionID runtime.SessionID, req runtime.PromptRequest) error {
	sid, input, err := MapPromptRequest(sessionID, req)
	if err != nil {
		return err
	}
	return a.httpClient.SendPrompt(ctx, sid, &input)
}

func (a *OpenCodeAdapter) Subscribe(ctx context.Context) (<-chan runtime.Event, error) {
	rawCh, err := a.sseClient.Subscribe(ctx)
	if err != nil {
		return nil, err
	}

	ch := make(chan runtime.Event, 16)
	go func() {
		defer close(ch)
		for raw := range rawCh {
			event, err := MapEvent(raw.Data, raw.Sequence)
			if err != nil {
				event = runtime.Event{
					Type:     "_unparseable_",
					Sequence: raw.Sequence,
					Raw:      raw.Data,
				}
			}
			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (a *OpenCodeAdapter) ReplyApproval(ctx context.Context, approvalID runtime.ApprovalID, reply runtime.ApprovalReply) error {
	input := MapApprovalReply(reply)
	return a.httpClient.ApprovePermission(ctx, string(approvalID), &input)
}

func (a *OpenCodeAdapter) Cancel(ctx context.Context, sessionID runtime.SessionID) error {
	return a.httpClient.AbortSession(ctx, string(sessionID))
}

func (a *OpenCodeAdapter) ListAgents(ctx context.Context) ([]runtime.Agent, error) {
	agents, err := a.httpClient.ListAgents(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]runtime.Agent, len(agents))
	for i, a := range agents {
		result[i] = runtime.Agent{Name: a.Name, Mode: a.Mode}
	}
	return result, nil
}

func (a *OpenCodeAdapter) Capabilities() runtime.RuntimeCapabilities {
	return OpenCodeCapabilities()
}
