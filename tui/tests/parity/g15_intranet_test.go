package parity_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"codea/tui/internal/opencode"
	"codea/tui/internal/runtime"
)

type g15Evidence struct {
	Timestamp       string   `json:"timestamp"`
	SourceCommit    string   `json:"sourceCommit"`
	Passed          bool     `json:"passed"`
	CodeaAgent      string   `json:"codeaAgent"`
	OpenCodeVersion string   `json:"openCodeVersion"`
	MirrorKinds     []string `json:"mirrorKinds"`
	PassedChecks    int      `json:"passedChecks"`
	TotalChecks     int      `json:"totalChecks"`
}

func requiredEnv(t *testing.T, key string) string {
	t.Helper()
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		t.Fatalf("%s is required for G15", key)
	}
	return v
}

func safeMirrorURL(t *testing.T, key string) string {
	t.Helper()
	raw := requiredEnv(t, key)
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		t.Fatalf("%s must be an absolute mirror URL: %q", key, raw)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		t.Fatalf("%s scheme must be http/https", key)
	}
	if strings.ContainsAny(raw, "\"'` \t\r\n;&|") {
		t.Fatalf("%s contains shell-unsafe characters; use a canonical mirror URL", key)
	}
	return strings.TrimRight(raw, "/")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func fileURI(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.ToSlash(abs)
	if goruntime.GOOS == "windows" && len(p) >= 2 && p[1] == ':' {
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func g15ModelServer(t *testing.T, commands map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/models" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"g15","object":"model"}]}`)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		var req struct {
			Messages []struct {
				Role string `json:"role"`
				Content any `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode model request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		lastRole := ""
		if len(req.Messages) > 0 {
			lastRole = req.Messages[len(req.Messages)-1].Role
		}
		serialized, _ := json.Marshal(req.Messages)
		upper := strings.ToUpper(string(serialized))

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, _ := w.(http.Flusher)
		emit := func(v any) {
			data, _ := json.Marshal(v)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if flusher != nil { flusher.Flush() }
		}
		chunk := func(delta any, finish any) map[string]any {
			return map[string]any{
				"id": "chatcmpl-g15", "object": "chat.completion.chunk",
				"created": time.Now().Unix(), "model": "g15",
				"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}},
			}
		}
		emit(chunk(map[string]any{"role": "assistant"}, nil))
		if lastRole == "tool" {
			emit(chunk(map[string]any{"content": "mirror-resolution-ok"}, nil))
			emit(chunk(map[string]any{}, "stop"))
			fmt.Fprint(w, "data: [DONE]\n\n")
			if flusher != nil { flusher.Flush() }
			return
		}
		marker := ""
		for k := range commands {
			if strings.Contains(upper, k) { marker = k; break }
		}
		if marker == "" {
			t.Errorf("unrecognized G15 prompt")
			emit(chunk(map[string]any{"content": "unrecognized"}, nil))
			emit(chunk(map[string]any{}, "stop"))
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}
		callID := "call_" + strings.ToLower(strings.ReplaceAll(marker, "G15_", ""))
		emit(chunk(map[string]any{"tool_calls": []any{map[string]any{
			"id": callID, "type": "function",
			"function": map[string]any{"name": "bash", "arguments": ""},
		}}}, nil))
		args, _ := json.Marshal(map[string]any{"command": commands[marker], "description": "G15 approved intranet dependency mirror resolution"})
		emit(chunk(map[string]any{"tool_calls": []any{map[string]any{
			"index": 0, "function": map[string]any{"arguments": string(args)},
		}}}, nil))
		emit(chunk(map[string]any{}, "tool_calls"))
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil { flusher.Flush() }
	}))
}

func waitHealth(t *testing.T, baseURL, username, password string, proc *exec.Cmd, stderr io.Reader) {
	t.Helper()
	client := &http.Client{Timeout: time.Second}
	for i := 0; i < 160; i++ {
		if proc.ProcessState != nil && proc.ProcessState.Exited() {
			b, _ := io.ReadAll(stderr)
			t.Fatalf("OpenCode exited before healthy: %s", b)
		}
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/global/health", nil)
		req.SetBasicAuth(username, password)
		resp, err := client.Do(req)
		if err == nil {
			var payload struct { Healthy bool `json:"healthy"`; Version string `json:"version"` }
			_ = json.NewDecoder(resp.Body).Decode(&payload)
			resp.Body.Close()
			if payload.Healthy {
				if payload.Version != "1.18.11" { t.Fatalf("OpenCode version=%s want 1.18.11", payload.Version) }
				return
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	b, _ := io.ReadAll(stderr)
	t.Fatalf("OpenCode did not become healthy: %s", b)
}

func runG15GeneralScenario(t *testing.T, adapter *opencode.OpenCodeAdapter, ch <-chan runtime.Event, marker string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	sess, err := adapter.CreateSession(ctx, runtime.CreateSessionRequest{Title: "g15-" + strings.ToLower(marker)})
	if err != nil { t.Fatalf("CreateSession(%s): %v", marker, err) }
	if err := adapter.Prompt(ctx, runtime.SessionID(sess.ID), runtime.PromptRequest{
		Agent: "general", Parts: []runtime.PromptPart{runtime.TextPart{Text: marker}},
	}); err != nil { t.Fatalf("Prompt(%s): %v", marker, err) }

	called, succeeded, answered := false, false, false
	for {
		select {
		case ev, ok := <-ch:
			if !ok { t.Fatalf("event stream closed during %s", marker) }
			if ev.SessionID != sess.ID { continue }
			switch ev.Type {
			case "approval.requested":
				if ev.Approval == nil || ev.Approval.ID == "" { t.Fatalf("%s approval missing ID", marker) }
				if err := adapter.ReplyApproval(ctx, runtime.ApprovalID(ev.Approval.ID), runtime.ApprovalReply{Decision: runtime.ApprovalOnce}); err != nil {
					t.Fatalf("%s approval: %v", marker, err)
				}
			case "tool.called":
				if ev.Tool != nil && ev.Tool.Name == "bash" { called = true }
			case "tool.success":
				if ev.Tool != nil && ev.Tool.Name == "bash" { succeeded = true }
			case "tool.failed":
				if ev.Tool != nil && ev.Tool.Name == "bash" { t.Fatalf("%s dependency command failed", marker) }
			case "answer.delta":
				if strings.TrimSpace(ev.Content) != "" { answered = true }
			}
			if ev.RawType == "session.idle" {
				if !called || !succeeded || !answered {
					t.Fatalf("%s incomplete: called=%v success=%v answer=%v", marker, called, succeeded, answered)
				}
				return
			}
		case <-ctx.Done():
			t.Fatalf("%s timed out: called=%v success=%v answer=%v", marker, called, succeeded, answered)
		}
	}
}

func TestG15IntranetMirrorsThroughGeneralAgent(t *testing.T) {
	if os.Getenv("CODEA_G15_INTRANET") != "1" {
		t.Skip("set CODEA_G15_INTRANET=1 on an approved intranet host")
	}
	sourceCommit := requiredEnv(t, "CODEA_SOURCE_COMMIT")
	opencodeBin := requiredEnv(t, "OPENCODE_BIN")
	pluginBundle := requiredEnv(t, "CODEA_PLUGIN_BUNDLE")
	mavenMirror := safeMirrorURL(t, "CODEA_G15_MAVEN_MIRROR_URL")
	npmMirror := safeMirrorURL(t, "CODEA_G15_NPM_REGISTRY_URL")
	pypiMirror := safeMirrorURL(t, "CODEA_G15_PYPI_INDEX_URL")
	goProxy := safeMirrorURL(t, "CODEA_G15_GOPROXY_URL")

	if v, err := exec.Command(opencodeBin, "--version").CombinedOutput(); err != nil || !strings.Contains(string(v), "1.18.11") {
		t.Fatalf("OPENCODE_BIN must be v1.18.11: %s err=%v", v, err)
	}
	for _, name := range []string{"mvn", "npm", "go"} {
		if _, err := exec.LookPath(name); err != nil { t.Fatalf("%s is required on G15 intranet host", name) }
	}
	pythonCmd := "python3"
	if goruntime.GOOS == "windows" { pythonCmd = "python" }
	if _, err := exec.LookPath(pythonCmd); err != nil { t.Fatalf("%s is required on G15 intranet host", pythonCmd) }

	workspace := t.TempDir()
	writeFile(t, filepath.Join(workspace, "maven", "pom.xml"), `<project xmlns="http://maven.apache.org/POM/4.0.0"><modelVersion>4.0.0</modelVersion><groupId>codea.g15</groupId><artifactId>mirror-smoke</artifactId><version>1.0.0</version><dependencies><dependency><groupId>junit</groupId><artifactId>junit</artifactId><version>4.13.2</version><scope>test</scope></dependency></dependencies></project>`)
	writeFile(t, filepath.Join(workspace, "maven", "settings.xml"), fmt.Sprintf(`<settings><mirrors><mirror><id>codea-approved</id><mirrorOf>*</mirrorOf><url>%s</url></mirror></mirrors></settings>`, mavenMirror))
	writeFile(t, filepath.Join(workspace, "npm", "package.json"), `{"private":true,"dependencies":{"lodash":"4.17.21"}}`)
	writeFile(t, filepath.Join(workspace, "python", "requirements.txt"), "requests==2.32.3\n")
	writeFile(t, filepath.Join(workspace, "go", "go.mod"), "module codea.g15/mirror\n\ngo 1.26\n\nrequire github.com/google/uuid v1.6.0\n")

	commands := map[string]string{
		"G15_MAVEN": "cd maven && mvn -B -s settings.xml -DskipTests dependency:go-offline",
		"G15_NPM": "cd npm && npm install --registry=" + npmMirror + " --ignore-scripts --no-audit --no-fund",
		"G15_PYPI": "cd python && " + pythonCmd + " -m pip install --index-url=" + pypiMirror + " --disable-pip-version-check --no-cache-dir --target=vendor -r requirements.txt",
		"G15_GO": "cd go && go env -w GOPROXY=" + goProxy + " GOSUMDB=off && go mod download all",
	}
	model := g15ModelServer(t, commands)
	defer model.Close()

	configDir := filepath.Join(workspace, ".opencode-g15")
	if err := os.MkdirAll(configDir, 0o700); err != nil { t.Fatal(err) }
	config := map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"model": "codea-g15/g15", "small_model": "codea-g15/g15",
		"enabled_providers": []string{"codea-g15"},
		"permission": map[string]any{"bash": "ask"},
		"plugin": []string{fileURI(t, pluginBundle)},
		"provider": map[string]any{"codea-g15": map[string]any{
			"npm": "@ai-sdk/openai-compatible", "name": "Codea G15 Local Model",
			"options": map[string]any{"baseURL": model.URL + "/v1", "apiKey": "{env:CODEA_G15_API_KEY}"},
			"models": map[string]any{"g15": map[string]any{"name": "G15", "limit": map[string]any{"context": 32768, "output": 4096}}},
		}},
	}
	cfgBytes, _ := json.MarshalIndent(config, "", "  ")
	writeFile(t, filepath.Join(configDir, "opencode.json"), string(cfgBytes)+"\n")

	port := freePort(t)
	username, password := "codea-g15", "codea-g15-pass"
	proc := exec.Command(opencodeBin, "serve", "--hostname", "127.0.0.1", "--port", strconv.Itoa(port))
	proc.Dir = workspace
	isolate := filepath.Join(workspace, ".runtime-home")
	for _, d := range []string{"home", "data", "state", "cache", "config"} {
		if err := os.MkdirAll(filepath.Join(isolate, d), 0o700); err != nil { t.Fatal(err) }
	}
	proc.Env = append(os.Environ(),
		"HOME="+filepath.Join(isolate, "home"),
		"XDG_DATA_HOME="+filepath.Join(isolate, "data"),
		"XDG_STATE_HOME="+filepath.Join(isolate, "state"),
		"XDG_CACHE_HOME="+filepath.Join(isolate, "cache"),
		"XDG_CONFIG_HOME="+filepath.Join(isolate, "config"),
		"OPENCODE_CONFIG_DIR="+configDir,
		"OPENCODE_SERVER_USERNAME="+username,
		"OPENCODE_SERVER_PASSWORD="+password,
		"CODEA_G15_API_KEY=local-g15",
		"OPENCODE_DISABLE_MODELS_FETCH=1",
		"OPENCODE_DISABLE_AUTOUPDATE=1",
		"OPENCODE_DISABLE_EMBEDDED_WEB_UI=1",
		"OPENCODE_DISABLE_LSP_DOWNLOAD=1",
		"OPENCODE_DISABLE_DEFAULT_PLUGINS=1",
	)
	stderrPipe, err := proc.StderrPipe()
	if err != nil { t.Fatal(err) }
	proc.Stdout = io.Discard
	if err := proc.Start(); err != nil { t.Fatal(err) }
	defer func() { _ = proc.Process.Kill(); _, _ = proc.Process.Wait() }()
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitHealth(t, baseURL, username, password, proc, bufio.NewReader(stderrPipe))

	adapter := opencode.NewOpenCodeAdapter(baseURL, username, password)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := adapter.Subscribe(ctx)
	if err != nil { t.Fatal(err) }
	for _, marker := range []string{"G15_MAVEN", "G15_NPM", "G15_PYPI", "G15_GO"} {
		runG15GeneralScenario(t, adapter, ch, marker)
	}

	evidencePath := os.Getenv("CODEA_G15_EVIDENCE")
	if evidencePath == "" { evidencePath = filepath.Join(workspace, "g15-intranet-evidence.json") }
	ev := g15Evidence{
		Timestamp: time.Now().UTC().Format(time.RFC3339), SourceCommit: sourceCommit,
		Passed: true, CodeaAgent: "general", OpenCodeVersion: "1.18.11",
		MirrorKinds: []string{"maven", "npm", "pypi", "go"}, PassedChecks: 4, TotalChecks: 4,
	}
	data, _ := json.MarshalIndent(ev, "", "  ")
	if err := os.MkdirAll(filepath.Dir(evidencePath), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(evidencePath, append(data, '\n'), 0o600); err != nil { t.Fatal(err) }
}
