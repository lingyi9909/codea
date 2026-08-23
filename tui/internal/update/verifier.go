package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ManifestEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type Manifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	Algorithm     string          `json:"algorithm"`
	Files         []ManifestEntry `json:"files"`
}
type ReleaseInfo struct {
	Root                string
	Version             string
	ConfigSchemaVersion int
	Manifest            Manifest
}
type Verifier struct{}

func (Verifier) Verify(root string) (*ReleaseInfo, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("release root is not a directory")
	}
	mb, err := os.ReadFile(filepath.Join(abs, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(mb, &m); err != nil {
		return nil, fmt.Errorf("manifest decode: %w", err)
	}
	if m.SchemaVersion != 1 || m.Algorithm != "sha256" {
		return nil, fmt.Errorf("unsupported manifest schema/algorithm")
	}
	declared := map[string]ManifestEntry{}
	for _, e := range m.Files {
		rel, err := safeRel(e.Path)
		if err != nil {
			return nil, err
		}
		if _, dup := declared[rel]; dup {
			return nil, fmt.Errorf("duplicate manifest path: %s", rel)
		}
		if len(e.SHA256) != 64 || e.Size < 0 {
			return nil, fmt.Errorf("invalid manifest entry: %s", rel)
		}
		declared[rel] = e
		path := filepath.Join(abs, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("manifest file missing %s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("manifest path is not a regular file: %s", rel)
		}
		if info.Size() != e.Size {
			return nil, fmt.Errorf("size mismatch: %s", rel)
		}
		hash, err := sha256File(path)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(hash, e.SHA256) {
			return nil, fmt.Errorf("checksum mismatch: %s", rel)
		}
	}
	err = filepath.WalkDir(abs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == abs {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink not allowed in release: %s", path)
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(abs, path)
		rel = filepath.ToSlash(rel)
		if rel == "manifest.json" {
			return nil
		}
		if _, ok := declared[rel]; !ok {
			return fmt.Errorf("unmanifested file: %s", rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	vb, err := os.ReadFile(filepath.Join(abs, "VERSION"))
	if err != nil {
		return nil, fmt.Errorf("VERSION: %w", err)
	}
	version := strings.TrimSpace(string(vb))
	if version == "" {
		return nil, fmt.Errorf("VERSION is empty")
	}
	schema := 1
	if b, err := os.ReadFile(filepath.Join(abs, "config", "codea-schema-version")); err == nil {
		n, e := strconv.Atoi(strings.TrimSpace(string(b)))
		if e != nil || n <= 0 {
			return nil, fmt.Errorf("invalid config schema version")
		}
		schema = n
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return &ReleaseInfo{Root: abs, Version: version, ConfigSchemaVersion: schema, Manifest: m}, nil
}

func safeRel(input string) (string, error) {
	s := strings.ReplaceAll(input, "\\", "/")
	if s == "" || strings.HasPrefix(s, "/") {
		return "", fmt.Errorf("unsafe manifest path: %s", input)
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(s)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(filepath.FromSlash(s)) {
		return "", fmt.Errorf("unsafe manifest path: %s", input)
	}
	return clean, nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
