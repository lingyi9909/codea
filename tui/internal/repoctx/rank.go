package repoctx

import (
	"path/filepath"
	"sort"
	"strings"
)

const ( scoreExactPath=1000; scoreExactSymbol=980; scoreChanged=900; scoreTypeImport=800; scoreGraphNeighbor=700; scoreLexical=400 )

func Rank(idx RepositoryIndex,q Query) []RankedItem {
	lower:=strings.ToLower(normalizeSlash(q.Text)); terms:=lexicalTerms(lower); termSet:=map[string]bool{}; for _,t:=range terms{termSet[t]=true}
	changed:=map[string]bool{}; for _,p:=range q.ChangedFiles{if n,ok:=normalizeRelativePath(p);ok{changed[n]=true}else{changed[normalizeSlash(p)]=true}}
	exactSeeds:=map[string]bool{}; for _,s:=range idx.Symbols{if queryExactSymbol(termSet,lower,s){exactSeeds[s.ID]=true}}
	neighbors:=map[string]bool{}; for _,r:=range idx.Relations{if exactSeeds[r.From]{neighbors[r.To]=true};if exactSeeds[r.To]{neighbors[r.From]=true}}
	byPath:=map[string]RankedItem{}; files:=map[string]SourceFile{}; for _,f:=range idx.Files{files[f.Path]=f}; syms:=map[string][]Symbol{}; for _,s:=range idx.Symbols{syms[s.Path]=append(syms[s.Path],s)}
	for path,f:=range files { best:=RankedItem{Path:path}; set:=func(score int,reason string,s *Symbol){if score>best.Score{best.Score=score;best.Reason=reason;best.Symbol=s}}; if strings.Contains(lower,strings.ToLower(normalizeSlash(path))){set(scoreExactPath,"exact-path",nil)}; for si:=range syms[path]{s:=&syms[path][si];if queryExactSymbol(termSet,lower,*s){set(scoreExactSymbol,"exact-symbol",s)}}; if changed[path]{set(scoreChanged,"changed-file",nil)}; if fileTypeImportMatch(f,termSet){set(scoreTypeImport,"type-import",nil)}; for si:=range syms[path]{s:=&syms[path][si];if neighbors[s.ID]{set(scoreGraphNeighbor,"graph-neighbor",s)}}; if fileLexicalMatch(f,syms[path],termSet){set(scoreLexical,"lexical",nil)}; if best.Score>0{byPath[path]=best} }
	out:=make([]RankedItem,0,len(byPath));for _,v:=range byPath{out=append(out,v)};sort.Slice(out,func(i,j int)bool{if out[i].Score!=out[j].Score{return out[i].Score>out[j].Score};if out[i].Path!=out[j].Path{return out[i].Path<out[j].Path};a,b:="","";if out[i].Symbol!=nil{a=out[i].Symbol.ID};if out[j].Symbol!=nil{b=out[j].Symbol.ID};return a<b});return out
}
func queryExactSymbol(terms map[string]bool,lower string,s Symbol)bool{n:=strings.ToLower(s.Name);if terms[n]{return true};if s.Owner!=""&&strings.Contains(lower,strings.ToLower(s.Owner+"#"+s.Name)){return true};return false}
func fileTypeImportMatch(f SourceFile,terms map[string]bool)bool{for _,imp:=range f.Imports{base:=strings.TrimSuffix(filepath.Base(strings.ReplaceAll(imp,".","/")),filepath.Ext(imp));if terms[strings.ToLower(base)]{return true}};return false}
func fileLexicalMatch(f SourceFile,syms []Symbol,terms map[string]bool)bool{if len(terms)==0{return false};pool:=append([]string{},f.Terms...);pool=append(pool,strings.FieldsFunc(strings.ToLower(normalizeSlash(f.Path)),func(r rune)bool{return r=='/'||r=='.'||r=='-'})...);for _,s:=range syms{pool=append(pool,strings.ToLower(s.Name),strings.ToLower(s.Owner),strings.ToLower(s.Package))};for _,x:=range pool{xl:=strings.ToLower(x);for t:=range terms{if t!=""&&(xl==t||strings.Contains(xl,t)){return true}}};return false}
