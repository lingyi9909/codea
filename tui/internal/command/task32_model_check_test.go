package command

import "testing"

func TestTask32ModelCheckBuiltinIsProtectedLocalAction(t *testing.T) {
	reg := NewRegistry()
	for _, def := range BuiltinCommands() {
		if err := reg.Register(def); err != nil {
			t.Fatalf("register builtin %q: %v", def.Name, err)
		}
	}
	out, err := reg.Execute("/model-check")
	if err != nil {
		t.Fatalf("execute /model-check: %v", err)
	}
	if out.Kind != OutcomeAction || out.Action != ActionModelCheck {
		t.Fatalf("/model-check outcome = %#v, want local %q action", out, ActionModelCheck)
	}
	if err := reg.Register(Definition{Name: "model-check", Source: SourceEnterprise, Action: ActionPrompt}); err == nil {
		t.Fatal("enterprise /model-check override must be rejected")
	}
	if err := reg.Register(Definition{Name: "model-check", Source: SourceProject, Action: ActionPrompt}); err == nil {
		t.Fatal("project /model-check override must be rejected")
	}
}
