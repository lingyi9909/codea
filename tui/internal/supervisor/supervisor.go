package supervisor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"codea/tui/internal/runtime"
)

const (
	defaultHostname       = "127.0.0.1"
	defaultUsername       = "opencode"
	defaultStartupTimeout = 30 * time.Second
	defaultStopTimeout    = 5 * time.Second
	readyInterval         = 200 * time.Millisecond
)

// probeClient bounds each readiness check so a hung server cannot stall the
// polling loop past its own deadline.
var probeClient = &http.Client{Timeout: 2 * time.Second}

// Config configures a RuntimeSupervisor.
type Config struct {
	OpenCodeBin    string
	Hostname       string // forced to 127.0.0.1 (loopback-only; V1 has no remote runtime)
	Port           int    // 0 selects a free local port
	ConfigDir      string
	ProjectRoot    string
	StartupTimeout time.Duration
	StopTimeout    time.Duration
	// CodeaSkillsOnly additionally disables project-skill discovery (strict
	// mode). User and external (.claude/.agents) skills are isolated in BOTH
	// modes by the always-on env in buildEnv.
	CodeaSkillsOnly bool
}

// Supervisor owns the OpenCode process lifecycle. Start/Stop/Status live here,
// not on AgentRuntime.
type Supervisor struct {
	config Config

	mu       sync.Mutex
	status   runtime.RuntimeStatus
	cmd      *exec.Cmd
	port     int
	username string
	password string
	lastErr  error
	exitCh   chan struct{}
	runID    uint64
}

func NewSupervisor(config Config) *Supervisor {
	// Hard-lock loopback: V1 has no remote-runtime requirement, so a caller
	// value like 0.0.0.0 would expose the runtime on the LAN. Ignore it.
	config.Hostname = defaultHostname
	if config.StartupTimeout == 0 {
		config.StartupTimeout = defaultStartupTimeout
	}
	if config.StopTimeout == 0 {
		config.StopTimeout = defaultStopTimeout
	}
	return &Supervisor{
		config:   config,
		status:   runtime.RuntimeStopped,
		username: defaultUsername,
	}
}

func (s *Supervisor) Status() runtime.RuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Supervisor) Port() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

func (s *Supervisor) Username() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.username
}

func (s *Supervisor) Password() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.password
}

func (s *Supervisor) BaseURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.port == 0 {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", s.config.Hostname, s.port)
}

func (s *Supervisor) LastError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastErr
}

// Start launches `opencode serve`, waits until the runtime reports healthy,
// and leaves status Healthy. It returns an error on startup failure and sets
// status Crashed.
func (s *Supervisor) Start(ctx context.Context) error {
	cmd, exitCh, runID, err := s.startProcess()
	if err != nil {
		return err
	}

	go s.monitor(cmd, exitCh, runID)

	if err := s.waitForReady(ctx); err != nil {
		s.cleanupFailedStart(cmd, exitCh, runID, err)
		return err
	}

	// Only transition to Healthy if this run is still the current one and the
	// monitor has not already reconciled a crash. Prevents a stale Healthy from
	// overwriting a concurrent Crashed write.
	if !s.markHealthy(runID) {
		return fmt.Errorf("opencode exited before startup completed")
	}
	return nil
}

// markHealthy transitions Starting → Healthy only if runID is still the current
// run and status is still Starting. Returns false (leaving status untouched)
// if the monitor already moved to Crashed/Stopped or a newer run took over.
func (s *Supervisor) markHealthy(runID uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runID != runID || s.status != runtime.RuntimeStarting {
		return false
	}
	s.status = runtime.RuntimeHealthy
	return true
}

// startProcess performs the guarded state transition and process spawn under
// one lock acquisition, so a concurrent Stop can never observe Starting with a
// nil cmd.
func (s *Supervisor) startProcess() (*exec.Cmd, chan struct{}, uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.status {
	case runtime.RuntimeStarting, runtime.RuntimeHealthy:
		return nil, nil, 0, fmt.Errorf("runtime already running (status %s)", s.status)
	}
	s.status = runtime.RuntimeStarting

	password, err := generatePassword()
	if err != nil {
		s.status = runtime.RuntimeCrashed
		s.lastErr = err
		return nil, nil, 0, err
	}

	port := s.config.Port
	if port == 0 {
		port, err = findFreePort()
		if err != nil {
			s.status = runtime.RuntimeCrashed
			s.lastErr = err
			return nil, nil, 0, err
		}
	}

	cmd := exec.Command(s.config.OpenCodeBin, buildArgs(s.config, port)...)
	cmd.Env = buildEnv(s.config, s.username, password)
	if s.config.ProjectRoot != "" {
		cmd.Dir = s.config.ProjectRoot
	}
	configureProcess(cmd)

	if err := cmd.Start(); err != nil {
		s.status = runtime.RuntimeCrashed
		s.lastErr = fmt.Errorf("start opencode: %w", err)
		return nil, nil, 0, s.lastErr
	}

	if err := attachProcess(cmd); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		s.status = runtime.RuntimeCrashed
		s.lastErr = fmt.Errorf("attach process: %w", err)
		return nil, nil, 0, s.lastErr
	}

	s.runID++
	runID := s.runID
	exitCh := make(chan struct{})
	s.cmd = cmd
	s.port = port
	s.password = password
	s.lastErr = nil
	s.exitCh = exitCh

	return cmd, exitCh, runID, nil
}

// Stop gracefully terminates the process group, waits up to StopTimeout, then
// force-kills. It is idempotent and safe on a Stopped or Crashed supervisor.
func (s *Supervisor) Stop() error {
	s.mu.Lock()
	switch s.status {
	case runtime.RuntimeStopped, runtime.RuntimeCrashed:
		s.mu.Unlock()
		return nil
	case runtime.RuntimeStopping:
		exitCh := s.exitCh
		s.mu.Unlock()
		if exitCh != nil {
			<-exitCh
		}
		return nil
	}

	s.status = runtime.RuntimeStopping
	cmd := s.cmd
	exitCh := s.exitCh
	timeout := s.config.StopTimeout
	s.mu.Unlock()

	stopProcess(cmd, exitCh, timeout)
	return nil
}

// monitor is the sole caller of cmd.Wait(). It reconciles the terminal state
// once the process exits and closes exitCh to release Stop/Start waiters.
func (s *Supervisor) monitor(cmd *exec.Cmd, exitCh chan struct{}, runID uint64) {
	err := cmd.Wait()
	detachProcess(cmd)

	s.mu.Lock()
	if s.runID != runID {
		// A newer run has taken over; this stale monitor must not touch status.
		s.mu.Unlock()
		close(exitCh)
		return
	}
	if s.status == runtime.RuntimeStopping {
		s.status = runtime.RuntimeStopped
	} else {
		s.status = runtime.RuntimeCrashed
		s.lastErr = fmt.Errorf("opencode exited unexpectedly: %w", err)
	}
	s.mu.Unlock()
	close(exitCh)
}

// cleanupFailedStart terminates a process that started but never became ready.
// If Stop() already took over shutdown, or a newer run has started, it does
// nothing.
func (s *Supervisor) cleanupFailedStart(cmd *exec.Cmd, exitCh chan struct{}, runID uint64, err error) {
	s.mu.Lock()
	if s.runID != runID || s.status == runtime.RuntimeStopping || s.status == runtime.RuntimeStopped {
		s.mu.Unlock()
		return
	}
	s.status = runtime.RuntimeStopping
	s.mu.Unlock()

	stopProcess(cmd, exitCh, s.config.StopTimeout)

	s.mu.Lock()
	if s.runID == runID {
		s.status = runtime.RuntimeCrashed
		s.lastErr = err
	}
	s.mu.Unlock()
}

func (s *Supervisor) waitForReady(ctx context.Context) error {
	s.mu.Lock()
	exitCh := s.exitCh
	s.mu.Unlock()

	deadline := time.NewTimer(s.config.StartupTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(readyInterval)
	defer ticker.Stop()

	baseURL := fmt.Sprintf("http://%s:%d", s.config.Hostname, s.port)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("startup timeout after %v", s.config.StartupTimeout)
		case <-exitCh:
			return fmt.Errorf("opencode exited before becoming ready")
		case <-ticker.C:
			if probeReady(ctx, baseURL, s.username, s.password) == nil {
				return nil
			}
		}
	}
}

func generatePassword() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func findFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("find free port: %w", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

// buildArgs returns the fixed `opencode serve` invocation. The hostname is
// hard-locked to loopback (V1 has no remote runtime), so config.Hostname is
// deliberately ignored to prevent a wildcard bind.
func buildArgs(_ Config, port int) []string {
	return []string{
		"serve",
		"--hostname", defaultHostname,
		"--port", fmt.Sprintf("%d", port),
	}
}

func buildEnv(config Config, username, password string) []string {
	env := append(os.Environ(),
		"OPENCODE_CONFIG_DIR="+config.ConfigDir,
		"OPENCODE_SERVER_USERNAME="+username,
		"OPENCODE_SERVER_PASSWORD="+password,
		// Offline env locked by Task 1: prevent models fetch, autoupdate, web
		// UI, LSP download and default plugin install during startup.
		"OPENCODE_DISABLE_CLAUDE_CODE=1",
		"OPENCODE_DISABLE_MODELS_FETCH=1",
		"OPENCODE_DISABLE_AUTOUPDATE=1",
		"OPENCODE_DISABLE_EMBEDDED_WEB_UI=1",
		"OPENCODE_DISABLE_LSP_DOWNLOAD=1",
		"OPENCODE_DISABLE_DEFAULT_PLUGINS=1",
		// Task 1 S6 isolation baseline (BOTH modes): external (.claude/.agents)
		// skills are disabled, and the native user skills dir
		// (~/.config/opencode/skills) is isolated by pointing XDG_CONFIG_HOME
		// away from ~/.config. Project skills stay discoverable unless strict
		// also sets OPENCODE_DISABLE_PROJECT_CONFIG.
		"OPENCODE_DISABLE_EXTERNAL_SKILLS=1",
		"XDG_CONFIG_HOME="+filepath.Join(config.ConfigDir, "xdg", "config"),
		"XDG_DATA_HOME="+filepath.Join(config.ConfigDir, "xdg", "data"),
		"XDG_CACHE_HOME="+filepath.Join(config.ConfigDir, "xdg", "cache"),
		"XDG_STATE_HOME="+filepath.Join(config.ConfigDir, "xdg", "state"),
	)
	if config.CodeaSkillsOnly {
		env = append(env,
			"OPENCODE_DISABLE_PROJECT_CONFIG=1",
		)
	}
	return env
}

// healthResponse is the supervisor-local view of GET /global/health. Kept local
// so the supervisor package does not import the OpenCode vendor DTO layer.
type healthResponse struct {
	Healthy bool `json:"healthy"`
}

// probeReady performs a single readiness check against /global/health with
// Basic Auth. It returns nil only when the endpoint answers 200 and healthy.
func probeReady(ctx context.Context, baseURL, username, password string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/global/health", nil)
	if err != nil {
		return err
	}
	if username != "" {
		req.SetBasicAuth(username, password)
	}
	resp, err := probeClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health returned HTTP %d", resp.StatusCode)
	}
	var h healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return fmt.Errorf("decode health: %w", err)
	}
	if !h.Healthy {
		return fmt.Errorf("health not ready")
	}
	return nil
}

// stopProcess sends a graceful terminate to the process group, waits up to
// timeout for exit, then force-kills the group as a fallback.
func stopProcess(cmd *exec.Cmd, exitCh chan struct{}, timeout time.Duration) {
	_ = terminateProcess(cmd)
	select {
	case <-exitCh:
	case <-time.After(timeout):
		killProcess(cmd)
		<-exitCh
	}
}
