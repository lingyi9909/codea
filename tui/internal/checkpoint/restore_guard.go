package checkpoint

import (
	"os"
	"path/filepath"
	"strings"
)

func (s *Service) ensureRestoreAncestorsSafe(full string) error {
	root := filepath.Clean(s.repo.ProjectRoot())
	parent := filepath.Dir(filepath.Clean(full))
	if !insidePath(root, parent) {
		return &Error{Code: CodeStateCorrupt, Message: "restore path ancestor escapes project root"}
	}

	rel, err := filepath.Rel(root, parent)
	if err != nil {
		return &Error{Code: CodeStateCorrupt, Message: "resolve restore path ancestors", Cause: err}
	}
	if rel == "." {
		return nil
	}

	current := root
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			// Once an ancestor does not exist, no deeper ancestor can exist yet.
			return nil
		}
		if err != nil {
			return &Error{Code: CodeStateCorrupt, Message: "inspect restore path ancestor", Cause: err}
		}
		linkLike, err := restorePathIsLinkLike(current, info)
		if err != nil {
			return &Error{Code: CodeStateCorrupt, Message: "inspect restore path reparse attributes", Cause: err}
		}
		if linkLike {
			return &Error{Code: CodeStateCorrupt, Message: "restore path has symlink or reparse-point ancestor"}
		}
	}
	return nil
}

func restorePathProtectedBySkipped(rel string, skipped []SkippedPath) bool {
	rel = normalizeRestoreMetadataPath(rel)
	if rel == "" {
		return false
	}
	for _, item := range skipped {
		base := normalizeRestoreMetadataPath(item.Path)
		if base == "" {
			continue
		}
		if rel == base || strings.HasPrefix(rel, base+"/") {
			return true
		}
	}
	return false
}

func normalizeRestoreMetadataPath(path string) string {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(path))))
	if clean == "." || clean == "/" {
		return ""
	}
	return strings.Trim(clean, "/")
}
