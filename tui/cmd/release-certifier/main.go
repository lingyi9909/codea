package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"codea/tui/internal/parity"
)

type parityArtifact struct {
	Passed bool           `json:"passed"`
	Result *parity.Result `json:"result"`
}

type certificationArtifact struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Timestamp     string               `json:"timestamp"`
	Passed        bool                 `json:"passed"`
	Certification parity.Certification `json:"certification"`
}

func loadCertification(sourceCommit, parityPath, gatesPath string) (parity.Certification, error) {
	if sourceCommit == "" {
		return parity.Certification{}, fmt.Errorf("source commit is required")
	}
	var p parityArtifact
	data, err := os.ReadFile(parityPath)
	if err != nil {
		return parity.Certification{}, fmt.Errorf("read parity evidence: %w", err)
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return parity.Certification{}, fmt.Errorf("parse parity evidence: %w", err)
	}
	if !p.Passed || p.Result == nil {
		return parity.Certification{}, fmt.Errorf("parity evidence is not passed")
	}
	if p.Result.Total != 12 || p.Result.Passed != 12 || p.Result.RequiredFailed != 0 {
		return parity.Certification{}, fmt.Errorf(
			"release parity must be exact 12/12 required scenarios with zero required failures, got passed=%d total=%d requiredFailed=%d",
			p.Result.Passed, p.Result.Total, p.Result.RequiredFailed,
		)
	}

	var gates []parity.ReleaseGateEvidence
	data, err = os.ReadFile(gatesPath)
	if err != nil {
		return parity.Certification{}, fmt.Errorf("read release gates: %w", err)
	}
	if err := json.Unmarshal(data, &gates); err != nil {
		return parity.Certification{}, fmt.Errorf("parse release gates: %w", err)
	}

	rate := float64(p.Result.Passed) / float64(p.Result.Total)
	cert := parity.Certification{
		SourceCommit:          sourceCommit,
		Gates:                 gates,
		Parity:                p.Result,
		GeneralCompletionRate: rate,
	}
	return cert, nil
}

func writeArtifact(path string, cert parity.Certification) error {
	out := certificationArtifact{
		SchemaVersion: 1,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Passed:        true,
		Certification: cert,
	}
	data, err := json.MarshalIndent(out, "", "  ")
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
	source := os.Getenv("CODEA_SOURCE_COMMIT")
	parityPath := os.Getenv("CODEA_PARITY_EVIDENCE")
	if parityPath == "" {
		parityPath = filepath.FromSlash("tests/parity/evidence/release-parity.json")
	}
	gatesPath := os.Getenv("CODEA_RELEASE_GATES")
	if gatesPath == "" {
		gatesPath = filepath.FromSlash("../tests/evidence/release-gates.json")
	}
	outPath := os.Getenv("CODEA_CERTIFICATION_EVIDENCE")
	if outPath == "" {
		outPath = filepath.FromSlash("../tests/evidence/release-certification.json")
	}

	cert, err := loadCertification(source, parityPath, gatesPath)
	if err != nil {
		return err
	}
	if err := cert.Validate(); err != nil {
		return fmt.Errorf("release certification failed: %w", err)
	}
	if err := writeArtifact(outPath, cert); err != nil {
		return fmt.Errorf("write certification evidence: %w", err)
	}
	fmt.Printf("release certification passed: %d gates, parity %.1f%%\n", len(cert.Gates), cert.GeneralCompletionRate*100)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Certification error:", err)
		os.Exit(1)
	}
}
