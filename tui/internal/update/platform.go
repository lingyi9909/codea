package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type updateLock interface {
	Release() error
	Close() error
}

type Switcher interface {
	Current() (string, error)
	Switch(target string) error
}

func NewPlatformSwitcher(home string) Switcher { return newPlatformSwitcher(home) }

func validateVersionTarget(home, target string) (string, error) {
	abs, err := filepath.Abs(target)
	if err != nil { return "", err }
	versions, err := filepath.Abs(filepath.Join(home, "versions"))
	if err != nil { return "", err }
	rel, err := filepath.Rel(versions, abs)
	if err != nil { return "", err }
	if rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." || filepath.IsAbs(rel) || strings.Contains(rel, string(os.PathSeparator)) {
		return "", fmt.Errorf("version target must be a direct child of %s: %s", versions, abs)
	}
	st, err := os.Stat(abs)
	if err != nil { return "", err }
	if !st.IsDir() { return "", fmt.Errorf("target is not a version directory: %s", abs) }
	return abs, nil
}
