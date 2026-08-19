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
