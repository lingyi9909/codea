package capability

import "codea/tui/internal/runtime"

// CompareResult holds the outcome of comparing product requirements against
// actual Runtime capabilities.
type CompareResult struct {
	RequiredSupported []string
	RequiredMissing   []string
	OptionalSupported []string
	OptionalMissing   []string
	Deferred          []string
}

// HasRequiredFailures returns true if any required capability is missing.
func (r *CompareResult) HasRequiredFailures() bool {
	return len(r.RequiredMissing) > 0
}

// nameToField maps a capability YAML name to the corresponding
// RuntimeCapabilities boolean field value.
var nameToField = map[string]func(runtime.RuntimeCapabilities) bool{
	"sessions":          func(c runtime.RuntimeCapabilities) bool { return c.Sessions },
	"streaming":         func(c runtime.RuntimeCapabilities) bool { return c.Streaming },
	"reasoning":         func(c runtime.RuntimeCapabilities) bool { return c.Reasoning },
	"fileRead":          func(c runtime.RuntimeCapabilities) bool { return c.FileRead },
	"fileWrite":         func(c runtime.RuntimeCapabilities) bool { return c.FileWrite },
	"edit":              func(c runtime.RuntimeCapabilities) bool { return c.Edit },
	"bash":              func(c runtime.RuntimeCapabilities) bool { return c.Bash },
	"toolApproval":      func(c runtime.RuntimeCapabilities) bool { return c.ToolApproval },
	"agents":            func(c runtime.RuntimeCapabilities) bool { return c.Agents },
	"subagents":         func(c runtime.RuntimeCapabilities) bool { return c.Subagents },
	"skills":            func(c runtime.RuntimeCapabilities) bool { return c.Skills },
	"plugins":           func(c runtime.RuntimeCapabilities) bool { return c.Plugins },
	"abort":             func(c runtime.RuntimeCapabilities) bool { return c.Abort },
	"messageHistory":    func(c runtime.RuntimeCapabilities) bool { return c.MessageHistory },
	"contextCompaction": func(c runtime.RuntimeCapabilities) bool { return c.ContextCompaction },
}

// Compare evaluates each product requirement against the Runtime's declared
// capabilities and categorizes the results.
func (inv *Inventory) Compare(caps runtime.RuntimeCapabilities) *CompareResult {
	result := &CompareResult{}

	for _, req := range inv.Requirements {
		switch req.Level {
		case Required:
			if lookup, ok := nameToField[req.Name]; ok && lookup(caps) {
				result.RequiredSupported = append(result.RequiredSupported, req.Name)
			} else {
				result.RequiredMissing = append(result.RequiredMissing, req.Name)
			}
		case Optional:
			if lookup, ok := nameToField[req.Name]; ok && lookup(caps) {
				result.OptionalSupported = append(result.OptionalSupported, req.Name)
			} else {
				result.OptionalMissing = append(result.OptionalMissing, req.Name)
			}
		case Deferred:
			result.Deferred = append(result.Deferred, req.Name)
		}
	}

	return result
}
