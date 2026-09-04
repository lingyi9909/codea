package command

import "testing"

func TestCheckpointCommandsAreProtectedBuiltins(t *testing.T) {
	reg := NewRegistry()
	for _, def := range BuiltinCommands() {
		if err := reg.Register(def); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"checkpoint", "checkpoints", "restore"} {
		if _, err := reg.Parse("/" + name); err != nil {
			t.Fatalf("builtin /%s missing: %v", name, err)
		}
		for _, source := range []Source{SourceEnterprise, SourceProject} {
			if err := reg.Register(Definition{Name: name, Source: source, Action: ActionPrompt, Template: "override"}); err == nil {
				t.Fatalf("%s command overrode protected /%s", source, name)
			}
		}
	}
}

func TestRestoreRoutesAsLocalActionWithArgument(t *testing.T) {
	reg := NewRegistry()
	for _, def := range BuiltinCommands() {
		if err := reg.Register(def); err != nil {
			t.Fatal(err)
		}
	}
	out, err := reg.Execute("/restore cp-000123")
	if err != nil {
		t.Fatal(err)
	}
	if out.Kind != OutcomeAction || out.Action != ActionRestore || out.Arguments != "cp-000123" {
		t.Fatalf("unexpected restore route: %+v", out)
	}
	if out.Prompt != "" {
		t.Fatalf("restore must never route through a model: %+v", out)
	}
}
