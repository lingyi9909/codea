package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codea/tui/internal/agent"
	"codea/tui/internal/doctor"
	"codea/tui/internal/opencode"
	runtimedomain "codea/tui/internal/runtime"
	"codea/tui/internal/skill"
	"codea/tui/internal/update"
)

func codeaHomeDir() string {
	if d := os.Getenv("CODEA_HOME"); d != "" { return d }
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codea")
}

func runInitCommand() error {
	if err := doctor.Init(codeaHomeDir(), codeaConfigDir()); err != nil { return err }
	fmt.Printf("Codea 初始化完成：%s\n", codeaHomeDir())
	return nil
}

func recoverInterruptedUpdateIfNeeded(ctx context.Context) error {
	marker := filepath.Join(codeaHomeDir(), "update.in-progress")
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("检查升级恢复标记: %w", err)
	}
	if err := update.RecoverHome(ctx, codeaHomeDir(), codeaConfigDir()); err != nil {
		return fmt.Errorf("恢复未完成升级事务: %w", err)
	}
	return nil
}

func doctorWorkspaceDir() string {
	return filepath.Join(codeaHomeDir(), "doctor-workspace")
}

type doctorProgressRunner interface {
	RunWithProgress(context.Context, func(string, doctor.Category), func(doctor.Result)) doctor.Report
}

func runDoctorService(ctx context.Context, svc doctorProgressRunner, out io.Writer) doctor.Report {
	return svc.RunWithProgress(ctx, func(name string, _ doctor.Category) {
		fmt.Fprintf(out, "正在检查：%s\n", name)
	}, nil)
}

// newDoctorService is the single composition point for both `codea doctor`
// and the TUI `/doctor` command. Runtime bootstrap can differ, but the checks,
// timeouts, version contract, and report semantics stay in one Doctor service.
func newDoctorService(rt runtimedomain.AgentRuntime, runtimeErr error, runtimeURL string) (*doctor.Service, error) {
	if strings.TrimSpace(runtimeURL) == "" {
		runtimeURL = "http://127.0.0.1"
	}
	return doctor.NewDefaultService(doctor.Config{
		HomeDir: codeaHomeDir(), ConfigDir: codeaConfigDir(), Runtime: rt,
		RuntimeStartError: runtimeErr, RuntimeURL: runtimeURL,
		ExpectedOpenCodeVersion: lockedOpenCodeVersion, BehaviorTimeout: 30 * time.Second,
	})
}

func doctorRuntimeURL() string {
	if raw := strings.TrimSpace(os.Getenv("OPENCODE_URL")); raw != "" {
		return raw
	}
	return "http://127.0.0.1"
}

func installedCodeaVersion() string {
	if v := strings.TrimSpace(os.Getenv("CODEA_VERSION")); v != "" {
		return v
	}
	current, err := update.NewPlatformSwitcher(codeaHomeDir()).Current()
	if err != nil || current == "" {
		return "unknown"
	}
	body, err := os.ReadFile(filepath.Join(current, "VERSION"))
	if err != nil {
		return "unknown"
	}
	if v := strings.TrimSpace(string(body)); v != "" {
		return v
	}
	return "unknown"
}

func runDoctorCommand() error {
	fmt.Println("Codea Doctor：正在准备并启动 Runtime...")
	if err := recoverInterruptedUpdateIfNeeded(context.Background()); err != nil {
		return err
	}
	cfgDir := codeaConfigDir()
	var adapter *opencode.OpenCodeAdapter
	cleanup := func() {}
	var runtimeErr error

	mode, err := skill.ResolveSkillMode(os.Getenv("CODEA_SKILL_MODE"))
	if err != nil {
		runtimeErr = fmt.Errorf("解析 Skill 模式: %w", err)
	} else {
		policy := skill.SkillPolicy{Mode: mode, Approved: skill.ParseApprovedSkills(os.Getenv("CODEA_APPROVED_SKILLS"))}
		roots := skillRoots()
		store := skill.NewFileStore(filepath.Join(cfgDir, "codea", "skills.json"))
		targetDir := filepath.Join(cfgDir, "skills")
		doctorRoot := doctorWorkspaceDir()
		if err := skill.SyncEnabled(roots, store, targetDir, policy); err != nil {
			runtimeErr = fmt.Errorf("同步 Skills: %w", err)
		} else if err := writePluginConfig(cfgDir); err != nil {
			runtimeErr = fmt.Errorf("写入 Plugin 配置: %w", err)
		} else if err := agent.Materialize(agentRoot(), filepath.Join(cfgDir, "agents")); err != nil {
			runtimeErr = fmt.Errorf("物化 Agents: %w", err)
		} else if err := os.MkdirAll(doctorRoot, 0o700); err != nil {
			runtimeErr = fmt.Errorf("创建 Doctor 工作区: %w", err)
		} else {
			startedAdapter, startedCleanup, startErr := bootstrapRuntimeAt(cfgDir, mode, doctorRoot)
			adapter = startedAdapter
			runtimeErr = startErr
			if startedCleanup != nil { cleanup = startedCleanup }
		}
	}
	defer cleanup()

	var doctorRuntime runtimedomain.AgentRuntime
	if adapter != nil {
		doctorRuntime = adapter
	}
	svc, err := newDoctorService(doctorRuntime, runtimeErr, doctorRuntimeURL())
	if err != nil { return err }
	report := runDoctorService(context.Background(), svc, os.Stdout)
	fmt.Print(doctor.FormatText(report))
	if report.HasFailures() { return fmt.Errorf("Doctor 检查存在 FAIL") }
	return nil
}
