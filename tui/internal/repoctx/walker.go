package repoctx

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const defaultMaxFileSize int64 = 1 << 20
const binaryProbeSize = 8192

type WalkerOptions struct {
	MaxFileSize int64
}

type Walker struct {
	root        string
	maxFileSize int64
}

func NewWalker(root string, options WalkerOptions) *Walker {
	max := options.MaxFileSize
	if max <= 0 {
		max = defaultMaxFileSize
	}
	return &Walker{root: root, maxFileSize: max}
}

var excludedDirs = map[string]struct{}{
	".git": {}, "target": {}, "build": {}, "dist": {}, "node_modules": {}, "vendor": {},
	"out": {}, ".idea": {}, ".gradle": {},
}

func (w *Walker) Walk() ([]SourceFile, error) {
	root, err := filepath.Abs(w.root)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve project root symlink: %w", err)
	}

	files := make([]SourceFile, 0)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		norm, ok := normalizeRelativePath(rel)
		if !ok {
			return nil
		}
		if norm == "." || norm == "" {
			return nil
		}

		if entry.IsDir() {
			if isExcludedDir(norm) {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil
			}
			inside, err := isWithinRoot(root, resolved)
			if err != nil || !inside {
				return nil
			}
			resolvedInfo, err := os.Stat(resolved)
			if err != nil || resolvedInfo.IsDir() {
				return nil
			}
			info = resolvedInfo
			path = resolved
		}
		if info.Size() > w.maxFileSize {
			return nil
		}
		binary, err := isBinaryFile(path)
		if err != nil || binary {
			return nil
		}
		files = append(files, SourceFile{Path: norm, AbsPath: path, Extension: strings.ToLower(filepath.Ext(norm))})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func normalizeRelativePath(rel string) (string, bool) {
	rel = strings.TrimSpace(normalizeSlash(rel))
	if rel == "" {
		return "", false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	clean = normalizeSlash(clean)
	if clean == "." {
		return ".", true
	}
	if strings.HasPrefix(clean, "../") || clean == ".." || strings.HasPrefix(clean, "/") {
		return "", false
	}
	if len(clean) >= 2 && clean[1] == ':' {
		return "", false
	}
	return clean, true
}

func isExcludedDir(rel string) bool {
	rel = normalizeSlash(rel)
	if rel == ".mvn/wrapper" || strings.HasSuffix(rel, "/.mvn/wrapper") {
		return true
	}
	parts := strings.Split(rel, "/")
	for _, part := range parts {
		if _, ok := excludedDirs[part]; ok {
			return true
		}
	}
	return false
}

func isWithinRoot(root, candidate string) (bool, error) {
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return false, err
	}
	_, ok := normalizeRelativePath(rel)
	return ok, nil
}

func isBinaryFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	buf := make([]byte, binaryProbeSize)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return false, err
	}
	buf = buf[:n]
	if bytes.IndexByte(buf, 0) >= 0 {
		return true, nil
	}
	return false, nil
}
