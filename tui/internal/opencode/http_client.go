package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout        = 30 * time.Second
	agentColdStartHTTPTimeout = 90 * time.Second
)

// HTTPError carries a typed HTTP error with status code for classification.
type HTTPError struct {
	StatusCode int
	Method     string
	Path       string
	Body       []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%s %s returned HTTP %d: %s", e.Method, e.Path, e.StatusCode, strings.TrimSpace(string(e.Body)))
}

type HTTPClient struct {
	baseURL     string
	username    string
	password    string
	directory   string
	client      *http.Client
	agentClient *http.Client
}

func NewHTTPClient(baseURL, username, password string) *HTTPClient {
	return NewHTTPClientForDirectory(baseURL, username, password, "")
}

// NewHTTPClientForDirectory creates a vendor client whose instance-scoped
// requests are explicitly bound to directory. OpenCode otherwise falls back to
// the server process cwd, which can accidentally treat a user's home directory
// as the project when Codea Doctor is launched outside a repository.
func NewHTTPClientForDirectory(baseURL, username, password, directory string) *HTTPClient {
	return newHTTPClientWithTimeouts(
		baseURL,
		username,
		password,
		directory,
		defaultHTTPTimeout,
		agentColdStartHTTPTimeout,
	)
}

// newHTTPClientWithTimeouts keeps the ordinary request budget independent from
// the /agent cold-start budget. OpenCode initializes the instance/plugin/agent
// stack on the first /agent request, which is materially slower on Windows than
// health and steady-state calls. Keeping a dedicated client prevents a slow
// cold start from weakening timeouts for every Runtime operation.
func newHTTPClientWithTimeouts(baseURL, username, password, directory string, requestTimeout, agentTimeout time.Duration) *HTTPClient {
	return &HTTPClient{
		baseURL:     strings.TrimRight(baseURL, "/"),
		username:    username,
		password:    password,
		directory:   directory,
		client:      &http.Client{Timeout: requestTimeout},
		agentClient: &http.Client{Timeout: agentTimeout},
	}
}

func (client *HTTPClient) Health(ctx context.Context) (*OpenCodeGlobalHealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/global/health", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.do(req, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var health OpenCodeGlobalHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, err
	}
	return &health, nil
}

func (client *HTTPClient) CreateSession(ctx context.Context, input *OpenCodeSessionCreateRequest) (*OpenCodeSession, error) {
	var session OpenCodeSession
	if err := client.doJSON(ctx, http.MethodPost, "/session", input, &session, http.StatusOK); err != nil {
		return nil, err
	}
	return &session, nil
}

func (client *HTTPClient) SendPrompt(ctx context.Context, sessionID string, input *OpenCodeSessionPromptAsyncRequest) error {
	sessionPath, err := pathSegment("session ID", sessionID)
	if err != nil {
		return err
	}
	return client.doJSON(ctx, http.MethodPost, "/session/"+sessionPath+"/prompt_async", input, nil, http.StatusNoContent)
}

func (client *HTTPClient) ApprovePermission(ctx context.Context, requestID string, input *OpenCodePermissionReplyRequest) error {
	requestPath, err := pathSegment("permission request ID", requestID)
	if err != nil {
		return err
	}
	var accepted OpenCodePermissionReplyResponse
	if err := client.doJSON(ctx, http.MethodPost, "/permission/"+requestPath+"/reply", input, &accepted, http.StatusOK); err != nil {
		return err
	}
	if !accepted {
		return fmt.Errorf("permission %s reply was not accepted", requestID)
	}
	return nil
}

func (client *HTTPClient) AbortSession(ctx context.Context, sessionID string) error {
	sessionPath, err := pathSegment("session ID", sessionID)
	if err != nil {
		return err
	}
	var aborted OpenCodeSessionAbortResponse
	if err := client.doJSON(ctx, http.MethodPost, "/session/"+sessionPath+"/abort", nil, &aborted, http.StatusOK); err != nil {
		return err
	}
	if !aborted {
		return fmt.Errorf("session %s was not aborted", sessionID)
	}
	return nil
}

// GetSessionStatus returns all sessions. Used for recovery after SSE reconnect.
// Calls GET /session which returns a JSON array of session info objects.
func (client *HTTPClient) GetSessionStatus(ctx context.Context) ([]OpenCodeSessionV2Info, error) {
	var resp []OpenCodeSessionV2Info
	if err := client.doJSON(ctx, http.MethodGet, "/session", nil, &resp, http.StatusOK); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetSessionMessages returns messages for a session. Used for recovery after SSE reconnect.
// The OpenCode API returns a raw JSON array (not wrapped in {data: [...]}).
func (client *HTTPClient) GetSessionMessages(ctx context.Context, sessionID string) ([]OpenCodeSessionMessage, error) {
	sp, err := pathSegment("session ID", sessionID)
	if err != nil {
		return nil, err
	}
	var resp []OpenCodeSessionMessage
	if err := client.doJSON(ctx, http.MethodGet, "/session/"+sp+"/message", nil, &resp, http.StatusOK); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetSession returns a single session by ID.
func (client *HTTPClient) GetSession(ctx context.Context, sessionID string) (*OpenCodeSessionV2Info, error) {
	sp, err := pathSegment("session ID", sessionID)
	if err != nil {
		return nil, err
	}
	var resp OpenCodeV2SessionGetResponse
	if err := client.doJSON(ctx, http.MethodGet, "/session/"+sp, nil, &resp, http.StatusOK); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (client *HTTPClient) ListAgents(ctx context.Context) ([]OpenCodeAgent, error) {
	var agents OpenCodeAppAgentsResponse
	requestClient := client.agentClient
	if requestClient == nil {
		requestClient = client.client
	}
	scoped := *client
	scoped.client = requestClient
	if err := scoped.doJSON(ctx, http.MethodGet, "/agent", nil, &agents, http.StatusOK); err != nil {
		return nil, err
	}
	return []OpenCodeAgent(agents), nil
}

func pathSegment(label, value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", label)
	}
	return url.PathEscape(value), nil
}

func (client *HTTPClient) doJSON(ctx context.Context, method, path string, input, output any, expectedStatus ...int) error {
	var body io.Reader
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode %s %s request: %w", method, path, err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, body)
	if err != nil {
		return err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.do(req, expectedStatus...)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if output == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		return fmt.Errorf("decode %s %s response: %w", method, path, err)
	}
	return nil
}

func (client *HTTPClient) do(req *http.Request, expectedStatus ...int) (*http.Response, error) {
	req.Header.Set("Accept", "application/json")
	if client.directory != "" && !strings.HasPrefix(req.URL.Path, "/global/") {
		req.Header.Set("x-opencode-directory", client.directory)
	}
	if client.username != "" {
		req.SetBasicAuth(client.username, client.password)
	}
	resp, err := client.client.Do(req)
	if err != nil {
		return nil, err
	}
	for _, status := range expectedStatus {
		if resp.StatusCode == status {
			return resp, nil
		}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	return nil, &HTTPError{
		StatusCode: resp.StatusCode,
		Method:     req.Method,
		Path:       req.URL.Path,
		Body:       body,
	}
}
