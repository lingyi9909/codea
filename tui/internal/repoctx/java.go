package repoctx

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type javaToken struct {
	text string
	line int
}

type relationCandidate struct {
	from         string
	kind         RelationKind
	targetType   string
	targetMethod string
	receiver     string
	confidence   float64
	evidence     string
}

func ExtractJava(path string, source []byte) SourceFile {
	norm, ok := normalizeRelativePath(path)
	if !ok {
		norm = normalizeSlash(path)
	}
	f := SourceFile{Path: norm, Extension: ".java"}
	toks := lexJava(string(source))
	f.Package = javaQualifiedAfter(toks, "package")
	f.Imports = javaAllQualifiedAfter(toks, "import")
	typeDecls := javaTypeDecls(toks)
	for _, d := range typeDecls {
		kind := SymbolClass
		switch d.kind {
		case "interface":
			kind = SymbolInterface
		case "enum":
			kind = SymbolEnum
		case "record":
			kind = SymbolRecord
		}
		anns := javaAnnotationsBefore(toks, d.keywordIndex)
		sym := Symbol{ID: stableSymbolID(norm, d.name), Name: d.name, Kind: kind, Path: norm, StartLine: toks[d.keywordIndex].line, EndLine: toks[d.closeIndex].line, Package: f.Package, Role: javaRole(anns), Annotations: anns}
		f.Symbols = append(f.Symbols, sym)
		javaExtractMembers(&f, toks, d, sym)
	}
	sort.Slice(f.Symbols, func(i, j int) bool {
		if f.Symbols[i].StartLine != f.Symbols[j].StartLine {
			return f.Symbols[i].StartLine < f.Symbols[j].StartLine
		}
		return f.Symbols[i].ID < f.Symbols[j].ID
	})
	sort.Strings(f.Imports)
	sort.Strings(f.Unresolved)
	return f
}

func stableSymbolID(path, identity string) string { return normalizeSlash(path) + "::" + identity }

type javaTypeDecl struct {
	kind, name                          string
	keywordIndex, openIndex, closeIndex int
}

func javaTypeDecls(t []javaToken) []javaTypeDecl {
	out := []javaTypeDecl{}
	for i := 0; i < len(t)-1; i++ {
		if t[i].text != "class" && t[i].text != "interface" && t[i].text != "enum" && t[i].text != "record" {
			continue
		}
		if !isJavaIdent(t[i+1].text) {
			continue
		}
		open := -1
		for j := i + 2; j < len(t); j++ {
			if t[j].text == "{" {
				open = j
				break
			}
			if t[j].text == ";" {
				break
			}
		}
		if open < 0 {
			continue
		}
		close := matchJava(t, open, "{", "}")
		if close < 0 {
			close = len(t) - 1
		}
		out = append(out, javaTypeDecl{t[i].text, t[i+1].text, i, open, close})
	}
	return out
}

func javaExtractMembers(f *SourceFile, t []javaToken, d javaTypeDecl, owner Symbol) {
	fieldTypes := map[string]string{}
	constructorCount := javaConstructorCount(t, d)
	depth := 0
	memberStart := d.openIndex + 1
	for i := d.openIndex + 1; i < d.closeIndex; i++ {
		switch t[i].text {
		case "{":
			depth++
		case "}":
			depth--
		}
		if depth != 0 {
			continue
		}
		if t[i].text == ";" {
			javaParseField(f, t, memberStart, i, owner, fieldTypes)
			memberStart = i + 1
			continue
		}
		if t[i].text != "(" {
			continue
		}
		nameIdx := i - 1
		if nameIdx < memberStart || !isJavaIdent(t[nameIdx].text) || javaControlWord(t[nameIdx].text) {
			continue
		}
		closeParen := matchJava(t, i, "(", ")")
		if closeParen < 0 || closeParen >= d.closeIndex {
			continue
		}
		next := closeParen + 1
		for next < d.closeIndex && (t[next].text == "throws" || isJavaIdent(t[next].text) || t[next].text == "," || t[next].text == ".") {
			next++
		}
		if next >= d.closeIndex || (t[next].text != "{" && t[next].text != ";") {
			continue
		}
		anns := javaAnnotationsInRange(t, memberStart, nameIdx)
		params, paramVars := javaParams(t, i+1, closeParen)
		name := t[nameIdx].text
		kind := SymbolMethod
		typ := ""
		signature := name + "(" + strings.Join(params, ",") + ")"
		identity := d.name + "#" + signature
		if name == d.name {
			kind = SymbolConstructor
			identity = d.name + "#<init>(" + strings.Join(params, ",") + ")"
		} else {
			typ = javaReturnType(t, memberStart, nameIdx)
		}
		endLine := t[closeParen].line
		bodyClose := next
		if t[next].text == "{" {
			bodyClose = matchJava(t, next, "{", "}")
			if bodyClose < 0 {
				bodyClose = next
			}
			endLine = t[bodyClose].line
		}
		sym := Symbol{ID: stableSymbolID(f.Path, identity), Name: name, Kind: kind, Path: f.Path, StartLine: t[nameIdx].line, EndLine: endLine, Package: f.Package, Owner: d.name, Type: typ, Signature: signature, Annotations: anns}
		f.Symbols = append(f.Symbols, sym)
		if kind == SymbolConstructor {
			if evidence, confirmed := javaConstructorDIEvidence(owner, anns, constructorCount); confirmed {
				for _, ptype := range params {
					if ptype != "" && !javaPrimitive(ptype) {
						f.candidates = append(f.candidates, relationCandidate{from: owner.ID, kind: RelationInjects, targetType: ptype, confidence: 1, evidence: evidence})
					}
				}
			}
		}
		if t[next].text == "{" {
			vars := map[string]string{}
			for k, v := range fieldTypes {
				vars[k] = v
			}
			for k, v := range paramVars {
				vars[k] = v
			}
			javaCalls(f, t, next+1, bodyClose, sym, vars)
			i = bodyClose
			memberStart = i + 1
		} else {
			i = closeParen
			memberStart = i + 1
		}
	}
}

func javaConstructorCount(t []javaToken, d javaTypeDecl) int {
	count := 0
	depth := 0
	for i := d.openIndex + 1; i < d.closeIndex; i++ {
		switch t[i].text {
		case "{":
			depth++
			continue
		case "}":
			depth--
			continue
		}
		if depth != 0 || t[i].text != "(" || i <= d.openIndex+1 {
			continue
		}
		nameIdx := i - 1
		if t[nameIdx].text != d.name || (nameIdx > d.openIndex && t[nameIdx-1].text == "@") {
			continue
		}
		closeParen := matchJava(t, i, "(", ")")
		if closeParen < 0 || closeParen >= d.closeIndex {
			continue
		}
		next := closeParen + 1
		for next < d.closeIndex && (t[next].text == "throws" || isJavaIdent(t[next].text) || t[next].text == "," || t[next].text == ".") {
			next++
		}
		if next < d.closeIndex && (t[next].text == "{" || t[next].text == ";") {
			count++
		}
	}
	return count
}

func javaConstructorDIEvidence(owner Symbol, anns []string, constructorCount int) (string, bool) {
	if containsString(anns, "Autowired") {
		return "@Autowired constructor", true
	}
	if containsString(anns, "Inject") {
		return "@Inject constructor", true
	}
	if owner.Role != "" && constructorCount == 1 {
		return "single constructor on Spring component", true
	}
	return "", false
}

func javaParseField(f *SourceFile, t []javaToken, start, end int, owner Symbol, fieldTypes map[string]string) {
	if start >= end {
		return
	}
	for i := start; i < end; i++ {
		if t[i].text == "(" {
			return
		}
	}
	anns := javaAnnotationsInRange(t, start, end)
	typ, name, ok := javaDeclaredTypeAndName(t, start, end)
	if !ok {
		return
	}
	fieldTypes[name] = typ
	if containsString(anns, "Autowired") {
		f.candidates = append(f.candidates, relationCandidate{from: owner.ID, kind: RelationInjects, targetType: typ, confidence: 1, evidence: "@Autowired field"})
	}
}

func javaCalls(f *SourceFile, t []javaToken, start, end int, from Symbol, vars map[string]string) {
	for i := start; i+3 < end; i++ {
		if !isJavaIdent(t[i].text) || t[i+1].text != "." || !isJavaIdent(t[i+2].text) || t[i+3].text != "(" {
			continue
		}
		recv, method := t[i].text, t[i+2].text
		if typ, ok := vars[recv]; ok {
			f.candidates = append(f.candidates, relationCandidate{from: from.ID, kind: RelationCalls, targetType: typ, targetMethod: method, receiver: recv, confidence: 1, evidence: "typed receiver " + recv})
		} else if recv != "this" && recv != "super" && recv != "String" {
			f.Unresolved = append(f.Unresolved, fmt.Sprintf("%s.%s receiver type unresolved at %s:%d", recv, method, f.Path, t[i].line))
		}
	}
}

func javaParams(t []javaToken, start, end int) ([]string, map[string]string) {
	types := []string{}
	vars := map[string]string{}
	seg := start
	depth := 0
	parse := func(a, b int) {
		typ, name, ok := javaDeclaredTypeAndName(t, a, b)
		if !ok {
			return
		}
		types = append(types, typ)
		vars[name] = typ
	}
	for i := start; i < end; i++ {
		switch t[i].text {
		case "<", "(", "[":
			depth++
		case ">", ")", "]":
			if depth > 0 {
				depth--
			}
		case ",":
			if depth == 0 {
				parse(seg, i)
				seg = i + 1
			}
		}
	}
	if seg < end {
		parse(seg, end)
	}
	return types, vars
}

func javaDeclaredTypeAndName(t []javaToken, start, end int) (string, string, bool) {
	if start >= end {
		return "", "", false
	}
	nameIdx := -1
	angleDepth := 0
	for i := end - 1; i >= start; i-- {
		switch t[i].text {
		case ">":
			angleDepth++
			continue
		case "<":
			if angleDepth > 0 {
				angleDepth--
			}
			continue
		}
		if angleDepth == 0 && isJavaIdent(t[i].text) && !javaModifier(t[i].text) && !javaAnnotationNameAt(t, i) {
			nameIdx = i
			break
		}
	}
	if nameIdx < 0 {
		return "", "", false
	}

	typeStart := javaSkipLeadingDeclarationDecorators(t, start, nameIdx)
	if typeStart >= nameIdx {
		return "", "", false
	}
	typeEnd := nameIdx
	depth := 0
	for i := typeStart; i < nameIdx; i++ {
		if t[i].text == "<" {
			if depth == 0 {
				typeEnd = i
				break
			}
			depth++
		}
	}
	var b strings.Builder
	for i := typeStart; i < typeEnd; i++ {
		if javaModifier(t[i].text) {
			continue
		}
		if isJavaIdent(t[i].text) || t[i].text == "." {
			b.WriteString(t[i].text)
		}
	}
	typ := strings.Trim(b.String(), ".")
	if typ == "" {
		return "", "", false
	}
	return typ, t[nameIdx].text, true
}

func javaSkipLeadingDeclarationDecorators(t []javaToken, start, end int) int {
	i := start
	for i < end {
		if javaModifier(t[i].text) {
			i++
			continue
		}
		if t[i].text != "@" {
			break
		}
		i++
		if i < end && isJavaIdent(t[i].text) {
			i++
			for i+1 < end && t[i].text == "." && isJavaIdent(t[i+1].text) {
				i += 2
			}
		}
		if i < end && t[i].text == "(" {
			if close := matchJava(t, i, "(", ")"); close >= 0 && close < end {
				i = close + 1
			}
		}
	}
	return i
}

func javaReturnType(t []javaToken, start, nameIdx int) string {
	for i := nameIdx - 1; i >= start; i-- {
		if isJavaIdent(t[i].text) && !javaModifier(t[i].text) && !javaAnnotationNameAt(t, i) {
			return t[i].text
		}
	}
	return ""
}

func javaQualifiedAfter(t []javaToken, keyword string) string {
	for i := 0; i < len(t); i++ {
		if t[i].text == keyword {
			var b strings.Builder
			for j := i + 1; j < len(t) && t[j].text != ";"; j++ {
				if t[j].text == "static" {
					continue
				}
				b.WriteString(t[j].text)
			}
			return b.String()
		}
	}
	return ""
}

func javaAllQualifiedAfter(t []javaToken, keyword string) []string {
	out := []string{}
	for i := 0; i < len(t); i++ {
		if t[i].text == keyword {
			var b strings.Builder
			for j := i + 1; j < len(t) && t[j].text != ";"; j++ {
				if t[j].text == "static" {
					continue
				}
				b.WriteString(t[j].text)
			}
			if b.Len() > 0 {
				out = append(out, b.String())
			}
		}
	}
	return out
}

func javaAnnotationsBefore(t []javaToken, idx int) []string {
	start := idx - 1
	for start >= 0 && t[start].text != ";" && t[start].text != "{" && t[start].text != "}" {
		start--
	}
	return javaAnnotationsInRange(t, start+1, idx)
}

func javaAnnotationsInRange(t []javaToken, start, end int) []string {
	out := []string{}
	for i := start; i+1 < end; i++ {
		if t[i].text == "@" && isJavaIdent(t[i+1].text) {
			out = append(out, t[i+1].text)
		}
	}
	return uniqueStrings(out)
}

func javaAnnotationNameAt(t []javaToken, i int) bool { return i > 0 && t[i-1].text == "@" }

func javaRole(anns []string) string {
	for _, p := range []struct{ a, r string }{{"RestController", "rest-controller"}, {"Controller", "controller"}, {"Service", "service"}, {"Repository", "repository"}, {"Component", "component"}} {
		if containsString(anns, p.a) {
			return p.r
		}
	}
	return ""
}

func matchJava(t []javaToken, open int, left, right string) int {
	depth := 0
	for i := open; i < len(t); i++ {
		if t[i].text == left {
			depth++
		}
		if t[i].text == right {
			depth--
			if depth == 0 {
				return i
			}
		}
	return -1
}

func containsString(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func uniqueStrings(xs []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func javaControlWord(s string) bool {
	switch s {
	case "if", "for", "while", "switch", "catch", "new", "return", "throw", "synchronized":
		return true
	}
	return false
}

func javaModifier(s string) bool {
	switch s {
	case "public", "protected", "private", "static", "final", "abstract", "native", "synchronized", "default", "transient", "volatile", "strictfp", "sealed", "non-sealed", "var":
		return true
	}
	return false
}

func javaPrimitive(s string) bool {
	switch s {
	case "byte", "short", "int", "long", "float", "double", "boolean", "char", "void", "String":
		return true
	}
	return false
}

func isJavaIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range []rune(s) {
		if i == 0 {
			if !(unicode.IsLetter(r) || r == '_' || r == '$') {
				return false
			}
		} else if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$') {
			return false
		}
	}
	return true
}

func lexJava(src string) []javaToken {
	out := []javaToken{}
	line := 1
	for i := 0; i < len(src); {
		c := src[i]
		if c == '\n' {
			line++
			i++
			continue
		}
		if unicode.IsSpace(rune(c)) {
			i++
			continue
		}
		if i+1 < len(src) && src[i:i+2] == "//" {
			i += 2
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(src) && src[i:i+2] == "/*" {
			i += 2
			for i+1 < len(src) && src[i:i+2] != "*/" {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			if i+1 < len(src) {
				i += 2
			}
			continue
		}
		if i+2 < len(src) && src[i:i+3] == "\"\"\"" {
			i += 3
			for i+2 < len(src) && src[i:i+3] != "\"\"\"" {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			if i+2 < len(src) {
				i += 3
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote := c
			i++
			for i < len(src) {
				if src[i] == '\n' {
					line++
				}
				if src[i] == '\\' {
					i += 2
					continue
				}
				if i < len(src) && src[i] == quote {
					i++
					break
				}
				i++
			}
			continue
		}
		r := rune(c)
		if unicode.IsLetter(r) || c == '_' || c == '$' {
			start := i
			i++
			for i < len(src) {
				rr := rune(src[i])
				if unicode.IsLetter(rr) || unicode.IsDigit(rr) || src[i] == '_' || src[i] == '$' {
					i++
				} else {
					break
				}
			}
			out = append(out, javaToken{src[start:i], line})
			continue
		}
		out = append(out, javaToken{string(c), line})
		i++
	}
	return out
}
