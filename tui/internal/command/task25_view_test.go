package command

import "testing"

func TestTask25ViewIsBuiltInLocalWorkspaceAction(t *testing.T) {
	defs := BuiltinCommands()
	var view Definition
	found := false
	for _, def := range defs {
		if def.Name == "view" {
			view = def
			found = true
			break
		}
	}
	if !found {
		t.Fatal("/view is not registered as a built-in")
	}
	if view.Action != ActionView {
		t.Fatalf("/view action = %q, want %q", view.Action, ActionView)
	}
	if view.Agent != "" {
		t.Fatalf("/view agent = %q, local view action must not route to Runtime", view.Agent)
	}
}

func TestTask25ViewArgumentsArePreservedForApplicationValidation(t *testing.T) {
	reg := NewRegistry()
	for _, def := range BuiltinCommands() {
		if err := reg.Register(def); err != nil {
			t.Fatalf("register %s: %v", def.Name, err)
		}
	}
	out, err := reg.Execute("/view verbose")
	if err != nil {
		t.Fatalf("execute /view: %v", err)
	}
	if out.Kind != OutcomeAction || out.Action != ActionView {
		t.Fatalf("outcome = %#v, want local ActionView", out)
	}
	if out.Arguments != "verbose" {
		t.Fatalf("arguments = %q, want verbose", out.Arguments)
	}
}
