package main

import (
	"strings"
	"testing"

	"codea/tui/internal/parity"
)

func TestLoadRunnerConfigRequiresTwoDistinctExplicitEndpoints(t *testing.T) {
	get := func(values map[string]string) func(string) string {
		return func(key string) string { return values[key] }
	}

	if _, err := loadRunnerConfig(get(nil)); err == nil {
		t.Fatal("missing baseline/candidate endpoints must fail")
	}
	if _, err := loadRunnerConfig(get(map[string]string{
		"CODEA_PARITY_BASELINE_URL":  "http://127.0.0.1:4101",
		"CODEA_PARITY_CANDIDATE_URL": "http://127.0.0.1:4101",
	})); err == nil {
		t.Fatal("same baseline/candidate endpoint must fail; that would be a self-comparison")
	}

	cfg, err := loadRunnerConfig(get(map[string]string{
		"CODEA_PARITY_BASELINE_URL":      "http://127.0.0.1:4101/",
		"CODEA_PARITY_CANDIDATE_URL":     "http://127.0.0.1:4102/",
		"CODEA_PARITY_BASELINE_USERNAME": "base-user",
		"CODEA_PARITY_BASELINE_PASSWORD": "base-pass",
		"CODEA_PARITY_CANDIDATE_USERNAME": "candidate-user",
		"CODEA_PARITY_CANDIDATE_PASSWORD": "candidate-pass",
		"CODEA_PARITY_EVIDENCE":          "evidence/out.json",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaselineURL != "http://127.0.0.1:4101" || cfg.CandidateURL != "http://127.0.0.1:4102" {
		t.Fatalf("unexpected normalized URLs: %+v", cfg)
	}
	if cfg.EvidencePath != "evidence/out.json" {
		t.Fatalf("evidence path=%q", cfg.EvidencePath)
	}
}

func TestLoadRunnerConfigDefaultsEvidencePathButNotRuntimeEndpoints(t *testing.T) {
	values := map[string]string{
		"CODEA_PARITY_BASELINE_URL":  "http://127.0.0.1:4201",
		"CODEA_PARITY_CANDIDATE_URL": "http://127.0.0.1:4202",
	}
	cfg, err := loadRunnerConfig(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EvidencePath == "" {
		t.Fatal("evidence path must have a deterministic default")
	}
}

func TestFormatFailureSummaryIncludesScenarioSilentLossAndReasons(t *testing.T) {
	result := &parity.Result{
		RequiredFailed: 1,
		Scenarios: []parity.ScenarioResult{
			{
				Name:       "RawEventHandling",
				Required:   true,
				Passed:     false,
				SilentLoss: true,
				Failures: []parity.Failure{
					{Reason: "candidate missing raw event"},
					{Reason: "candidate missing session.idle"},
				},
			},
		},
	}

	got := formatFailureSummary(result)
	for _, want := range []string{
		"RawEventHandling",
		"required=true",
		"silentLoss=true",
		"candidate missing raw event",
		"candidate missing session.idle",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("failure summary missing %q:\n%s", want, got)
		}
	}
}
