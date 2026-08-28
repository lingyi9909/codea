package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codea/tui/internal/repoctx"
)

func TestRepoMapCompositionUsesResolvedProjectDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project space 中文")
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "OnlyHere.go"), []byte("package onlyhere\nfunc OnlyHere() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := repoctx.NewService(root).BuildMap(context.Background(), repoctx.Query{Text: "OnlyHere", MaxChars: 4000})
	if err != nil {
		t.Fatal(err)
	}
	rendered := m.Render()
	if !strings.Contains(rendered, "src/OnlyHere.go") {
		t.Fatalf("repo map did not use explicit project root:\n%s", rendered)
	}
}
