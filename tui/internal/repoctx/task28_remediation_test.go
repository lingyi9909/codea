package repoctx

import (
	"context"
	"strings"
	"testing"
)

func TestTask28RemediationJavaRelationResolutionUsesPackageAndImports(t *testing.T) {
	root := t.TempDir()
	writeRepoFile(t, root, "src/app/Caller.java", `package app;
import lib.good.Service;
class Caller {
    private final Service service;
    Caller(Service service) { this.service = service; }
    void run() { service.work(); }
}`)
	writeRepoFile(t, root, "src/lib/good/Service.java", `package lib.good; class Service { void work() {} }`)
	writeRepoFile(t, root, "src/lib/bad/Service.java", `package lib.bad; class Service { void work() {} }`)

	idx, err := NewIndexer(root).Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	from := symbolByPathOwnerName(idx.Symbols, "src/app/Caller.java", "Caller", "run")
	good := symbolByPathOwnerName(idx.Symbols, "src/lib/good/Service.java", "Service", "work")
	bad := symbolByPathOwnerName(idx.Symbols, "src/lib/bad/Service.java", "Service", "work")
	if from.ID == "" || good.ID == "" || bad.ID == "" {
		t.Fatalf("missing symbols: %+v", idx.Symbols)
	}
	if !hasRelation(idx.Relations, from.ID, good.ID, RelationCalls) {
		t.Fatalf("imported Java target not resolved: relations=%+v unresolved=%v", idx.Relations, idx.Unresolved)
	}
	if hasRelation(idx.Relations, from.ID, bad.ID, RelationCalls) {
		t.Fatalf("wrong-package Java target resolved: %+v", idx.Relations)
	}
}

func TestTask28RemediationJavaUnimportedForeignTypeStaysUnresolved(t *testing.T) {
	root := t.TempDir()
	writeRepoFile(t, root, "src/app/Caller.java", `package app; class Caller { Service service; void run() { service.work(); } }`)
	writeRepoFile(t, root, "src/lib/Service.java", `package lib.foreign; class Service { void work() {} }`)

	idx, err := NewIndexer(root).Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	from := symbolByPathOwnerName(idx.Symbols, "src/app/Caller.java", "Caller", "run")
	foreign := symbolByPathOwnerName(idx.Symbols, "src/lib/Service.java", "Service", "work")
	if hasRelation(idx.Relations, from.ID, foreign.ID, RelationCalls) {
		t.Fatalf("unimported foreign Java type promoted: %+v", idx.Relations)
	}
	if !strings.Contains(strings.Join(idx.Unresolved, "\n"), "unresolved") {
		t.Fatalf("expected unresolved evidence, got %v", idx.Unresolved)
	}
}

func TestTask28RemediationGoRelationResolutionUsesImportAliasAndPackage(t *testing.T) {
	root := t.TempDir()
	writeRepoFile(t, root, "cmd/runner.go", `package cmd
import svc "example.com/project/internal/good"
type Runner struct { service *svc.Service }
func (r *Runner) Run() { r.service.Work() }
`)
	writeRepoFile(t, root, "internal/good/service.go", `package good
type Service struct{}
func (s *Service) Work() {}
`)
	writeRepoFile(t, root, "internal/bad/service.go", `package bad
type Service struct{}
func (s *Service) Work() {}
`)

	idx, err := NewIndexer(root).Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	from := symbolByPathOwnerName(idx.Symbols, "cmd/runner.go", "Runner", "Run")
	good := symbolByPathOwnerName(idx.Symbols, "internal/good/service.go", "Service", "Work")
	bad := symbolByPathOwnerName(idx.Symbols, "internal/bad/service.go", "Service", "Work")
	if !hasRelation(idx.Relations, from.ID, good.ID, RelationCalls) {
		t.Fatalf("aliased Go import target not resolved: relations=%+v unresolved=%v", idx.Relations, idx.Unresolved)
	}
	if hasRelation(idx.Relations, from.ID, bad.ID, RelationCalls) {
		t.Fatalf("wrong-package Go target resolved: %+v", idx.Relations)
	}
}

func TestTask28RemediationGoUnimportedForeignTypeStaysUnresolved(t *testing.T) {
	root := t.TempDir()
	writeRepoFile(t, root, "cmd/runner.go", `package cmd
type Runner struct { service *Service }
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
	foreign := symbolByPathOwnerName(idx.Symbols, "internal/good/service.go", "Service", "Work")
	if hasRelation(idx.Relations, from.ID, foreign.ID, RelationCalls) {
		t.Fatalf("unimported foreign Go type promoted: %+v", idx.Relations)
	}
	if !strings.Contains(strings.Join(idx.Unresolved, "\n"), "unresolved") {
		t.Fatalf("expected unresolved evidence, got %v", idx.Unresolved)
	}
}

func TestTask28RemediationJavaGenericDeclaredTypeUsesOuterType(t *testing.T) {
	f := ExtractJava("GenericHolder.java", []byte(`class GenericHolder {
    private final List<OrderService> services;
    GenericHolder(List<OrderService> services) { this.services = services; }
    void run() { services.size(); }
}`))
	ctor := findSymbol(t, f.Symbols, "GenericHolder", SymbolConstructor)
	if ctor.Signature != "GenericHolder(List)" {
		t.Fatalf("generic constructor signature=%q", ctor.Signature)
	}
	run := findSymbol(t, f.Symbols, "run", SymbolMethod)
	if !hasCandidate(f, RelationCalls, run.ID, "List", "size") {
		t.Fatalf("generic receiver declared type misparsed: %+v", f.candidates)
	}
	if hasCandidate(f, RelationCalls, run.ID, "OrderService", "size") {
		t.Fatalf("generic argument promoted to receiver declared type: %+v", f.candidates)
	}
}

func TestTask28RemediationConstructorInjectsRequireSpringDIEvidence(t *testing.T) {
	f := ExtractJava("Constructors.java", []byte(`
@Service class Managed {
    Managed(Dependency dependency) {}
}
class Plain {
    Plain(Dependency dependency) {}
}
class Explicit {
    @Autowired Explicit(Dependency dependency) {}
}
class Dependency {}
`))
	managed := findSymbol(t, f.Symbols, "Managed", SymbolClass)
	plain := findSymbol(t, f.Symbols, "Plain", SymbolClass)
	explicit := findSymbol(t, f.Symbols, "Explicit", SymbolClass)
	if !hasCandidate(f, RelationInjects, managed.ID, "Dependency", "") {
		t.Fatalf("single-constructor Spring component lost DI evidence: %+v", f.candidates)
	}
	if hasCandidate(f, RelationInjects, plain.ID, "Dependency", "") {
		t.Fatalf("plain POJO constructor promoted to injects: %+v", f.candidates)
	}
	if !hasCandidate(f, RelationInjects, explicit.ID, "Dependency", "") {
		t.Fatalf("explicit @Autowired constructor lost DI evidence: %+v", f.candidates)
	}
}

func TestTask28RemediationWalkerExcludesMavenWrapperOnly(t *testing.T) {
	root := t.TempDir()
	writeWalkerFile(t, root, ".mvn/wrapper/maven-wrapper.properties", "wrapperUrl=https://example.invalid")
	writeWalkerFile(t, root, ".mvn/extensions.xml", "<extensions/>")
	writeWalkerFile(t, root, "src/main.go", "package main")

	files, err := NewWalker(root, WalkerOptions{}).Walk()
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	for _, f := range files {
		paths[f.Path] = true
	}
	if paths[".mvn/wrapper/maven-wrapper.properties"] {
		t.Fatalf(".mvn/wrapper should be excluded: %+v", files)
	}
	if !paths[".mvn/extensions.xml"] || !paths["src/main.go"] {
		t.Fatalf("narrow .mvn exclusion removed valid files: %+v", files)
	}
}

func symbolByPathOwnerName(symbols []Symbol, path, owner, name string) Symbol {
	for _, s := range symbols {
		if s.Path == path && s.Owner == owner && s.Name == name {
			return s
		}
	}
	return Symbol{}
}
