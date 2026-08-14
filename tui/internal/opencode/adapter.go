package opencode

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"codea/tui/internal/runtime"
)

// OpenCodeAdapter implements runtime.AgentRuntime using the OpenCode HTTP/SSE vendor client.
type OpenCodeAdapter struct {
	httpClient *HTTPClient
	reconnect  *ReconnectingSSEClient
	tracker    *SessionTracker
}

// NewOpenCodeAdapter creates an adapter backed by an OpenCode server at baseURL.
func NewOpenCodeAdapter(baseURL, username, password string) *OpenCodeAdapter {
	a := &OpenCodeAdapter{
		httpClient: NewHTTPClient(baseURL, username, password),
		reconnect:  NewReconnectingSSEClient(NewSSEClient(baseURL, username, password)),
		tracker:    NewSessionTracker(),
	}
	a.reconnect.SetReconnectHook(a.tracker.MakeRecoveryHook(a.httpClient))
	return a
}

// classifyError maps a raw error from the HTTP client to a RuntimeError so
// callers can use errors.As / runtime.IsXxx for stable error discrimination.
func classifyError(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return runtime.NewCancelledError(op, err.Error(), err)
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		vd := httpErrorVendorDetails(httpErr)
		switch {
		case httpErr.StatusCode == http.StatusUnauthorized || httpErr.StatusCode == http.StatusForbidden:
			rerr := runtime.NewAuthError(op, httpErr.Error(), err)
			rerr.VendorDetails = vd
			return rerr
		case httpErr.StatusCode >= 500:
			rerr := runtime.NewTransportError(op, httpErr.Error(), err)
			rerr.VendorDetails = vd
			return rerr
		default:
			rerr := runtime.NewProtocolError(op, httpErr.Error(), err)
			rerr.VendorDetails = vd
			return rerr
		}
	}
	return runtime.NewTransportError(op, err.Error(), err)
}

// httpErrorVendorDetails serializes the raw HTTP error metadata into
// RuntimeError.VendorDetails so callers can inspect status/method/path/body.
func httpErrorVendorDetails(e *HTTPError) json.RawMessage {
	data, _ := json.Marshal(map[string]any{
		"statusCode": e.StatusCode,
		"method":     e.Method,
		"path":       e.Path,
		"body":       string(e.Body),
	})
	return data
}

func (a *OpenCodeAdapter) Health(ctx context.Context) (runtime.HealthInfo, error) {
	resp, err := a.httpClient.Health(ctx)
	if err != nil {
		return runtime.HealthInfo{}, classifyError("Health", err)
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
		return runtime.Session{}, classifyError("CreateSession", err)
	}
	return runtime.Session{ID: session.ID}, nil
}

func (a *OpenCodeAdapter) Prompt(ctx context.Context, sessionID runtime.SessionID, req runtime.PromptRequest) error {
	sid, input, err := MapPromptRequest(sessionID, req)
	if err != nil {
		return err
	}
	return classifyError("Prompt", a.httpClient.SendPrompt(ctx, sid, &input))
}

func (a *OpenCodeAdapter) Subscribe(ctx context.Context) (<-chan runtime.Event, error) {
	rawCh, err := a.reconnect.Subscribe(ctx)
	if err != nil {
		return nil, classifyError("Subscribe", err)
	}

	ch := make(chan runtime.Event, 16)
	go func() {
		defer close(ch)
		for raw := range rawCh {
			// Recovery events injected by ReconnectHook — unwrap directly.
			if ev, ok := unwrapRecoveryEvent(raw.Data); ok {
				a.tracker.Record(ev)
				if !sendRuntimeEvent(ctx, ch, ev) {
					return
				}
				continue
			}

			// Disconnect event — record and forward.
			if IsSSEDisconnect(raw) {
				ev, _ := MapEvent(raw.Data, raw.Sequence)
				a.tracker.Record(ev)
				if !sendRuntimeEvent(ctx, ch, ev) {
					return
				}
				continue
			}

			// Live event — map, record, forward.
			ev, _ := MapEvent(raw.Data, raw.Sequence)
			// Dedup boundary: a live event that raced into the reconnect buffer
			// during recovery and duplicates a just-compensated message/part is
			// suppressed so the Application sees it exactly once.
			if a.tracker.ShouldSuppressLive(ev) {
				continue
			}
			a.tracker.Record(ev)
			if !sendRuntimeEvent(ctx, ch, ev) {
				return
			}
		}
	}()

	return ch, nil
}

func sendRuntimeEvent(ctx context.Context, ch chan<- runtime.Event, ev runtime.Event) bool {
	select {
	case ch <- ev:
		return true
	default:
		// Channel full — block-send Backpressure error, then original event.
		// Blocking the backpressure event guarantees it is delivered before
		// the stalled event, giving consumers visibility into backpressure.
		bpEvent := runtime.Event{
			Type:  CodeaEventRuntimeError,
			Error: runtime.NewBackpressureError("EventChannel", "channel full, backpressure applied", nil),
		}
		select {
		case ch <- bpEvent:
		case <-ctx.Done():
			return false
		}
		select {
		case ch <- ev:
			return true
		case <-ctx.Done():
			return false
		}
	}
}

func (a *OpenCodeAdapter) ReplyApproval(ctx context.Context, approvalID runtime.ApprovalID, reply runtime.ApprovalReply) error {
	input := MapApprovalReply(reply)
	return classifyError("ReplyApproval", a.httpClient.ApprovePermission(ctx, string(approvalID), &input))
}

func (a *OpenCodeAdapter) Cancel(ctx context.Context, sessionID runtime.SessionID) error {
	return classifyError("Cancel", a.httpClient.AbortSession(ctx, string(sessionID)))
}

func (a *OpenCodeAdapter) ListAgents(ctx context.Context) ([]runtime.Agent, error) {
	agents, err := a.httpClient.ListAgents(ctx)
	if err != nil {
		return nil, classifyError("ListAgents", err)
	}
	result := make([]runtime.Agent, len(agents))
	for i, a := range agents {
		result[i] = runtime.Agent{Name: a.Name, Mode: a.Mode}
	}
	return result, nil
}

func (a *OpenCodeAdapter) ListSessions(ctx context.Context) ([]runtime.Session, error) {
	infos, err := a.httpClient.GetSessionStatus(ctx)
	if err != nil {
		return nil, classifyError("ListSessions", err)
	}
	result := make([]runtime.Session, len(infos))
	for i, info := range infos {
		result[i] = MapSession(info)
	}
	return result, nil
}

func (a *OpenCodeAdapter) Capabilities() runtime.RuntimeCapabilities {
	return OpenCodeCapabilities()
}
