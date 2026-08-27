package agent

import (
	"strings"
	"testing"
)

func TestTask24ParseManifestRetainsAskTools(t *testing.T) {
	m, err := ParseManifest([]byte(`name: debug
displayName: Debug Agent
tools:
  read: allow
  grep: allow
  glob: allow
  write: ask
  edit: ask
  bash: ask
`))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	want := []string{"bash", "edit", "write"}
	if len(m.AskTools) != len(want) {
		t.Fatalf("askTools = %v, want %v", m.AskTools, want)
	}
	for i := range want {
		if m.AskTools[i] != want[i] {
			t.Errorf("askTools[%d] = %q, want %q", i, m.AskTools[i], want[i])
		}
	}
}

func TestTask24RenderAskToolsBehindFailClosedWildcard(t *testing.T) {
	out := string(Render(Manifest{
		Name:        "debug",
		DisplayName: "Debug Agent",
		AllowTools:  []string{"glob", "grep", "read"},
		AskTools:    []string{"bash", "edit", "write"},
	}, "Debug prompt.\n"))

	for _, want := range []string{
		"  \"*\": deny\n",
		"  glob: allow\n",
		"  grep: allow\n",
		"  read: allow\n",
		"  bash: ask\n",
		"  edit: ask\n",
		"  write: ask\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered Debug Agent missing %q in:\n%s", want, out)
		}
	}
}
