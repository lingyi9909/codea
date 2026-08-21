package apidocumentation

import (
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
    for _, n := range needles {
        if !strings.Contains(s, n) { t.Errorf("missing %q", n) }
    }
}

func TestAPIDocumentationManifestUsesOnlyBoundedTools(t *testing.T) {
    m := read(t, "distribution/agents/api-documentation/manifest.yaml")
    requireAll(t, m,
        "name: api-documentation",
        "mode: enterprise-controlled",
        "- api-documentation",
        "extract_api_spec: allow",
        "validate_api_example: allow",
        "write_document: allow",
        "dify-query: allow",
        "write: deny", "edit: deny", "bash: deny",
        "noFabrication: true",
        "Not determined from code",
    )
    for _, forbidden := range []string{"collect_review_context: allow", "write_test_file: allow", "run_project_test: allow"} {
        if strings.Contains(m, forbidden) { t.Errorf("api-documentation must not allow %q", forbidden) }
    }
}

func TestAPIDocumentationWorkflowIsDeterministicFirst(t *testing.T) {
    a := read(t, "distribution/agents/api-documentation/agent.md")
    requireAll(t, a,
        "extract_api_spec",
        "validate_api_example",
        "write_document",
        "deterministic extraction is authoritative",
        "Not determined from code",
        "DECLARED", "REFERENCED", "INFERRED",
        "Dify", "optional business context",
        "must not override code evidence",
        "never fabricate",
    )
    if strings.Index(a, "extract_api_spec") > strings.Index(a, "Dify") {
        t.Error("deterministic extraction must happen before optional Dify enrichment")
    }
}

func TestAPIDocumentationTemplateCarriesTraceabilityAndUncertainty(t *testing.T) {
    tpl := read(t, "distribution/agents/api-documentation/output-template.md")
    requireAll(t, tpl,
        "# API Documentation",
        "Source",
        "HTTP Method",
        "Path",
        "Request",
        "Response",
        "Error Codes",
        "Evidence",
        "Confidence",
        "REFERENCED",
        "Not determined from code",
    )
}

func TestAPIDocumentationRequiredSkillExists(t *testing.T) {
    s := read(t, "distribution/skills/api-documentation/SKILL.md")
    requireAll(t, s, "name: api-documentation", "deterministic", "no fabrication")
}
