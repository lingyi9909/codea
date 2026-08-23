package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"codea/tui/internal/agent"
	"codea/tui/internal/doctor"
	"codea/tui/internal/opencode"
	"codea/tui/internal/skill"
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

func runDoctorCommand() error {
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
		if err := skill.SyncEnabled(roots, store, targetDir, policy); err != nil {
			runtimeErr = fmt.Errorf("同步 Skills: %w", err)
		} else if err := writePluginConfig(cfgDir); err != nil {
			runtimeErr = fmt.Errorf("写入 Plugin 配置: %w", err)
		} else if err := agent.Materialize(agentRoot(), filepath.Join(cfgDir, "agents")); err != nil {
			runtimeErr = fmt.Errorf("物化 Agents: %w", err)
		} else {
			adapter, cleanup, runtimeErr = bootstrapRuntime(cfgDir, mode)
		}
	}
	defer cleanup()

	runtimeURL := os.Getenv("OPENCODE_URL")
	if runtimeURL == "" { runtimeURL = "http://127.0.0.1" }
	svc, err := doctor.NewDefaultService(doctor.Config{
		HomeDir: codeaHomeDir(), ConfigDir: cfgDir, Runtime: adapter,
		RuntimeStartError: runtimeErr, RuntimeURL: runtimeURL,
		ExpectedOpenCodeVersion: "1.18.11", BehaviorTimeout: 30 * time.Second,
	})
	if err != nil { return err }
	report := svc.Run(context.Background())
	fmt.Print(doctor.FormatText(report))
	if report.HasFailures() { return fmt.Errorf("Doctor 检查存在 FAIL") }
	return nil
}
