package repoctx

import "strings"

type SymbolKind string

const (
	SymbolClass       SymbolKind = "class"
	SymbolInterface   SymbolKind = "interface"
	SymbolEnum        SymbolKind = "enum"
	SymbolRecord      SymbolKind = "record"
	SymbolMethod      SymbolKind = "method"
	SymbolConstructor SymbolKind = "constructor"
	SymbolFunction    SymbolKind = "function"
	SymbolType        SymbolKind = "type"
	SymbolField       SymbolKind = "field"
)

type RelationKind string

const (
	RelationCalls      RelationKind = "calls"
	RelationInjects    RelationKind = "injects"
	RelationReferences RelationKind = "references"
	RelationImports    RelationKind = "imports"
)

type Symbol struct {
	ID          string
	Name        string
	Kind        SymbolKind
	Path        string
	StartLine   int
	EndLine     int
	Package     string
	Role        string
	Signature   string
	Owner       string
	Type        string
	Annotations []string
	TypeRefs    []string
}

type Relation struct {
	From       string
	To         string
	Kind       RelationKind
	Confidence float64
	Evidence   string
}

type SourceFile struct {
	Path       string
	AbsPath    string
	Extension  string
	Package    string
	Imports    []string
	Terms      []string
	Symbols    []Symbol
	Relations  []Relation
	Unresolved []string
}

type RepositoryIndex struct {
	Files      []SourceFile
	Symbols    []Symbol
	Relations  []Relation
	Unresolved []string
}

type Query struct {
	Text         string
	ChangedFiles []string
	MaxChars     int
}

type RepoMap struct {
	Summary      string
	Files        []string
	Symbols      []Symbol
	Relations    []Relation
	Unresolved   []string
	ChangedFiles []string
	Truncated    bool
	maxChars     int
}

type RankedItem struct {
	Path   string
	Symbol *Symbol
	Score  int
	Reason string
}

func normalizeSlash(s string) string {
	return strings.ReplaceAll(s, "\\", "/")
}
