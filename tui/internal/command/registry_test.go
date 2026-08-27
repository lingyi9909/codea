package command

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuiltinsAreRegisteredAndFilterable(t *testing.T) {
	reg := NewRegistry()
	for _, def := range BuiltinCommands() {
		if err := reg.Register(def); err != nil {
			t.Fatalf("register builtin %q: %v", def.Name, err)
		}
	}

	got := reg.Filter("/se")
	var names []string
	for _, def := range got {
		names = append(names, def.Name)
	}
	want := []string{"sessions"}
	if !reflect.DeepEqual(gotNames(names), want) {
		t.Fatalf("Filter(/se) = %v, want %v", names, want)
	}

	all := reg.Commands()
	if len(all) != 15 {
		t.Fatalf("builtins = %d, want 15 through Task 25", len(all))
	}
	seen := map[string]bool{}
	for _, def := range all {
		seen[def.Name] = true
		if def.Availability != AvailabilityAvailable {
			t.Fatalf("/%s availability = %q, want %q", def.Name, def.Availability, AvailabilityAvailable)
		}
	}
	for _, name := range []string{"help", "clear", "status", "sessions", "skills", "agents", "cancel", "doctor", "model", "compact", "view", "review", "test", "api-doc", "debug"} {
		if !seen[name] {
			t.Fatalf("missing builtin /%s", name)
		}
	}
}

func gotNames(in []string) []string { return in }

func TestDefinitionOwnsCapabilityAndAvailabilityMetadata(t *testing.T) {
	reg := NewRegistry()
	want := Definition{
		Name:               "cap-aware",
		Source:             SourceBuiltin,
		Action:             ActionStatus,
		RequiredCapability: "compaction",
		Agent:              "general",
		Availability:       AvailabilityUnavailable,
	}
	if err := reg.Register(want); err != nil {
		t.Fatal(err)
	}
	got := reg.Commands()[0]
	if got.RequiredCapability != want.RequiredCapability || got.Agent != want.Agent || got.Availability != want.Availability {
		t.Fatalf("registered metadata = %#v, want capability=%q agent=%q availability=%q", got, want.RequiredCapability, want.Agent, want.Availability)
	}
}

func TestParsePreservesCommandArguments(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(Definition{Name: "review", Source: SourceBuiltin, Action: ActionReview}); err != nil {
		t.Fatal(err)
	}

	inv, err := reg.Parse("/review OrderService  --deep ")
	if err != nil {
		t.Fatal(err)
	}
	if inv.Definition.Name != "review" {
		t.Fatalf("name = %q, want review", inv.Definition.Name)
	}
	if inv.Arguments != "OrderService  --deep " {
		t.Fatalf("arguments = %q", inv.Arguments)
	}
}

func TestUnknownCommandIsDeterministic(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Parse("/does-not-exist")
	var cmdErr *Error
	if !errors.As(err, &cmdErr) {
		t.Fatalf("error = %T %v, want *Error", err, err)
	}
	if cmdErr.Code != CodeNotFound {
		t.Fatalf("code = %q, want %q", cmdErr.Code, CodeNotFound)
	}
}

func TestCollisionFailsClosedWithoutOverride(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(Definition{Name: "doctor", Aliases: []string{"diag"}, Source: SourceBuiltin, Action: ActionDoctor}); err != nil {
		t.Fatal(err)
	}

	err := reg.Register(Definition{Name: "project-doctor", Aliases: []string{"doctor"}, Source: SourceProject, Action: ActionPrompt})
	var cmdErr *Error
	if !errors.As(err, &cmdErr) || cmdErr.Code != CodeConflict {
		t.Fatalf("collision error = %T %v, want %s", err, err, CodeConflict)
	}

	inv, err := reg.Parse("/doctor")
	if err != nil {
		t.Fatal(err)
	}
	if inv.Definition.Source != SourceBuiltin {
		t.Fatalf("doctor source = %q, want builtin", inv.Definition.Source)
	}
}

func TestFutureControlledBuiltinsAreReservedBeforeOwningTaskRegistersThem(t *testing.T) {
	for _, def := range []Definition{
		{Name: "review", Source: SourceEnterprise, Action: ActionPrompt},
		{Name: "project-model", Aliases: []string{"model"}, Source: SourceProject, Action: ActionPrompt},
		{Name: "project-view", Aliases: []string{"view"}, Source: SourceProject, Action: ActionPrompt},
	} {
		reg := NewRegistry()
		err := reg.Register(def)
		var cmdErr *Error
		if !errors.As(err, &cmdErr) || cmdErr.Code != CodeConflict {
			t.Fatalf("Register(%#v) error = %T %v, want %s", def, err, err, CodeConflict)
		}
	}

	reg := NewRegistry()
	if err := reg.Register(Definition{Name: "review", Source: SourceBuiltin, Action: ActionReview}); err != nil {
		t.Fatalf("controlled built-in owner should be allowed to register /review: %v", err)
	}
	if err := reg.Register(Definition{Name: "view", Source: SourceBuiltin, Action: ActionView}); err != nil {
		t.Fatalf("controlled built-in owner should be allowed to register /view: %v", err)
	}
}

func TestWorkspaceLoaderLoadsEnterpriseThenProjectAndRendersArguments(t *testing.T) {
	root := t.TempDir()
	enterprise := filepath.Join(root, "distribution", "commands")
	project := filepath.Join(root, "project", ".codea", "commands")
	if err := os.MkdirAll(enterprise, 0o755); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(project, 0o755); err != nil { t.Fatal(err) }

	enterpriseDoc := "---\nname: check-order\ndescription: Check the order module\nagent: code-reviewer\n---\n\nReview order:\n$ARGUMENTS\n"
	projectDoc := "---\nname: explain\ndescription: Explain selected code\n---\n\nExplain exactly:\n$ARGUMENTS\n"
	if err := os.WriteFile(filepath.Join(enterprise, "check-order.md"), []byte(enterpriseDoc), 0o644); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(project, "explain.md"), []byte(projectDoc), 0o644); err != nil { t.Fatal(err) }

	reg, err := LoadWorkspaceRegistry(enterprise, project)
	if err != nil {
		t.Fatal(err)
	}

	out, err := reg.Execute("/check-order OrderService  --changed-only")
	if err != nil {
		t.Fatal(err)
	}
	if out.Kind != OutcomePrompt || out.Agent != "code-reviewer" {
		t.Fatalf("outcome = %#v", out)
	}
	if out.Prompt != "Review order:\nOrderService  --changed-only\n" {
		t.Fatalf("prompt = %q", out.Prompt)
	}

	inv, err := reg.Parse("/explain Foo.java")
	if err != nil { t.Fatal(err) }
	if inv.Definition.Source != SourceProject {
		t.Fatalf("source = %q, want project", inv.Definition.Source)
	}
}

func TestWorkspaceLoaderRejectsProjectOverrideOfBuiltin(t *testing.T) {
	project := filepath.Join(t.TempDir(), ".codea", "commands")
	if err := os.MkdirAll(project, 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(project, "doctor.md"), []byte("---\nname: doctor\ndescription: replace doctor\n---\n\nnope\n"), 0o644); err != nil { t.Fatal(err) }

	_, err := LoadWorkspaceRegistry("", project)
	var cmdErr *Error
	if !errors.As(err, &cmdErr) || cmdErr.Code != CodeConflict {
		t.Fatalf("error = %T %v, want %s", err, err, CodeConflict)
	}
}

func TestWorkspaceLoaderRejectsProjectClaimOnFutureControlledBuiltin(t *testing.T) {
	project := filepath.Join(t.TempDir(), ".codea", "commands")
	if err := os.MkdirAll(project, 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(project, "review.md"), []byte("---\nname: review\ndescription: steal future review\n---\n\nnope\n"), 0o644); err != nil { t.Fatal(err) }

	_, err := LoadWorkspaceRegistry("", project)
	var cmdErr *Error
	if !errors.As(err, &cmdErr) || cmdErr.Code != CodeConflict || cmdErr.Command != "review" {
		t.Fatalf("error = %T %v, want %s for /review", err, err, CodeConflict)
	}
}
