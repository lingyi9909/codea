package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"codea/tui/internal/parity"
)

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, data, 0o600); err != nil { t.Fatal(err) }
}

func TestLoadCertificationUsesParityAndAllReleaseGates(t *testing.T) {
	dir := t.TempDir()
	parityPath := filepath.Join(dir, "parity.json")
	gatesPath := filepath.Join(dir, "gates.json")
	result := &parity.Result{Total: 10, Passed: 10, RequiredFailed: 0}
	writeJSON(t, parityPath, map[string]any{"passed": true, "result": result})
	gates := make([]parity.ReleaseGateEvidence, 0, len(parity.RequiredReleaseGateIDs()))
	for _, id := range parity.RequiredReleaseGateIDs() {
		gates = append(gates, parity.ReleaseGateEvidence{ID:id, Status:parity.ReleaseGatePass, Evidence:"evidence/"+id, SourceCommit:"abc123"})
	}
	writeJSON(t, gatesPath, gates)

	cert, err := loadCertification("abc123", parityPath, gatesPath)
	if err != nil { t.Fatal(err) }
	if cert.SourceCommit != "abc123" { t.Fatalf("source=%q", cert.SourceCommit) }
	if cert.GeneralCompletionRate != 1 { t.Fatalf("rate=%f", cert.GeneralCompletionRate) }
	if err := cert.Validate(); err != nil { t.Fatalf("validate: %v", err) }
}

func TestLoadCertificationRejectsParityThatDidNotPass(t *testing.T) {
	dir := t.TempDir(); parityPath := filepath.Join(dir,"parity.json"); gatesPath := filepath.Join(dir,"gates.json")
	writeJSON(t, parityPath, map[string]any{"passed": false, "result": &parity.Result{Total:1, Failed:1, RequiredFailed:1}})
	writeJSON(t, gatesPath, []parity.ReleaseGateEvidence{})
	if _, err := loadCertification("abc123", parityPath, gatesPath); err == nil { t.Fatal("expected failed parity rejection") }
}
