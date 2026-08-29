package repoctx

import (
	"context"
	"strings"
	"testing"
)

func TestTask28RemediationExternalGoImportNotPromoted(t *testing.T) {
	root := t.TempDir()
	writeRepoFile(t, root, "go.mod", "module example.com/company/order\n\ngo 1.26\n")
	writeRepoFile(t, root, "cmd/runner.go", `package cmd
import svc "github.com/external/project/internal/good"
type Runner struct { service *svc.Service }
func (r *Runner) Run() { r.service.Work() }
`)
	writeRepoFile(t, root, "internal/good/service.go", `package good
type Service struct{}
func (s *Service) Work() {}
`)

	idx, err := NewIndexer(root).Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	from := symbolByPathOwnerName(idx.Symbols, "cmd/runner.go", "Runner", "Run")
	local := symbolByPathOwnerName(idx.Symbols, "internal/good/service.go", "Service", "Work")
	if from.ID == "" || local.ID == "" {
		t.Fatalf("missing symbols: %+v", idx.Symbols)
	}
	if hasRelation(idx.Relations, from.ID, local.ID, RelationCalls) {
		t.Fatalf("external-module import promoted to local relation: %+v", idx.Relations)
	}
	if !strings.Contains(strings.Join(idx.Unresolved, "\n"), "unresolved") {
		t.Fatalf("expected unresolved evidence, got %v", idx.Unresolved)
	}
}
