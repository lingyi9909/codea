package repoctx

import (
	"context"
	"fmt"
	"os"
	"sort"
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
	if err != nil { return RepositoryIndex{}, err }
	idx := RepositoryIndex{}
	for _, wf := range walked {
		data, err := os.ReadFile(wf.AbsPath)
		if err != nil { idx.Unresolved = append(idx.Unresolved, fmt.Sprintf("%s: read: %v", wf.Path, err)); continue }
		var f SourceFile
		switch wf.Extension { case ".java": f = ExtractJava(wf.Path, data); case ".go": f = ExtractGo(wf.Path, data); default: f = ExtractFallback(wf.Path, data) }
		f.AbsPath = wf.AbsPath
		idx.Files = append(idx.Files, f); idx.Symbols = append(idx.Symbols, f.Symbols...); idx.Unresolved = append(idx.Unresolved, f.Unresolved...); idx.candidates = append(idx.candidates, f.candidates...)
	}
	idx.resolveCandidates()
	sort.Slice(idx.Files, func(a,b int) bool { return idx.Files[a].Path < idx.Files[b].Path })
	sort.Slice(idx.Symbols, func(a,b int) bool { return idx.Symbols[a].ID < idx.Symbols[b].ID })
	sort.Slice(idx.Relations, func(a,b int) bool { ra,rb:=idx.Relations[a],idx.Relations[b]; if ra.From!=rb.From{return ra.From<rb.From}; if ra.To!=rb.To{return ra.To<rb.To}; return ra.Kind<rb.Kind })
	idx.Unresolved = uniqueSorted(idx.Unresolved)
	return idx,nil
}

func (idx *RepositoryIndex) resolveCandidates() {
	types:=map[string][]Symbol{}; methods:=map[string][]Symbol{}
	for _,s:=range idx.Symbols {
		if s.Owner=="" && (s.Kind==SymbolClass||s.Kind==SymbolInterface||s.Kind==SymbolEnum||s.Kind==SymbolRecord||s.Kind==SymbolType) { types[s.Name]=append(types[s.Name],s) }
		if s.Owner!="" && (s.Kind==SymbolMethod||s.Kind==SymbolFunction) { methods[s.Owner+"\x00"+s.Name]=append(methods[s.Owner+"\x00"+s.Name],s) }
	}
	for _,c:=range idx.candidates {
		targetType:=simpleTypeName(c.targetType)
		if c.kind==RelationInjects { matches:=types[targetType]; if len(matches)==1 { idx.Relations=append(idx.Relations,Relation{From:c.from,To:matches[0].ID,Kind:c.kind,Confidence:c.confidence,Evidence:c.evidence}) } else { idx.Unresolved=append(idx.Unresolved,resolutionMessage(c,len(matches))) }; continue }
		if c.kind==RelationCalls { matches:=methods[targetType+"\x00"+c.targetMethod]; if len(matches)==1 { idx.Relations=append(idx.Relations,Relation{From:c.from,To:matches[0].ID,Kind:c.kind,Confidence:c.confidence,Evidence:c.evidence}) } else { idx.Unresolved=append(idx.Unresolved,resolutionMessage(c,len(matches))) } }
	}
}
func resolutionMessage(c relationCandidate,n int)string { state:="unresolved"; if n>1{state="ambiguous"}; target:=simpleTypeName(c.targetType); if c.targetMethod!=""{target+="#"+c.targetMethod}; return fmt.Sprintf("%s relation %s -> %s (%d matches)",state,c.from,target,n) }
func simpleTypeName(s string)string { s=strings.TrimSpace(s); if i:=strings.LastIndex(s,".");i>=0{s=s[i+1:]}; return strings.Trim(s,"[]*") }
func uniqueSorted(xs []string)[]string { m:=map[string]struct{}{}; for _,x:=range xs{if strings.TrimSpace(x)!=""{m[x]=struct{}{}}}; out:=make([]string,0,len(m)); for x:=range m{out=append(out,x)}; sort.Strings(out); return out }
