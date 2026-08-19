package unittest

import (
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "testing"
)

func repoRoot(t *testing.T) string {
    t.Helper()
    _, f, _, ok := runtime.Caller(0)
    if !ok { t.Fatal("runtime.Caller failed") }
    return filepath.Clean(filepath.Join(filepath.Dir(f), "..", "..", "..", ".."))
}
func read(t *testing.T, rel string) string {
    t.Helper()
    b, e := os.ReadFile(filepath.Join(repoRoot(t), rel))
    if e != nil { t.Fatalf("read %s: %v", rel, e) }
    return string(b)
}
func all(t *testing.T, s string, ns ...string) {
    t.Helper()
    for _, n := range ns { if !strings.Contains(s, n) { t.Errorf("missing %q", n) } }
}

func TestUTManifestEnforcesControlledTools(t *testing.T) {
    m := read(t, "distribution/agents/unit-test-generator/manifest.yaml")
    all(t, m, "name: unit-test-generator", "mode: enterprise-controlled", "- unit-test", "analyze_test_project: allow", "write_test_file: allow", "run_project_test: allow", "dify-query: allow", "write: deny", "edit: deny", "bash: deny", "maxRepairAttempts: 3", "neverOverwriteExisting: true")
}
func TestUTWorkflowAnalyzePlanGenerateRunRepair(t *testing.T) {
    a := read(t, "distribution/agents/unit-test-generator/agent.md")
    all(t, a, "analyze_test_project", "Test Plan", "analyze → plan → generate → write → run → classify → repair", "testFramework = unknown", "do not guess", "existing tests", "write_test_file", "run_project_test", "testMethod", "testClass", "module", "extraArgs", "must not")
}
func TestUTPlanAndOwnershipSafety(t *testing.T) {
    a := read(t, "distribution/agents/unit-test-generator/agent.md")
    all(t, a, "target class", "target method", "test file", "dependencies", "happy-path", "boundary", "invalid-input", "exception", "branch", "state-transition", "overwrite=false", "filesCreatedByCurrentRun", "production code", "existing tests", "@Disabled", "PRODUCT_CODE_DEFECT_SUSPECTED")
}
func TestUTRepairIsBounded(t *testing.T) {
    a := read(t, "distribution/agents/unit-test-generator/agent.md")
    all(t, a, "Attempt 0", "Repair 1", "Repair 2", "Repair 3", "STOP", "maximum 3 repair attempts")
}
func TestUTErrorCategories(t *testing.T) {
    e := read(t, "distribution/agents/unit-test-generator/error-categories.yaml")
    for _, c := range []string{"COMPILE_ERROR", "DEPENDENCY_ERROR", "TEST_FAILURE", "ASSERTION_FAILURE", "MOCK_CONFIGURATION", "TEST_RUNTIME_ERROR", "TIMEOUT", "INFRASTRUCTURE_ERROR"} { if !strings.Contains(e, c) { t.Errorf("missing %s", c) } }
    all(t, e, "repairable: true", "repairable: false", "PRODUCT_CODE_DEFECT_SUSPECTED")
}
func TestUTFinalReportUsesRealToolResult(t *testing.T) {
    a := read(t, "distribution/agents/unit-test-generator/agent.md")
    all(t, a, "passed", "failed", "errors", "skipped", "duration", "failureDetails", "exitCode", "category", "Never claim tests pass without run_project_test")
}
