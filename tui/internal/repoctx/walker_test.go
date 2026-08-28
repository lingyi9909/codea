package repoctx

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func writeWalkerFile(t *testing.T, root, rel, content string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestWalkerIncludesSourceAndExcludesGeneratedTrees(t *testing.T) {
	root := t.TempDir()
	writeWalkerFile(t, root, "src/main/java/com/acme/App.java", "class App {}")
	for _, rel := range []string{"target/A.java", "build/B.go", "dist/C.ts", "node_modules/x.js", ".git/config", "vendor/v.go"} {
		writeWalkerFile(t, root, rel, "source")
	}
	w := NewWalker(root, WalkerOptions{})
	files, err := w.Walk()
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(files))
	for i := range files {
		got[i] = files[i].Path
	}
	want := []string{"src/main/java/com/acme/App.java"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("files=%v want=%v", got, want)
	}
}

func TestWalkerExcludesBinaryAndOversizedFiles(t *testing.T) {
	root := t.TempDir()
	writeWalkerFile(t, root, "src/a.go", "package a")
	writeWalkerFile(t, root, "src/b.bin", "abc\x00def")
	writeWalkerFile(t, root, "src/large.java", strings.Repeat("x", 33))
	files, err := NewWalker(root, WalkerOptions{MaxFileSize: 32}).Walk()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != "src/a.go" {
		t.Fatalf("files=%+v", files)
	}
}

func TestWalkerSymlinkOutsideRootExcluded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privilege varies on Windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	target := writeWalkerFile(t, outside, "secret.java", "class Secret {}")
	if err := os.Symlink(target, filepath.Join(root, "leak.java")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	files, err := NewWalker(root, WalkerOptions{}).Walk()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("outside symlink included: %+v", files)
	}
}

func TestNormalizeRelativePathAcceptsWindowsBackslashes(t *testing.T) {
	got, ok := normalizeRelativePath(`src\\main\\java\\Order.java`)
	if !ok || got != "src/main/java/Order.java" {
		t.Fatalf("got=%q ok=%v", got, ok)
	}
}

func TestWalkerUnicodeRootAndStableOrder(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project space 中文")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	writeWalkerFile(t, root, "z/Z.go", "package z")
	writeWalkerFile(t, root, "a/A.java", "class A {}")
	w := NewWalker(root, WalkerOptions{})
	first, err := w.Walk()
	if err != nil {
		t.Fatal(err)
	}
	second, err := w.Walk()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("walk order unstable\n%+v\n%+v", first, second)
	}
	if first[0].Path != "a/A.java" || first[1].Path != "z/Z.go" {
		t.Fatalf("unexpected order: %+v", first)
	}
}
