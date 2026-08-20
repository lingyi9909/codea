package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseManifestExtractsNameAndDenyTools(t *testing.T) {
	body := `name: code-reviewer
version: 1.0.0
displayName: Code Reviewer
mode: enterprise-controlled
requiredSkills:
  - code-review
optionalSkills:
  - java-review
  - security-review
tools:
  read: allow
  grep: allow
  glob: allow
  collect_review_context: allow
  dify-query: allow
  write: deny
  edit: deny
  bash: deny
constraints:
  maxCallChainDepth: 3
  minimumFindingConfidence: 0.80
  readOnly: true
`
	m, err := ParseManifest([]byte(body))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if m.Name != "code-reviewer" {
		t.Errorf("name = %q, want code-reviewer", m.Name)
	}
	if m.DisplayName != "Code Reviewer" {
		t.Errorf("displayName = %q, want Code Reviewer", m.DisplayName)
	}
	want := []string{"bash", "edit", "write"}
	if len(m.DenyTools) != len(want) {
		t.Fatalf("denyTools = %v, want %v", m.DenyTools, want)
	}
	for i := range want {
		if m.DenyTools[i] != want[i] {
			t.Errorf("denyTools[%d] = %q, want %q", i, m.DenyTools[i], want[i])
		}
	}
}

func TestParseManifestRequiresName(t *testing.T) {
	if _, err := ParseManifest([]byte("tools:\n  write: deny\n")); err == nil {
		t.Fatal("expected error for manifest without name")
	}
}

func TestRenderProducesOpenCodeFrontmatter(t *testing.T) {
	m := Manifest{Name: "code-reviewer", DisplayName: "Code Reviewer", DenyTools: []string{"bash", "edit", "write"}}
	out := string(Render(m, "You are the reviewer.\n"))

	for _, want := range []string{
		"---\n",
		"description: Code Reviewer\n",
		"mode: all\n",
		"permission:\n",
		"  write: deny\n",
		"  edit: deny\n",
		"  bash: deny\n",
		"---\n",
		"You are the reviewer.\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered agent missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderNoDenyToolsOmitsPermissionBlock(t *testing.T) {
	m := Manifest{Name: "api-documentation", DisplayName: "API Docs"}
	out := string(Render(m, "prompt"))
	if !strings.Contains(out, "permission:\n") {
		t.Errorf("expected an (empty) permission block to be emitted, got:\n%s", out)
	}
	if strings.Contains(out, "write: deny") {
		t.Errorf("must not emit denies for an agent with none, got:\n%s", out)
	}
}

func TestMaterializeWritesAgentMarkdownAndClearsStale(t *testing.T) {
	root := t.TempDir()
	agents := filepath.Join(root, "agents")
	target := filepath.Join(root, "target")

	if err := os.MkdirAll(filepath.Join(agents, "code-reviewer"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agents, "code-reviewer", "manifest.yaml"), []byte("name: code-reviewer\ndisplayName: Code Reviewer\ntools:\n  write: deny\n  edit: deny\n  bash: deny\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agents, "code-reviewer", "agent.md"), []byte("Reviewer prompt.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A directory without manifest.yaml is ignored.
	if err := os.MkdirAll(filepath.Join(agents, "not-an-agent"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Materialize(agents, target); err != nil {
		t.Fatalf("Materialize: %v", err)
	}

	out, err := os.ReadFile(filepath.Join(target, "code-reviewer.md"))
	if err != nil {
		t.Fatalf("expected code-reviewer.md: %v", err)
	}
	s := string(out)
	for _, want := range []string{"mode: all\n", "write: deny\n", "Reviewer prompt.\n"} {
		if !strings.Contains(s, want) {
			t.Errorf("code-reviewer.md missing %q:\n%s", want, s)
		}
	}

	// A stale file from a previous run is removed on re-materialize.
	stale := filepath.Join(target, "stale-agent.md")
	if err := os.WriteFile(stale, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Materialize(agents, target); err != nil {
		t.Fatalf("re-Materialize: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale agent was not removed")
	}
}

func TestMaterializeMissingRootIsNoOp(t *testing.T) {
	if err := Materialize(filepath.Join(t.TempDir(), "does-not-exist"), filepath.Join(t.TempDir(), "target")); err != nil {
		t.Fatalf("Materialize on missing root should be a no-op, got %v", err)
	}
}

// TestMaterializeToConfigDir is the smoke-harness entry point: it materializes
// the real distribution/agents into an OpenCode config dir supplied via env so
// the real-agent smoke can start OpenCode against the actual generated files.
// Without the env vars it is a no-op (skipped) so `go test ./...` stays green.
func TestMaterializeToConfigDir(t *testing.T) {
	root := os.Getenv("CODEA_AGENT_SYNC_ROOT")
	dir := os.Getenv("CODEA_AGENT_SYNC_DIR")
	if root == "" || dir == "" {
		t.Skip("CODEA_AGENT_SYNC_ROOT/DIR not set")
	}
	if err := Materialize(root, dir); err != nil {
		t.Fatalf("Materialize(%s, %s): %v", root, dir, err)
	}
}
