package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	runtimedomain "codea/tui/internal/runtime"
	"codea/tui/internal/update"
)

type defaultCheckConfig struct {
	home, configDir, releaseRoot string
	runtime                      runtimedomain.AgentRuntime
	runtimeStartErr              error
	runtimeURL, expectedVersion  string
	behaviorTimeout              time.Duration
}

func defaultChecks(c defaultCheckConfig) []Check {
	current := func() (string, error) {
		if c.releaseRoot != "" {
			return c.releaseRoot, nil
		}
		return update.NewPlatformSwitcher(c.home).Current()
	}

	var modelOnce sync.Once
	var modelStatus Status
	var modelDetail string
	modelProbe := func(ctx context.Context) (Status, string) {
		if c.runtimeStartErr != nil || c.runtime == nil {
			return Skip, "未连接 Runtime，无法验证模型"
		}
		modelOnce.Do(func() {
			modelStatus, modelDetail = inferenceProbe(ctx, c.runtime, c.behaviorTimeout)
		})
		return modelStatus, modelDetail
	}

	return []Check{
		staticCheck("发行包完整性", func(context.Context) (Status, string) {
			root, err := current()
			if err != nil {
				return Fail, err.Error()
			}
			info, err := (update.Verifier{}).Verify(root)
			if err != nil {
				return Fail, err.Error()
			}
			return Pass, "Codea " + info.Version + " Manifest/hash 完整"
		}),
		staticCheck("配置 Schema", func(context.Context) (Status, string) {
			p := filepath.Join(c.configDir, "codea", "config.json")
			b, err := os.ReadFile(p)
			if os.IsNotExist(err) {
				return Warn, "config.json 不存在，按 schemaVersion=1 兼容"
			}
			if err != nil {
				return Fail, err.Error()
			}
			var v map[string]any
			if err := json.Unmarshal(b, &v); err != nil {
				return Fail, "config.json JSON 无效: " + err.Error()
			}
			raw, ok := v["schemaVersion"]
			if !ok {
				return Fail, "缺少 schemaVersion"
			}
			n, ok := raw.(float64)
			if !ok || n < 1 || n != float64(int(n)) {
				return Fail, "schemaVersion 必须为正整数"
			}
			return Pass, fmt.Sprintf("schemaVersion=%d", int(n))
		}),
		staticCheck("权限配置", func(context.Context) (Status, string) {
			root, err := current()
			if err != nil {
				return Fail, err.Error()
			}
			p := filepath.Join(root, "config", "opencode", "permissions.json")
			b, err := os.ReadFile(p)
			if err != nil {
				return Fail, err.Error()
			}
			var v map[string]any
			if json.Unmarshal(b, &v) != nil {
				return Fail, "permissions.json JSON 无效"
			}
			agents, ok := v["agents"].(map[string]any)
			if !ok || len(agents) == 0 {
				return Fail, "permissions.json 缺少 agents"
			}
			return Pass, fmt.Sprintf("%d 个 Agent 权限块", len(agents))
		}),
		staticCheck("Skill Manifest", func(context.Context) (Status, string) {
			root, err := current()
			if err != nil {
				return Fail, err.Error()
			}
			return checkChildManifests(filepath.Join(root, "skills"), []string{"SKILL.md"})
		}),
		staticCheck("Agent Manifest", func(context.Context) (Status, string) {
			root, err := current()
			if err != nil {
				return Fail, err.Error()
			}
			return checkChildManifests(filepath.Join(root, "agents"), []string{"manifest.yaml", "agent.md"})
		}),
		staticCheck("Plugin Bundle", func(context.Context) (Status, string) {
			root, err := current()
			if err != nil {
				return Fail, err.Error()
			}
			p := filepath.Join(root, "plugins", "index.js")
			st, err := os.Stat(p)
			if err != nil || st.IsDir() || st.Size() == 0 {
				return Fail, "plugins/index.js 缺失或为空"
			}
			for _, name := range []string{"package.json", "bun.lock", "bun.lockb"} {
				if _, err := os.Stat(filepath.Join(root, "plugins", name)); err == nil {
					return Fail, "运行包不应包含 " + name
				} else if !os.IsNotExist(err) {
					return Fail, err.Error()
				}
			}
			return Pass, fmt.Sprintf("bundle=%d bytes", st.Size())
		}),
		staticCheck("版本兼容", func(context.Context) (Status, string) {
			root, err := current()
			if err != nil {
				return Fail, err.Error()
			}
			b, err := os.ReadFile(filepath.Join(root, "runtime-version.json"))
			if err != nil {
				return Fail, err.Error()
			}
			var v struct {
				OpenCodeVersion string `json:"openCodeVersion"`
			}
			if err := json.Unmarshal(b, &v); err != nil {
				return Fail, err.Error()
			}
			if v.OpenCodeVersion != c.expectedVersion {
				return Fail, fmt.Sprintf("OpenCode=%s，要求=%s", v.OpenCodeVersion, c.expectedVersion)
			}
			return Pass, "OpenCode " + v.OpenCodeVersion
		}),
		connectionCheck("Runtime 健康", func(ctx context.Context) (Status, string) {
			if c.runtimeStartErr != nil {
				return Fail, "Runtime 启动失败: " + c.runtimeStartErr.Error()
			}
			if c.runtime == nil {
				return Skip, "未连接 Runtime"
			}
			h, err := c.runtime.Health(ctx)
			if err != nil {
				return Fail, err.Error()
			}
			if !h.Healthy {
				return Fail, "Runtime healthy=false"
			}
			if h.Version != c.expectedVersion {
				return Fail, fmt.Sprintf("Runtime=%s，要求=%s", h.Version, c.expectedVersion)
			}
			return Pass, "OpenCode " + h.Version + " healthy"
		}),
		connectionCheck("企业 Agent", func(ctx context.Context) (Status, string) {
			if c.runtime == nil {
				return Skip, "未连接 Runtime"
			}
			agents, err := c.runtime.ListAgents(ctx)
			if err != nil {
				return Fail, err.Error()
			}
			set := map[string]bool{}
			for _, a := range agents {
				set[a.Name] = true
			}
			var missing []string
			for _, name := range []string{"code-reviewer", "unit-test-generator", "api-documentation"} {
				if !set[name] {
					missing = append(missing, name)
				}
			}
			if len(missing) > 0 {
				return Fail, "缺少: " + strings.Join(missing, ",")
			}
			return Pass, "3 个企业 Agent 已加载"
		}),
		connectionCheck("模型连接", modelProbe),
		behaviorCheck("SSE", func(ctx context.Context) (Status, string) {
			if c.runtime == nil {
				return Skip, "未连接 Runtime"
			}
			ch, err := c.runtime.Subscribe(ctx)
			if err != nil {
				return Fail, err.Error()
			}
			if ch == nil {
				return Fail, "Subscribe 返回 nil channel"
			}
			return Pass, "全局事件流已建立"
		}),
		behaviorCheck("模型推理", modelProbe),
		networkCheck("Runtime 网络绑定", func(context.Context) (Status, string) {
			if strings.TrimSpace(c.runtimeURL) == "" {
				return Warn, "Runtime URL 未提供，无法验证绑定地址"
			}
			u, err := url.Parse(c.runtimeURL)
			if err != nil {
				return Fail, err.Error()
			}
			host := u.Hostname()
			ip := net.ParseIP(host)
			if host == "localhost" || (ip != nil && ip.IsLoopback()) {
				return Pass, "loopback: " + host
			}
			return Fail, "Runtime 必须绑定 loopback，当前: " + host
		}),
	}
}

func staticCheck(name string, fn func(context.Context) (Status, string)) Check {
	return checkFunc{name: name, category: CategoryStatic, fn: fn}
}
func connectionCheck(name string, fn func(context.Context) (Status, string)) Check {
	return checkFunc{name: name, category: CategoryConnection, fn: fn}
}
func behaviorCheck(name string, fn func(context.Context) (Status, string)) Check {
	return checkFunc{name: name, category: CategoryBehavior, fn: fn}
}
func networkCheck(name string, fn func(context.Context) (Status, string)) Check {
	return checkFunc{name: name, category: CategoryNetwork, fn: fn}
}

func checkChildManifests(root string, required []string) (Status, string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return Fail, err.Error()
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		count++
		for _, name := range required {
			p := filepath.Join(root, entry.Name(), name)
			st, err := os.Stat(p)
			if err != nil || st.IsDir() || st.Size() == 0 {
				return Fail, fmt.Sprintf("%s 缺少 %s", entry.Name(), name)
			}
		}
	}
	if count == 0 {
		return Fail, "没有可用条目"
	}
	return Pass, fmt.Sprintf("%d 个条目", count)
}

func inferenceProbe(parent context.Context, rt runtimedomain.AgentRuntime, timeout time.Duration) (Status, string) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	events, err := rt.Subscribe(ctx)
	if err != nil {
		return Fail, "SSE: " + err.Error()
	}
	if events == nil {
		return Fail, "SSE channel=nil"
	}
	session, err := rt.CreateSession(ctx, runtimedomain.CreateSessionRequest{Title: "Codea Doctor"})
	if err != nil {
		return Fail, "create session: " + err.Error()
	}
	defer rt.Cancel(context.Background(), runtimedomain.SessionID(session.ID))
	req := runtimedomain.PromptRequest{
		MessageID: fmt.Sprintf("doctor-%d", time.Now().UnixNano()),
		Agent:     "general",
		Parts: []runtimedomain.PromptPart{
			runtimedomain.TextPart{Text: "这是 Codea Doctor 健康检查。请只回复 CODEA_DOCTOR_OK，不要调用任何工具。", Synthetic: true},
		},
	}
	if err := rt.Prompt(ctx, runtimedomain.SessionID(session.ID), req); err != nil {
		return Fail, "prompt: " + err.Error()
	}
	for {
		select {
		case <-ctx.Done():
			return Fail, "等待模型响应超时"
		case ev, ok := <-events:
			if !ok {
				return Fail, "SSE 提前关闭"
			}
			if ev.SessionID != "" && ev.SessionID != session.ID {
				continue
			}
			if ev.Error != nil {
				return Fail, ev.Error.Error()
			}
			if strings.TrimSpace(ev.Content) != "" {
				return Pass, "收到模型响应"
			}
		}
	}
}
