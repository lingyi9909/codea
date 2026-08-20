package codereview

import (
    "encoding/json"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "testing"
)

func repoRoot(t *testing.T) string {
    t.Helper()
    _, file, _, ok := runtime.Caller(0)
    if !ok { t.Fatal("runtime.Caller failed") }
    return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}
func read(t *testing.T, rel string) string {
    t.Helper()
    b, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
    if err != nil { t.Fatalf("read %s: %v", rel, err) }
    return string(b)
}
func requireAll(t *testing.T, s string, needles ...string) {
    t.Helper()
    for _, n := range needles { if !strings.Contains(s, n) { t.Errorf("missing %q", n) } }
}

func TestReviewerManifestIsReadOnlyAndToolBounded(t *testing.T) {
    m := read(t, "distribution/agents/code-reviewer/manifest.yaml")
    requireAll(t, m, "name: code-reviewer", "mode: enterprise-controlled", "- code-review", "- java-review", "- security-review", "collect_review_context: allow", "dify-query: allow", "write: deny", "edit: deny", "bash: deny")
}
func TestReviewerWorkflowIsScopeFirstAndExpandsBoundedCallChains(t *testing.T) {
    a := read(t, "distribution/agents/code-reviewer/agent.md")
    requireAll(t, a, "collect_review_context", "staged", "unstaged", "base-branch", "commit", "range", "file-path", "maximum call-chain depth: 3", "Controller", "Service", "Repository", "Mapper", "DTO", "Domain", "read the downstream implementation before concluding")
    if strings.Index(a, "collect_review_context") > strings.Index(a, "grep") { t.Error("review context collection must precede repository expansion") }
}
func TestReviewerFindingQualityAndCleanDiffRules(t *testing.T) {
    a := read(t, "distribution/agents/code-reviewer/agent.md")
    requireAll(t, a, "introducedByChange", "Critical", "Major", "Minor", "Suggestion", "confidence >= 0.80", "uncertainObservations", "findings=[]", "Do not fabricate")
    requireAll(t, a, "file", "lineRange", "severity", "title", "description", "evidence", "recommendation")
}
func TestReviewerDifyIsOptionalKnowledgeOnly(t *testing.T) {
    a := read(t, "distribution/agents/code-reviewer/agent.md")
    requireAll(t, a, "businessKnowledgeUnavailable", "Dify", "must not be used as code evidence", "review must continue")
}
func TestReviewerOutputSchemaIsJSONSchema(t *testing.T) {
    raw := read(t, "distribution/agents/code-reviewer/output-schema.json")
    var s map[string]any
    if err := json.Unmarshal([]byte(raw), &s); err != nil { t.Fatal(err) }
    if s["$schema"] == nil || s["type"] != "object" { t.Fatal("not a JSON object schema") }
    p := s["properties"].(map[string]any)
    for _, k := range []string{"summary", "scope", "findings", "observations", "uncertainObservations", "reviewStats", "businessKnowledgeUnavailable"} { if p[k] == nil { t.Errorf("missing schema property %s", k) } }
    f := p["findings"].(map[string]any)["items"].(map[string]any)
    req := f["required"].([]any)
    joined := ""
    for _, v := range req { joined += v.(string) + " " }
    for _, k := range []string{"file", "lineRange", "severity", "title", "description", "evidence", "introducedByChange", "confidence", "recommendation"} { if !strings.Contains(joined, k) { t.Errorf("finding required missing %s", k) } }
}

// TestReviewerConfidenceThresholdEnforcedInSchema is the negative contract test
// for the confidence floor: a finding with confidence 0.79 must fail the schema
// minimum, and 0.80 must pass. It reads the declared minimum directly (the Go
// module has no JSON Schema validator) and asserts the boundary so the schema
// cannot silently regress back to a 0..1 range.
func TestReviewerConfidenceThresholdEnforcedInSchema(t *testing.T) {
    raw := read(t, "distribution/agents/code-reviewer/output-schema.json")
    var s map[string]any
    if err := json.Unmarshal([]byte(raw), &s); err != nil { t.Fatal(err) }
    conf := s["properties"].(map[string]any)["findings"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["confidence"].(map[string]any)
    min, ok := conf["minimum"].(float64)
    if !ok { t.Fatal("confidence schema missing numeric minimum") }
    if min != 0.8 { t.Fatalf("confidence minimum = %v, want 0.8", min) }
    if 0.79 >= min { t.Errorf("confidence 0.79 must be below the minimum %v (0.79 should FAIL)", min) }
    if 0.80 < min { t.Errorf("confidence 0.80 must meet the minimum %v (0.80 should PASS)", min) }
}
