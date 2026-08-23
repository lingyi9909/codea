package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type VersionManager struct{ root string }

func NewVersionManager(home string) *VersionManager { return &VersionManager{root: filepath.Join(home, "versions")} }
func (v *VersionManager) Path(version string) string { return filepath.Join(v.root, version) }
func (v *VersionManager) Install(stagedRoot, version string) (string, error) {
	if strings.TrimSpace(version) == "" || strings.ContainsAny(version, "/\\") { return "", fmt.Errorf("invalid version: %q", version) }
	if err := os.MkdirAll(v.root, 0o755); err != nil { return "", err }
	target := v.Path(version)
	if _, err := os.Stat(target); err == nil { return "", fmt.Errorf("version already installed: %s", version) } else if !os.IsNotExist(err) { return "", err }
	if err := os.Rename(stagedRoot, target); err != nil { return "", fmt.Errorf("install version %s: %w", version, err) }
	return target, nil
}
func (v *VersionManager) Remove(version string) error { if version == "" { return nil }; return os.RemoveAll(v.Path(version)) }
func (v *VersionManager) Exists(version string) bool { st, err := os.Stat(v.Path(version)); return err == nil && st.IsDir() }
