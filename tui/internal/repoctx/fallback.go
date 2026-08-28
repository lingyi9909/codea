package repoctx

import (
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const fallbackMaxTerms = 256
const fallbackMaxTermLen = 64

func ExtractFallback(path string, source []byte) SourceFile {
	norm, ok := normalizeRelativePath(path)
	if !ok {
		norm = normalizeSlash(path)
	}
	f := SourceFile{Path: norm, Extension: strings.ToLower(filepath.Ext(norm))}
	seen := map[string]struct{}{}
	add := func(s string) {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" || len([]rune(s)) > fallbackMaxTermLen {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
	}
	for _, r := range lexicalTerms(string(source)) {
		add(r)
		if len(seen) >= fallbackMaxTerms {
			break
		}
	}
	for _, part := range strings.FieldsFunc(norm, func(r rune) bool { return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') }) {
		add(part)
	}
	f.Terms = make([]string, 0, len(seen))
	for term := range seen {
		f.Terms = append(f.Terms, term)
	}
	sort.Strings(f.Terms)
	if len(f.Terms) > fallbackMaxTerms {
		f.Terms = f.Terms[:fallbackMaxTerms]
	}
	return f
}
func lexicalTerms(s string) []string {
	out := []string{}
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
		}
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(unicode.ToLower(r))
		} else {
			flush()
		}
	}
	flush()
	return out
}
