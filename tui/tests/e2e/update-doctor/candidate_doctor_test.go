package updatedoctor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codea/tui/internal/doctor"
	"codea/tui/internal/skill"
	"codea/tui/internal/update"
)

func TestRealCandidateDoctor(t *testing.T) {
	candidateDir := os.Getenv("CODEA_CANDIDATE_DIR")
	configDir := os.Getenv("CODEA_CANDIDATE_CONFIG_DIR")
	evidenceFile := os.Getenv("CODEA_CANDIDATE_EVIDENCE")
	if candidateDir == "" || configDir == "" {
		t.Skip("real candidate doctor environment not configured")
	}

	factory := doctor.NewCandidateRuntimeFactory(doctor.CandidateRuntimeOptions{
		SkillMode:      skill.SkillModeStrict,
		StartupTimeout: 30 * time.Second,
		StopTimeout:    5 * time.Second,
	})
	checker := &doctor.UpdateChecker{
		Factory:                 factory,
		ExpectedOpenCodeVersion: "1.18.11",
		Timeout:                 30 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	candidate := update.Candidate{Version: "0.1.0", VersionDir: candidateDir, ConfigDir: configDir}
	if err := checker.Check(ctx, update.CheckPreSwitch, candidate); err != nil {
		t.Fatalf("candidate doctor: %v", err)
	}

	if evidenceFile != "" {
		if err := os.MkdirAll(filepath.Dir(evidenceFile), 0o755); err != nil {
			t.Fatal(err)
		}
		payload := map[string]any{
			"candidateVersion": "0.1.0",
			"openCodeVersion": "1.18.11",
			"phase": "pre-switch",
			"candidateDoctorPassed": true,
			"runtimeIsolation": true,
		}
		data, _ := json.MarshalIndent(payload, "", "  ")
		data = append(data, '\n')
		if err := os.WriteFile(evidenceFile, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
