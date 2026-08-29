package repoctx

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Indexer struct {
	root   string
	walker *Walker
}

func NewIndexer(root string) *Indexer {
	return &Indexer{root: root, walker: NewWalker(root, WalkerOptions{})}
}

func (i *Indexer) Build(ctx context.Context) (RepositoryIndex, error) {
	_ = ctx
	walked, err := i.walker.Walk()
	if err != nil {
		return RepositoryIndex{}, err
	}
	goModulePath := singleRootGoModulePath(i.root, walked)
	idx := RepositoryIndex{}
	for _, wf := range walked {
		data, err := os.ReadFile(wf.AbsPath)
		if err != nil {
			idx.Unresolved = append(idx.Unresolved, fmt.Sprintf("%s: read: %v", wf.Path, err))
			continue
		}
		var f SourceFile
		switch wf.Extension {
		case ".java":
			f = ExtractJava(wf.Path, data)
		case ".go":
			f = ExtractGo(wf.Path, data)
		default:
			f = ExtractFallback(wf.Path, data)
		}
		f.AbsPath = wf.AbsPath
		idx.Files = append(idx.Files, f)
		idx.Symbols = append(idx.Symbols, f.Symbols...)
		idx.Unresolved = append(idx.Unresolved, f.Unresolved...)
		idx.candidates = append(idx.candidates, f.candidates...)
	}
	idx.resolveCandidates(goModulePath)
	sort.Slice(idx.Files, func(a, b int) bool { return idx.Files[a].Path < idx.Files[b].Path })
	sort.Slice(idx.Symbols, func(a, b int) bool { return idx.Symbols[a].ID < idx.Symbols[b].ID })
	sort.Slice(idx.Relations, func(a, b int) bool {
		ra, rb := idx.Relations[a], idx.Relations[b]
		if ra.From != rb.From {
			return ra.From < rb.From
		}
		if ra.To != rb.To {
			return ra.To < rb.To
		}
		return ra.Kind < rb.Kind
	})
	idx.Unresolved = uniqueSorted(idx.Unresolved)
	return idx, nil
}

func (idx *RepositoryIndex) resolveCandidates(goModulePath string) {
	types := map[string][]Symbol{}
	methods := map[string][]Symbol{}
	symbolsByID := map[string]Symbol{}
	filesByPath := map[string]SourceFile{}
	for _, f := range idx.Files {
		filesByPath[f.Path] = f
	}
	for _, s := range idx.Symbols {
		symbolsByID[s.ID] = s
		if s.Owner == "" && (s.Kind == SymbolClass || s.Kind == SymbolInterface || s.Kind == SymbolEnum || s.Kind == SymbolRecord || s.Kind == SymbolType) {
			types[s.Name] = append(types[s.Name], s)
		}
		if s.Owner != "" && (s.Kind == SymbolMethod || s.Kind == SymbolFunction) {
			methods[s.Owner+"\x00"+s.Name] = append(methods[s.Owner+"\x00"+s.Name], s)
		}
	}
	for _, c := range idx.candidates {
		from, ok := symbolsByID[c.from]
		if !ok {
			idx.Unresolved = append(idx.Unresolved, resolutionMessage(c, 0))
			continue
		}
		source, ok := filesByPath[from.Path]
		if !ok {
			idx.Unresolved = append(idx.Unresolved, resolutionMessage(c, 0))
			continue
		}
		targetType := simpleTypeName(c.targetType)
		switch c.kind {
		case RelationInjects:
			matches := contextualMatches(source, c.targetType, types[targetType], goModulePath)
			if len(matches) == 1 {
				idx.Relations = append(idx.Relations, Relation{From: c.from, To: matches[0].ID, Kind: c.kind, Confidence: c.confidence, Evidence: c.evidence})
			} else {
				idx.Unresolved = append(idx.Unresolved, resolutionMessage(c, len(matches)))
			}
		case RelationCalls:
			matches := contextualMatches(source, c.targetType, methods[targetType+"\x00"+c.targetMethod], goModulePath)
			if len(matches) == 1 {
				idx.Relations = append(idx.Relations, Relation{From: c.from, To: matches[0].ID, Kind: c.kind, Confidence: c.confidence, Evidence: c.evidence})
			} else {
				idx.Unresolved = append(idx.Unresolved, resolutionMessage(c, len(matches)))
			}
		}
	}
}

func contextualMatches(source SourceFile, targetType string, matches []Symbol, goModulePath string) []Symbol {
	if len(matches) == 0 {
		return nil
	}
	switch source.Extension {
	case ".java":
		return javaContextualMatches(source, targetType, matches)
	case ".go":
		return goContextualMatches(source, targetType, matches, goModulePath)
	default:
		return nil
	}
}

func javaContextualMatches(source SourceFile, targetType string, matches []Symbol) []Symbol {
	clean := cleanTypeReference(targetType)
	simple := simpleTypeName(clean)
	if i := strings.LastIndex(clean, "."); i > 0 {
		return symbolsInPackage(matches, clean[:i])
	}

	exactPackages := map[string]struct{}{}
	for _, imp := range source.Imports {
		if strings.HasSuffix(imp, "."+simple) && !strings.HasSuffix(imp, ".*") {
			exactPackages[strings.TrimSuffix(imp, "."+simple)] = struct{}{}
		}
	}
	if len(exactPackages) > 0 {
		return symbolsInPackages(matches, exactPackages)
	}

	if same := symbolsInPackage(matches, source.Package); len(same) > 0 {
		return same
	}

	wildcardPackages := map[string]struct{}{}
	for _, imp := range source.Imports {
		if strings.HasSuffix(imp, ".*") {
			wildcardPackages[strings.TrimSuffix(imp, ".*")] = struct{}{}
		}
	}
	return symbolsInPackages(matches, wildcardPackages)
}

func goContextualMatches(source SourceFile, targetType string, matches []Symbol, goModulePath string) []Symbol {
	clean := cleanTypeReference(targetType)
	if i := strings.LastIndex(clean, "."); i > 0 {
		qualifier := clean[:i]
		importPath, ok := source.ImportAliases[qualifier]
		if !ok || goModulePath == "" {
			return nil
		}
		out := make([]Symbol, 0, len(matches))
		for _, s := range matches {
			if goImportMatchesSymbol(importPath, goModulePath, s) {
				out = append(out, s)
			}
		}
		return out
	}

	sourceDir := pathpkg.Dir(source.Path)
	out := make([]Symbol, 0, len(matches))
	for _, s := range matches {
		if s.Package == source.Package && pathpkg.Dir(s.Path) == sourceDir {
			out = append(out, s)
		}
	}
	return out
}

func goImportMatchesSymbol(importPath, modulePath string, s Symbol) bool {
	if modulePath == "" {
		return false
	}
	dir := pathpkg.Clean(pathpkg.Dir(s.Path))
	canonical := strings.TrimSuffix(modulePath, "/")
	if dir != "." {
		canonical += "/" + dir
	}
	return importPath == canonical
}

func singleRootGoModulePath(root string, walked []SourceFile) string {
	foundRoot := false
	for _, f := range walked {
		if pathpkg.Base(f.Path) != "go.mod" {
			continue
		}
		if f.Path != "go.mod" || foundRoot {
			return ""
		}
		foundRoot = true
	}
	if !foundRoot {
		return ""
	}
	return readGoModuleDirective(filepath.Join(root, "go.mod"))
}

func readGoModuleDirective(filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(io.LimitReader(f, 64<<10))
	modulePath := ""
	moduleCount := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "module" {
			continue
		}
		moduleCount++
		if moduleCount > 1 {
			return ""
		}
		candidate := fields[1]
		if strings.HasPrefix(candidate, "\"") {
			unquoted, err := strconv.Unquote(candidate)
			if err != nil {
				return ""
			}
			candidate = unquoted
		}
		candidate = strings.TrimSuffix(candidate, "/")
		if candidate == "" || strings.Contains(candidate, "\\") || strings.ContainsAny(candidate, " \t\r\n") || strings.HasPrefix(candidate, ".") {
			return ""
		}
		modulePath = candidate
	}
	if scanner.Err() != nil || moduleCount != 1 {
		return ""
	}
	return modulePath
}

func symbolsInPackage(matches []Symbol, pkg string) []Symbol {
	out := make([]Symbol, 0, len(matches))
	for _, s := range matches {
		if s.Package == pkg {
			out = append(out, s)
		}
	}
	return out
}

func symbolsInPackages(matches []Symbol, pkgs map[string]struct{}) []Symbol {
	if len(pkgs) == 0 {
		return nil
	}
	out := make([]Symbol, 0, len(matches))
	for _, s := range matches {
		if _, ok := pkgs[s.Package]; ok {
			out = append(out, s)
		}
	}
	return out
}

func cleanTypeReference(s string) string {
	s = strings.TrimSpace(s)
	return strings.Trim(s, "[]*")
}

func resolutionMessage(c relationCandidate, n int) string {
	state := "unresolved"
	if n > 1 {
		state = "ambiguous"
	}
	target := simpleTypeName(c.targetType)
	if c.targetMethod != "" {
		target += "#" + c.targetMethod
	}
	return fmt.Sprintf("%s relation %s -> %s (%d matches)", state, c.from, target, n)
}

func simpleTypeName(s string) string {
	s = cleanTypeReference(s)
	if i := strings.LastIndex(s, "."); i >= 0 {
		s = s[i+1:]
	}
	return s
}

func uniqueSorted(xs []string) []string {
	m := map[string]struct{}{}
	for _, x := range xs {
		if strings.TrimSpace(x) != "" {
			m[x] = struct{}{}
		}
	}
	out := make([]string, 0, len(m))
	for x := range m {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}
