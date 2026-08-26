package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const lockedOpenCodeVersion = "1.18.11"

// prepareOfflinePluginDependencies prevents OpenCode v1.18.11 from attempting
// an npm install before loading Codea's self-contained file:// plugin.
//
// OpenCode starts a dependency install for OPENCODE_CONFIG_DIR and, when an
// external plugin is configured, /agent waits for that install to finish. In an
// air-gapped environment this can block until the HTTP client times out even
// though Codea's plugin bundle has no runtime npm dependencies. OpenCode's npm
// service skips reify when node_modules exists and package.json/package-lock.json
// already declare the pinned @opencode-ai/plugin dependency, so Codea writes the
// minimal deterministic metadata it needs before spawning the runtime.
func prepareOfflinePluginDependencies(cfgDir, openCodeVersion string) error {
	if openCodeVersion != lockedOpenCodeVersion {
		return fmt.Errorf("unsupported OpenCode dependency bootstrap version %q; locked=%s", openCodeVersion, lockedOpenCodeVersion)
	}
	if err := os.MkdirAll(filepath.Join(cfgDir, "node_modules"), 0o755); err != nil {
		return fmt.Errorf("create runtime node_modules sentinel: %w", err)
	}

	pkg := struct {
		Private      bool              `json:"private"`
		Dependencies map[string]string `json:"dependencies"`
	}{
		Private: true,
		Dependencies: map[string]string{
			"@opencode-ai/plugin": openCodeVersion,
		},
	}
	if err := writeJSONFile(filepath.Join(cfgDir, "package.json"), pkg); err != nil {
		return fmt.Errorf("write offline package.json: %w", err)
	}

	lock := struct {
		Name            string `json:"name"`
		LockfileVersion int    `json:"lockfileVersion"`
		Requires        bool   `json:"requires"`
		Packages        map[string]struct {
			Dependencies map[string]string `json:"dependencies,omitempty"`
		} `json:"packages"`
	}{
		Name:            "codea-runtime-config",
		LockfileVersion: 3,
		Requires:        true,
		Packages: map[string]struct {
			Dependencies map[string]string `json:"dependencies,omitempty"`
		}{
			"": {
				Dependencies: map[string]string{
					"@opencode-ai/plugin": openCodeVersion,
				},
			},
		},
	}
	if err := writeJSONFile(filepath.Join(cfgDir, "package-lock.json"), lock); err != nil {
		return fmt.Errorf("write offline package-lock.json: %w", err)
	}
	return nil
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
