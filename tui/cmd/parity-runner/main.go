package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codea/tui/internal/opencode"
	"codea/tui/internal/parity"
)

type runnerConfig struct {
	BaselineURL      string
	CandidateURL     string
	BaselineUsername string
	BaselinePassword string
	CandidateUsername string
	CandidatePassword string
	EvidencePath     string
}

type parityEvidence struct {
	SchemaVersion  int            `json:"schemaVersion"`
	Timestamp      string         `json:"timestamp"`
	BaselineURL    string         `json:"baselineUrl"`
	CandidateURL   string         `json:"candidateUrl"`
	Result         *parity.Result `json:"result"`
	Passed         bool           `json:"passed"`
}

func normalizeEndpoint(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("runtime endpoint is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse runtime endpoint: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("runtime endpoint must be an absolute http(s) URL")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	if u.Path == "/" {
		u.Path = ""
	}
	return strings.TrimRight(u.String(), "/"), nil
}

func loadRunnerConfig(getenv func(string) string) (runnerConfig, error) {
	baseline, err := normalizeEndpoint(getenv("CODEA_PARITY_BASELINE_URL"))
	if err != nil {
		return runnerConfig{}, fmt.Errorf("baseline: %w", err)
	}
	candidate, err := normalizeEndpoint(getenv("CODEA_PARITY_CANDIDATE_URL"))
	if err != nil {
		return runnerConfig{}, fmt.Errorf("candidate: %w", err)
	}
	if baseline == candidate {
		return runnerConfig{}, fmt.Errorf("baseline and candidate runtime endpoints must be different")
	}
	evidence := strings.TrimSpace(getenv("CODEA_PARITY_EVIDENCE"))
	if evidence == "" {
		evidence = filepath.FromSlash("tests/parity/evidence/release-parity.json")
	}
	return runnerConfig{
		BaselineURL:       baseline,
		CandidateURL:      candidate,
		BaselineUsername:  getenv("CODEA_PARITY_BASELINE_USERNAME"),
		BaselinePassword:  getenv("CODEA_PARITY_BASELINE_PASSWORD"),
		CandidateUsername: getenv("CODEA_PARITY_CANDIDATE_USERNAME"),
		CandidatePassword: getenv("CODEA_PARITY_CANDIDATE_PASSWORD"),
		EvidencePath:      evidence,
	}, nil
}

func writeEvidence(path string, evidence parityEvidence) error {
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func run() error {
	cfg, err := loadRunnerConfig(os.Getenv)
	if err != nil {
		return err
	}

	baseline := opencode.NewOpenCodeAdapter(cfg.BaselineURL, cfg.BaselineUsername, cfg.BaselinePassword)
	candidate := opencode.NewOpenCodeAdapter(cfg.CandidateURL, cfg.CandidateUsername, cfg.CandidatePassword)
	runner := parity.Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	result := runner.RunAll(ctx, parity.V1RequiredScenarios())

	evidence := parityEvidence{
		SchemaVersion: 1,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		BaselineURL:   cfg.BaselineURL,
		CandidateURL:  cfg.CandidateURL,
		Result:        result,
		Passed:        result.RequiredFailed == 0,
	}
	if err := writeEvidence(cfg.EvidencePath, evidence); err != nil {
		return fmt.Errorf("write evidence: %w", err)
	}
	if result.RequiredFailed != 0 {
		return fmt.Errorf("release parity failed: %d required scenario(s) failed", result.RequiredFailed)
	}
	fmt.Printf("release parity passed: %d/%d scenarios\n", result.Passed, result.Total)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Parity error:", err)
		os.Exit(1)
	}
}
