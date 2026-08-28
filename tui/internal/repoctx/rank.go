package repoctx

import (
	"path/filepath"
	"sort"
	"strings"
)

const (
	scoreExactPath     = 1000
	scoreExactSymbol   = 980
	scoreChanged       = 900
	scoreTypeImport    = 800
	scoreGraphNeighbor = 700
	scoreLexical       = 400
)

func Rank(idx RepositoryIndex, q Query) []RankedItem {
	lower := strings.ToLower(normalizeSlash(q.Text))
	terms := lexicalTerms(lower)
	termSet := map[string]bool{}
	for _, term := range terms {
		termSet[term] = true
	}

	changed := map[string]bool{}
	for _, path := range q.ChangedFiles {
		if normalized, ok := normalizeRelativePath(path); ok {
			changed[normalized] = true
		} else {
			changed[normalizeSlash(path)] = true
		}
	}

	exactSeeds := map[string]bool{}
	for _, symbol := range idx.Symbols {
		if queryExactSymbol(termSet, lower, symbol) {
			exactSeeds[symbol.ID] = true
		}
	}

	neighbors := map[string]bool{}
	for _, relation := range idx.Relations {
		if exactSeeds[relation.From] {
			neighbors[relation.To] = true
		}
		if exactSeeds[relation.To] {
			neighbors[relation.From] = true
		}
	}

	files := map[string]SourceFile{}
	for _, file := range idx.Files {
		files[file.Path] = file
	}
	symbolsByPath := map[string][]Symbol{}
	for _, symbol := range idx.Symbols {
		symbolsByPath[symbol.Path] = append(symbolsByPath[symbol.Path], symbol)
	}

	byPath := map[string]RankedItem{}
	for path, file := range files {
		best := RankedItem{Path: path}
		set := func(score int, reason string, symbol *Symbol) {
			if score > best.Score {
				best.Score = score
				best.Reason = reason
				best.Symbol = symbol
			}
		}

		if strings.Contains(lower, strings.ToLower(normalizeSlash(path))) {
			set(scoreExactPath, "exact-path", nil)
		}
		for i := range symbolsByPath[path] {
			symbol := &symbolsByPath[path][i]
			if queryExactSymbol(termSet, lower, *symbol) {
				set(scoreExactSymbol, "exact-symbol", symbol)
			}
		}
		if changed[path] {
			set(scoreChanged, "changed-file", nil)
		}
		if fileTypeImportMatch(file, symbolsByPath[path], termSet) {
			set(scoreTypeImport, "type-import", nil)
		}
		for i := range symbolsByPath[path] {
			symbol := &symbolsByPath[path][i]
			if neighbors[symbol.ID] {
				set(scoreGraphNeighbor, "graph-neighbor", symbol)
			}
		}
		if fileLexicalMatch(file, symbolsByPath[path], termSet) {
			set(scoreLexical, "lexical", nil)
		}
		if best.Score > 0 {
			byPath[path] = best
		}
	}

	out := make([]RankedItem, 0, len(byPath))
	for _, item := range byPath {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		a, b := "", ""
		if out[i].Symbol != nil {
			a = out[i].Symbol.ID
		}
		if out[j].Symbol != nil {
			b = out[j].Symbol.ID
		}
		return a < b
	})
	return out
}

func queryExactSymbol(terms map[string]bool, lower string, symbol Symbol) bool {
	name := strings.ToLower(symbol.Name)
	if terms[name] {
		return true
	}
	return symbol.Owner != "" && strings.Contains(lower, strings.ToLower(symbol.Owner+"#"+symbol.Name))
}

func fileTypeImportMatch(file SourceFile, symbols []Symbol, terms map[string]bool) bool {
	for _, imp := range file.Imports {
		base := strings.TrimSuffix(filepath.Base(strings.ReplaceAll(imp, ".", "/")), filepath.Ext(imp))
		if terms[strings.ToLower(base)] {
			return true
		}
	}
	for _, symbol := range symbols {
		if symbol.Type != "" && terms[strings.ToLower(simpleTypeName(symbol.Type))] {
			return true
		}
		for _, ref := range symbol.TypeRefs {
			if terms[strings.ToLower(simpleTypeName(ref))] {
				return true
			}
		}
	}
	return false
}

func fileLexicalMatch(file SourceFile, symbols []Symbol, terms map[string]bool) bool {
	if len(terms) == 0 {
		return false
	}
	pool := append([]string{}, file.Terms...)
	pool = append(pool, strings.FieldsFunc(strings.ToLower(normalizeSlash(file.Path)), func(r rune) bool {
		return r == '/' || r == '.' || r == '-'
	})...)
	for _, symbol := range symbols {
		pool = append(pool, strings.ToLower(symbol.Name), strings.ToLower(symbol.Owner), strings.ToLower(symbol.Package))
	}
	for _, value := range pool {
		lower := strings.ToLower(value)
		for term := range terms {
			if term != "" && (lower == term || strings.Contains(lower, term)) {
				return true
			}
		}
	}
	return false
}
