package capability

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// RequirementLevel classifies whether a capability is required, optional, or deferred.
type RequirementLevel string

const (
	Required RequirementLevel = "required"
	Optional RequirementLevel = "optional"
	Deferred RequirementLevel = "deferred"
)

// CapabilityRequirement is a single product capability requirement.
type CapabilityRequirement struct {
	Name  string
	Level RequirementLevel
}

// Inventory holds the product capability requirements loaded from a YAML file.
type Inventory struct {
	Requirements []CapabilityRequirement
}

// Load reads a capability requirements YAML file and returns an Inventory.
// It only parses the "capabilities:" section; it does not depend on any
// third-party YAML library.
func Load(path string) (*Inventory, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read capabilities file: %w", err)
	}
	defer f.Close()

	var reqs []CapabilityRequirement
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(f)
	inCapabilities := false
	for scanner.Scan() {
		line := scanner.Text()

		if !inCapabilities {
			if strings.TrimSpace(line) == "capabilities:" {
				inCapabilities = true
			}
			continue
		}

		// Exit the capabilities section when we hit a non-indented key.
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "\t") {
			// End of indented block.
			break
		}

		parts := strings.SplitN(strings.TrimSpace(trimmed), ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		level := strings.TrimSpace(parts[1])

		if level == "" {
			return nil, fmt.Errorf("empty requirement level for capability %q", name)
		}

		rl := RequirementLevel(level)
		switch rl {
		case Required, Optional, Deferred:
		default:
			return nil, fmt.Errorf("unknown requirement level %q for capability %q", level, name)
		}

		if seen[name] {
			return nil, fmt.Errorf("duplicate capability %q", name)
		}
		seen[name] = true

		reqs = append(reqs, CapabilityRequirement{Name: name, Level: rl})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read capabilities file: %w", err)
	}

	return &Inventory{Requirements: reqs}, nil
}
