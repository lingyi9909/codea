package repoctx

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const defaultRepoMapChars = 8000

type Service struct {
	root    string
	indexer *Indexer
	runner  CommandRunner
}

func NewService(root string) *Service {
	return &Service{root: root, indexer: NewIndexer(root)}
}

// Invalidate drops the current indexer instance after an external source-tree
// restore. Build currently rescans files on every call, but keeping this explicit
// boundary prevents future caching from serving pre-restore repository state.
func (s *Service) Invalidate() {
	if s == nil {
		return
	}
	s.indexer = NewIndexer(s.root)
}

func (s *Service) BuildMap(ctx context.Context, q Query) (RepoMap, error) {
	idx, err := s.indexer.Build(ctx)
	if err != nil {
		return RepoMap{}, err
	}
	changed := normalizeChangedFiles(q.ChangedFiles)
	if len(changed) == 0 {
		changed = normalizeChangedFiles(GitChangedFiles(ctx, s.root, s.runner))
	}
	q.ChangedFiles = changed
	ranked := Rank(idx, q)

	m := RepoMap{ChangedFiles: changed, maxChars: q.MaxChars}
	if m.maxChars <= 0 {
		m.maxChars = defaultRepoMapChars
	}

	pathOrder := map[string]int{}
	for _, item := range ranked {
		if isSensitiveRepoPath(item.Path) {
			continue
		}
		if _, exists := pathOrder[item.Path]; exists {
			continue
		}
		pathOrder[item.Path] = len(m.Files)
		m.Files = append(m.Files, item.Path)
	}

	for _, symbol := range idx.Symbols {
		if _, selected := pathOrder[symbol.Path]; selected && !isSensitiveRepoPath(symbol.Path) {
			m.Symbols = append(m.Symbols, symbol)
		}
	}
	sort.SliceStable(m.Symbols, func(i, j int) bool {
		pi, pj := pathOrder[m.Symbols[i].Path], pathOrder[m.Symbols[j].Path]
		if pi != pj {
			return pi < pj
		}
		if m.Symbols[i].StartLine != m.Symbols[j].StartLine {
			return m.Symbols[i].StartLine < m.Symbols[j].StartLine
		}
		return m.Symbols[i].ID < m.Symbols[j].ID
	})

	selectedSymbols := map[string]bool{}
	for _, symbol := range m.Symbols {
		selectedSymbols[symbol.ID] = true
	}
	for _, relation := range idx.Relations {
		if selectedSymbols[relation.From] && selectedSymbols[relation.To] {
			m.Relations = append(m.Relations, relation)
		}
	}
	for _, unresolved := range idx.Unresolved {
		if !containsSensitiveRepoEvidence(unresolved) {
			m.Unresolved = append(m.Unresolved, unresolved)
		}
	}
	m.ChangedFiles = filterSensitivePaths(m.ChangedFiles)
	m.Summary = fmt.Sprintf("%d relevant files, %d symbols, %d confirmed relations", len(m.Files), len(m.Symbols), len(m.Relations))
	return m, nil
}

func (m *RepoMap) Render() string {
	m.Truncated = false
	budget := m.maxChars
	if budget <= 0 {
		budget = defaultRepoMapChars
	}
	if budget == 0 {
		m.Truncated = true
		return ""
	}

	var b strings.Builder
	used := 0
	appendLine := func(line string) bool {
		candidate := line + "\n"
		count := utf8.RuneCountInString(candidate)
		if used+count > budget {
			m.Truncated = true
			return false
		}
		b.WriteString(candidate)
		used += count
		return true
	}
	finish := func() string { return strings.TrimSuffix(b.String(), "\n") }

	if !appendLine("REPO CONTEXT") {
		m.Truncated = true
		runes := []rune("REPO CONTEXT")
		if len(runes) > budget {
			runes = runes[:budget]
		}
		return string(runes)
	}
	if m.Summary != "" && !appendLine(m.Summary) {
		return finish()
	}

	appendSection := func(title string, lines []string) bool {
		if len(lines) == 0 {
			return true
		}
		if !appendLine(title) {
			return false
		}
		for _, line := range lines {
			if !appendLine("- " + line) {
				return false
			}
		}
		return true
	}

	files := filterSensitivePaths(m.Files)
	if !appendSection("Relevant files", files) {
		return finish()
	}

	symbolLines := make([]string, 0, len(m.Symbols))
	for _, symbol := range m.Symbols {
		if isSensitiveRepoPath(symbol.Path) {
			continue
		}
		name := symbol.Name
		if symbol.Owner != "" {
			name = symbol.Owner + "#" + symbol.Name
		}
		if symbol.Signature != "" && !strings.Contains(name, "(") {
			name += strings.TrimPrefix(symbol.Signature, symbol.Name)
		}
		symbolLines = append(symbolLines, fmt.Sprintf("%s:%d %s", symbol.Path, symbol.StartLine, name))
	}
	if !appendSection("Relevant symbols", symbolLines) {
		return finish()
	}

	byID := map[string]Symbol{}
	for _, symbol := range m.Symbols {
		byID[symbol.ID] = symbol
	}
	relationLines := make([]string, 0, len(m.Relations))
	for _, relation := range m.Relations {
		from, fromOK := byID[relation.From]
		to, toOK := byID[relation.To]
		if !fromOK || !toOK || isSensitiveRepoPath(from.Path) || isSensitiveRepoPath(to.Path) {
			continue
		}
		evidence := relation.Evidence
		if evidence == "" {
			evidence = string(relation.Kind)
		}
		relationLines = append(relationLines, fmt.Sprintf("%s -> %s [%s]", symbolLabel(from), symbolLabel(to), evidence))
	}
	if !appendSection("Confirmed relations", relationLines) {
		return finish()
	}

	unresolved := make([]string, 0, len(m.Unresolved))
	for _, item := range m.Unresolved {
		if !containsSensitiveRepoEvidence(item) {
			unresolved = append(unresolved, item)
		}
	}
	if !appendSection("Unresolved/ambiguous evidence", unresolved) {
		return finish()
	}
	if !appendSection("Changed files", filterSensitivePaths(m.ChangedFiles)) {
		return finish()
	}
	return finish()
}

func symbolLabel(symbol Symbol) string {
	if symbol.Owner != "" {
		return symbol.Owner + "#" + symbol.Name
	}
	return symbol.Name
}

func normalizeChangedFiles(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if normalized, ok := normalizeRelativePath(path); ok {
			out = append(out, normalized)
		}
	}
	return uniqueSorted(out)
}

func filterSensitivePaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if !isSensitiveRepoPath(path) {
			out = append(out, path)
		}
	}
	return out
}

func isSensitiveRepoPath(path string) bool {
	base := strings.ToLower(filepath.Base(normalizeSlash(path)))
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}
	switch strings.ToLower(filepath.Ext(base)) {
	case ".pem", ".key", ".p12", ".pfx", ".jks", ".keystore":
		return true
	}
	return false
}

func containsSensitiveRepoEvidence(evidence string) bool {
	lower := strings.ToLower(normalizeSlash(evidence))
	return strings.Contains(lower, "/.env") || strings.Contains(lower, ".env:") || strings.Contains(lower, ".pem") || strings.Contains(lower, ".key")
}
