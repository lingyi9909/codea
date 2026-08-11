package architecture

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestVendorBoundaryNoLeakage(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "-json", "./...")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}

	forbidden := "codea/tui/internal/opencode"

	decoder := json.NewDecoder(strings.NewReader(string(output)))
	for decoder.More() {
		var pkg struct {
			ImportPath string   `json:"ImportPath"`
			Imports    []string `json:"Imports"`
		}
		if err := decoder.Decode(&pkg); err != nil {
			continue
		}

		isOpencode := strings.Contains(pkg.ImportPath, "internal/opencode")
		isCmd := strings.Contains(pkg.ImportPath, "/cmd/")
		isTest := strings.Contains(pkg.ImportPath, "/tests/")

		if isOpencode || isCmd || isTest {
			continue
		}

		for _, imp := range pkg.Imports {
			if imp == forbidden {
				t.Errorf("forbidden import: %s imports %s", pkg.ImportPath, forbidden)
			}
		}
	}
}
