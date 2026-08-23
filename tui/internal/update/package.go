package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxExtractBytes int64 = 1 << 30
const maxExtractFiles = 20000

func PreparePackage(packagePath, stagingDir string) (string, error) {
	if packagePath == "" || stagingDir == "" {
		return "", fmt.Errorf("package path and staging dir are required")
	}
	if err := os.RemoveAll(stagingDir); err != nil { return "", err }
	if err := os.MkdirAll(stagingDir, 0o700); err != nil { return "", err }
	st, err := os.Stat(packagePath)
	if err != nil { return "", err }
	if st.IsDir() {
		if err := copyTree(packagePath, filepath.Join(stagingDir, "release")); err != nil { return "", err }
	} else {
		lower := strings.ToLower(packagePath)
		switch {
		case strings.HasSuffix(lower, ".zip"):
			err = extractZip(packagePath, stagingDir)
		case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
			err = extractTarGz(packagePath, stagingDir)
		default:
			return "", fmt.Errorf("unsupported package format: %s", packagePath)
		}
		if err != nil { return "", err }
	}
	return discoverReleaseRoot(stagingDir)
}

func discoverReleaseRoot(stage string) (string, error) {
	if _, err := os.Stat(filepath.Join(stage, "VERSION")); err == nil { return stage, nil }
	if _, err := os.Stat(filepath.Join(stage, "release", "VERSION")); err == nil { return filepath.Join(stage, "release"), nil }
	entries, err := os.ReadDir(stage)
	if err != nil { return "", err }
	var roots []string
	for _, e := range entries {
		if e.IsDir() {
			p := filepath.Join(stage, e.Name())
			if _, err := os.Stat(filepath.Join(p, "VERSION")); err == nil { roots = append(roots, p) }
		}
	}
	if len(roots) != 1 { return "", fmt.Errorf("package must contain exactly one release root, found %d", len(roots)) }
	return roots[0], nil
}

func copyTree(src, dst string) error {
	srcAbs, err := filepath.Abs(src)
	if err != nil { return err }
	return filepath.WalkDir(srcAbs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil { return walkErr }
		rel, _ := filepath.Rel(srcAbs, path)
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil { return err }
		if info.Mode()&os.ModeSymlink != 0 { return fmt.Errorf("symlink not allowed in package: %s", rel) }
		if d.IsDir() { return os.MkdirAll(target, info.Mode().Perm()) }
		if !info.Mode().IsRegular() { return fmt.Errorf("non-regular file in package: %s", rel) }
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { return err }
		in, err := os.Open(path)
		if err != nil { return err }
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil { in.Close(); return err }
		_, copyErr := io.Copy(out, in)
		inErr := in.Close(); closeErr := out.Close()
		if copyErr != nil { return copyErr }
		if inErr != nil { return inErr }
		return closeErr
	})
}

func extractZip(path, dst string) error {
	r, err := zip.OpenReader(path)
	if err != nil { return err }
	defer r.Close()
	var total int64
	if len(r.File) > maxExtractFiles { return fmt.Errorf("archive has too many files") }
	for _, f := range r.File {
		rel, err := safeArchivePath(f.Name)
		if err != nil { return err }
		if f.Mode()&os.ModeSymlink != 0 { return fmt.Errorf("symlink not allowed in archive: %s", rel) }
		target := filepath.Join(dst, filepath.FromSlash(rel))
		if f.FileInfo().IsDir() { if err := os.MkdirAll(target, 0o755); err != nil { return err }; continue }
		total += int64(f.UncompressedSize64)
		if total > maxExtractBytes { return fmt.Errorf("archive exceeds extraction limit") }
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { return err }
		rc, err := f.Open(); if err != nil { return err }
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode().Perm())
		if err != nil { rc.Close(); return err }
		_, e1 := io.Copy(out, io.LimitReader(rc, maxExtractBytes+1)); e2 := rc.Close(); e3 := out.Close()
		if e1 != nil { return e1 }; if e2 != nil { return e2 }; if e3 != nil { return e3 }
	}
	return nil
}

func extractTarGz(path, dst string) error {
	f, err := os.Open(path); if err != nil { return err }; defer f.Close()
	gz, err := gzip.NewReader(f); if err != nil { return err }; defer gz.Close()
	tr := tar.NewReader(gz)
	var total int64; count := 0
	for {
		h, err := tr.Next(); if err == io.EOF { break }; if err != nil { return err }
		count++; if count > maxExtractFiles { return fmt.Errorf("archive has too many files") }
		rel, err := safeArchivePath(h.Name); if err != nil { return err }
		target := filepath.Join(dst, filepath.FromSlash(rel))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil { return err }
		case tar.TypeReg, tar.TypeRegA:
			total += h.Size; if total > maxExtractBytes { return fmt.Errorf("archive exceeds extraction limit") }
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { return err }
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(h.Mode).Perm()); if err != nil { return err }
			_, e := io.CopyN(out, tr, h.Size); ce := out.Close(); if e != nil { return e }; if ce != nil { return ce }
		default:
			return fmt.Errorf("unsupported archive entry type for %s", rel)
		}
	}
	return nil
}

func safeArchivePath(name string) (string, error) {
	s := strings.ReplaceAll(name, "\\", "/")
	if s == "" || strings.HasPrefix(s, "/") { return "", fmt.Errorf("unsafe archive path: %s", name) }
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(s)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(filepath.FromSlash(s)) { return "", fmt.Errorf("unsafe archive path: %s", name) }
	return clean, nil
}
