package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"codea/tui/internal/agent"
	"codea/tui/internal/opencode"
	runtimedomain "codea/tui/internal/runtime"
	"codea/tui/internal/skill"
	"codea/tui/internal/supervisor"
	"codea/tui/internal/update"
)

type CandidateRuntimeOptions struct {
	SkillMode      skill.SkillMode
	ApprovedSkills map[string]bool
	ProjectRoot    string
	StartupTimeout time.Duration
	StopTimeout    time.Duration
}

type DefaultCandidateRuntimeFactory struct {
	Options CandidateRuntimeOptions
}

func NewCandidateRuntimeFactory(opts CandidateRuntimeOptions) *DefaultCandidateRuntimeFactory {
	return &DefaultCandidateRuntimeFactory{Options: opts}
}

func (f *DefaultCandidateRuntimeFactory) Start(ctx context.Context, candidate update.Candidate) (runtimedomain.AgentRuntime, string, func(), error) {
	if candidate.VersionDir == "" || candidate.ConfigDir == "" {
		return nil, "", nil, fmt.Errorf("candidate version/config dir required")
	}
	mode := f.Options.SkillMode
	if !mode.Valid() {
		mode = skill.SkillModeStrict
	}
	policy := skill.SkillPolicy{Mode: mode, Approved: f.Options.ApprovedSkills}
	if err := prepareCandidateConfig(candidate, policy); err != nil {
		return nil, "", nil, fmt.Errorf("prepare candidate config: %w", err)
	}

	bin := filepath.Join(candidate.VersionDir, "bin", "opencode")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if st, err := os.Stat(bin); err != nil || st.IsDir() {
		return nil, "", nil, fmt.Errorf("candidate opencode missing: %s", bin)
	}
	projectRoot := f.Options.ProjectRoot
	if projectRoot == "" {
		projectRoot = candidate.VersionDir
	}
	sup := supervisor.NewSupervisor(supervisor.Config{
		OpenCodeBin:     bin,
		ConfigDir:       candidate.ConfigDir,
		ProjectRoot:     projectRoot,
		StartupTimeout:  f.Options.StartupTimeout,
		StopTimeout:     f.Options.StopTimeout,
		CodeaSkillsOnly: mode == skill.SkillModeStrict,
	})
	if err := sup.Start(ctx); err != nil {
		return nil, "", nil, err
	}
	adapter := opencode.NewOpenCodeAdapter(sup.BaseURL(), sup.Username(), sup.Password())
	cleanup := func() { _ = sup.Stop() }
	return adapter, sup.BaseURL(), cleanup, nil
}

func prepareCandidateConfig(candidate update.Candidate, policy skill.SkillPolicy) error {
	if candidate.VersionDir == "" || candidate.ConfigDir == "" {
		return fmt.Errorf("candidate version/config dir required")
	}
	if err := os.MkdirAll(candidate.ConfigDir, 0o700); err != nil {
		return err
	}
	store := skill.NewFileStore(filepath.Join(candidate.ConfigDir, "codea", "skills.json"))
	roots := []skill.Root{{Dir: filepath.Join(candidate.VersionDir, "skills"), Source: skill.SourceCodea}}
	if err := skill.SyncEnabled(roots, store, filepath.Join(candidate.ConfigDir, "skills"), policy); err != nil {
		return fmt.Errorf("sync candidate skills: %w", err)
	}
	if err := agent.Materialize(filepath.Join(candidate.VersionDir, "agents"), filepath.Join(candidate.ConfigDir, "agents")); err != nil {
		return fmt.Errorf("materialize candidate agents: %w", err)
	}
	return mergeCandidatePluginConfig(candidate.ConfigDir, filepath.Join(candidate.VersionDir, "plugins", "index.js"))
}

func mergeCandidatePluginConfig(configDir, bundle string) error {
	st, err := os.Stat(bundle)
	if err != nil || st.IsDir() || st.Size() == 0 {
		return fmt.Errorf("candidate plugin bundle missing: %s", bundle)
	}
	cfgPath := filepath.Join(configDir, "opencode.json")
	cfg := map[string]any{}
	if data, err := os.ReadFile(cfgPath); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("decode candidate opencode.json: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	abs, err := filepath.Abs(bundle)
	if err != nil {
		return err
	}
	u := &url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	cfg["plugin"] = []string{u.String()}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0o600)
}
