package command

import "testing"

func TestTask24ProfessionalBuiltinsRouteDirectlyToExactAgents(t *testing.T) {
	want := map[string]string{
		"review":  "code-reviewer",
		"test":    "unit-test-generator",
		"api-doc": "api-documentation",
		"debug":   "debug",
	}

	defs := BuiltinCommands()
	seen := make(map[string]Definition, len(defs))
	for _, def := range defs {
		seen[def.Name] = def
	}

	for name, agent := range want {
		def, ok := seen[name]
		if !ok {
			t.Errorf("/%s is not registered as a built-in professional command", name)
			continue
		}
		if def.Action != ActionPrompt {
			t.Errorf("/%s action = %q, want %q", name, def.Action, ActionPrompt)
		}
		if def.Agent != agent {
			t.Errorf("/%s agent = %q, want %q", name, def.Agent, agent)
		}
		if def.Agent == "general" {
			t.Errorf("/%s must never route through general", name)
		}
		if def.Template != "$ARGUMENTS" {
			t.Errorf("/%s template = %q, want exact argument forwarding", name, def.Template)
		}
	}
}

func TestTask24ProfessionalCommandArgumentsRemainIntact(t *testing.T) {
	reg := NewRegistry()
	for _, def := range BuiltinCommands() {
		if err := reg.Register(def); err != nil {
			t.Fatalf("register %s: %v", def.Name, err)
		}
	}

	out, err := reg.Execute("/review OrderService  --changed-only")
	if err != nil {
		t.Fatalf("execute /review: %v", err)
	}
	if out.Kind != OutcomePrompt {
		t.Fatalf("kind = %q, want %q", out.Kind, OutcomePrompt)
	}
	if out.Agent != "code-reviewer" {
		t.Fatalf("agent = %q, want code-reviewer", out.Agent)
	}
	if out.Prompt != "OrderService  --changed-only" {
		t.Fatalf("prompt = %q, want arguments preserved", out.Prompt)
	}
}
