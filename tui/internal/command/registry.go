// Package command implements Codea's terminal-independent command workspace.
// It deliberately has no Bubble Tea or Runtime vendor dependencies.
package command

import (
	"fmt"
	"strings"
	"unicode"
)

type Source string

const (
	SourceBuiltin    Source = "builtin"
	SourceEnterprise Source = "enterprise"
	SourceProject    Source = "project"
)

type Action string

const (
	ActionHelp     Action = "help"
	ActionClear    Action = "clear"
	ActionStatus   Action = "status"
	ActionSessions Action = "sessions"
	ActionSkills   Action = "skills"
	ActionAgents   Action = "agents"
	ActionCancel   Action = "cancel"
	ActionDoctor   Action = "doctor"
	ActionPrompt   Action = "prompt"
	// ActionReview is reserved for callers/tests that register a professional
	// action. Task 22 does not register /review as a built-in; Task 24 owns that.
	ActionReview Action = "review"
)

type Definition struct {
	Name        string
	Aliases     []string
	Description string
	Category    string
	Usage       string
	Source      Source
	Action      Action
	Agent       string
	Template    string
}

type Invocation struct {
	Definition Definition
	Raw        string
	Arguments  string
}

type OutcomeKind string

const (
	OutcomeAction OutcomeKind = "action"
	OutcomePrompt OutcomeKind = "prompt"
)

type Outcome struct {
	Kind      OutcomeKind
	Action    Action
	Command   string
	Arguments string
	Agent     string
	Prompt    string
}

type Code string

const (
	CodeConflict Code = "COMMAND_CONFLICT"
	CodeNotFound Code = "COMMAND_NOT_FOUND"
	CodeInvalid  Code = "COMMAND_INVALID"
)

type Error struct {
	Code    Code
	Command string
	Detail  string
}

func (e *Error) Error() string {
	if e.Command == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Detail)
	}
	return fmt.Sprintf("%s: /%s: %s", e.Code, e.Command, e.Detail)
}

type Registry struct {
	definitions []Definition
	lookup      map[string]int
}

func NewRegistry() *Registry {
	return &Registry{lookup: make(map[string]int)}
}

func (r *Registry) Register(def Definition) error {
	def.Name = normalizeName(def.Name)
	if !validName(def.Name) {
		return &Error{Code: CodeInvalid, Command: def.Name, Detail: "invalid command name"}
	}
	if def.Source == "" {
		def.Source = SourceBuiltin
	}

	keys := make([]string, 0, len(def.Aliases)+1)
	keys = append(keys, def.Name)
	seen := map[string]struct{}{def.Name: {}}
	for i, alias := range def.Aliases {
		alias = normalizeName(alias)
		if !validName(alias) {
			return &Error{Code: CodeInvalid, Command: def.Name, Detail: "invalid alias " + alias}
		}
		if _, duplicate := seen[alias]; duplicate {
			return &Error{Code: CodeInvalid, Command: def.Name, Detail: "duplicate alias " + alias}
		}
		seen[alias] = struct{}{}
		def.Aliases[i] = alias
		keys = append(keys, alias)
	}

	// Validate every key before mutating state. A conflict therefore cannot
	// leave a partially registered command behind.
	for _, key := range keys {
		if idx, exists := r.lookup[key]; exists {
			owner := r.definitions[idx]
			return &Error{
				Code:    CodeConflict,
				Command: key,
				Detail:  fmt.Sprintf("%s command /%s conflicts with %s command /%s", def.Source, def.Name, owner.Source, owner.Name),
			}
		}
	}

	idx := len(r.definitions)
	r.definitions = append(r.definitions, def)
	for _, key := range keys {
		r.lookup[key] = idx
	}
	return nil
}

func (r *Registry) Commands() []Definition {
	out := make([]Definition, len(r.definitions))
	copy(out, r.definitions)
	return out
}

func (r *Registry) Filter(query string) []Definition {
	token := commandToken(query)
	if token == "" {
		return r.Commands()
	}
	out := make([]Definition, 0)
	for _, def := range r.definitions {
		if strings.HasPrefix(def.Name, token) {
			out = append(out, def)
			continue
		}
		for _, alias := range def.Aliases {
			if strings.HasPrefix(alias, token) {
				out = append(out, def)
				break
			}
		}
	}
	return out
}

func (r *Registry) Parse(input string) (Invocation, error) {
	if !strings.HasPrefix(input, "/") {
		return Invocation{}, &Error{Code: CodeInvalid, Detail: "command must start with /"}
	}
	rest := input[1:]
	if rest == "" {
		return Invocation{}, &Error{Code: CodeInvalid, Detail: "command name is empty"}
	}

	end := len(rest)
	for i, ch := range rest {
		if unicode.IsSpace(ch) {
			end = i
			break
		}
	}
	name := normalizeName(rest[:end])
	idx, ok := r.lookup[name]
	if !ok {
		return Invocation{}, &Error{Code: CodeNotFound, Command: name, Detail: "unknown command"}
	}

	args := ""
	if end < len(rest) {
		args = strings.TrimLeftFunc(rest[end:], unicode.IsSpace)
	}
	return Invocation{Definition: r.definitions[idx], Raw: input, Arguments: args}, nil
}

func (r *Registry) Execute(input string) (Outcome, error) {
	inv, err := r.Parse(input)
	if err != nil {
		return Outcome{}, err
	}
	out := Outcome{
		Kind:      OutcomeAction,
		Action:    inv.Definition.Action,
		Command:   inv.Definition.Name,
		Arguments: inv.Arguments,
		Agent:     inv.Definition.Agent,
	}
	if inv.Definition.Action == ActionPrompt {
		out.Kind = OutcomePrompt
		out.Prompt = strings.ReplaceAll(inv.Definition.Template, "$ARGUMENTS", inv.Arguments)
	}
	return out, nil
}

func commandToken(input string) string {
	input = strings.TrimPrefix(strings.TrimSpace(input), "/")
	for i, ch := range input {
		if unicode.IsSpace(ch) {
			input = input[:i]
			break
		}
	}
	return normalizeName(input)
}

func normalizeName(v string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(v, "/")))
}

func validName(v string) bool {
	if v == "" {
		return false
	}
	for _, ch := range v {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '-' || ch == '_' || ch == '.' {
			continue
		}
		return false
	}
	return true
}
