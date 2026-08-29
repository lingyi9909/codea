package repoctx

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	pathpkg "path"
	"sort"
	"strconv"
	"strings"
)

func ExtractGo(path string, source []byte) SourceFile {
	norm, ok := normalizeRelativePath(path)
	if !ok {
		norm = normalizeSlash(path)
	}
	out := SourceFile{Path: norm, Extension: ".go", ImportAliases: map[string]string{}}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, norm, source, parser.AllErrors)
	if err != nil {
		out.Unresolved = append(out.Unresolved, fmt.Sprintf("%s: Go parse: %v", norm, err))
	}
	if file == nil {
		return out
	}
	out.Package = file.Name.Name
	for _, imp := range file.Imports {
		p, e := strconv.Unquote(imp.Path.Value)
		if e != nil {
			continue
		}
		out.Imports = append(out.Imports, p)
		alias := pathpkg.Base(p)
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		if alias != "" && alias != "_" && alias != "." {
			out.ImportAliases[alias] = p
		}
	}
	sort.Strings(out.Imports)

	structFields := map[string]map[string]string{}
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			kind := SymbolType
			switch ts.Type.(type) {
			case *ast.InterfaceType:
				kind = SymbolInterface
			}
			sym := Symbol{ID: stableSymbolID(norm, ts.Name.Name), Name: ts.Name.Name, Kind: kind, Path: norm, StartLine: fset.Position(ts.Pos()).Line, EndLine: fset.Position(ts.End()).Line, Package: out.Package}
			out.Symbols = append(out.Symbols, sym)
			if st, ok := ts.Type.(*ast.StructType); ok {
				fields := map[string]string{}
				for _, fld := range st.Fields.List {
					typ := goExprName(fld.Type)
					for _, n := range fld.Names {
						if typ != "" {
							fields[n.Name] = typ
						}
				}
				structFields[ts.Name.Name] = fields
			}
		}
	}
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		kind := SymbolFunction
		owner := ""
		if fd.Recv != nil && len(fd.Recv.List) > 0 {
			kind = SymbolMethod
			owner = goExprName(fd.Recv.List[0].Type)
		}
		sig := goFuncSignature(fd)
		identity := fd.Name.Name + sig
		if owner != "" {
			identity = owner + "#" + fd.Name.Name + sig
		}
		sym := Symbol{ID: stableSymbolID(norm, identity), Name: fd.Name.Name, Kind: kind, Path: norm, StartLine: fset.Position(fd.Pos()).Line, EndLine: fset.Position(fd.End()).Line, Package: out.Package, Owner: owner, Signature: fd.Name.Name + sig}
		out.Symbols = append(out.Symbols, sym)
		if fd.Body != nil {
			goExtractCalls(&out, fd, sym, structFields)
		}
	}
	sort.Slice(out.Symbols, func(i, j int) bool {
		if out.Symbols[i].StartLine != out.Symbols[j].StartLine {
			return out.Symbols[i].StartLine < out.Symbols[j].StartLine
		}
		return out.Symbols[i].ID < out.Symbols[j].ID
	})
	sort.Strings(out.Unresolved)
	return out
}

func goExtractCalls(out *SourceFile, fd *ast.FuncDecl, from Symbol, structFields map[string]map[string]string) {
	vars := map[string]string{}
	if fd.Recv != nil {
		for _, r := range fd.Recv.List {
			typ := goExprName(r.Type)
			for _, n := range r.Names {
				vars[n.Name] = typ
			}
		}
	}
	if fd.Type.Params != nil {
		for _, p := range fd.Type.Params.List {
			typ := goExprName(p.Type)
			for _, n := range p.Names {
				vars[n.Name] = typ
			}
		}
	}
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.DeclStmt:
			gd, ok := x.Decl.(*ast.GenDecl)
			if ok {
				for _, sp := range gd.Specs {
					if vs, ok := sp.(*ast.ValueSpec); ok {
						typ := goExprName(vs.Type)
						for _, n := range vs.Names {
							if typ != "" {
								vars[n.Name] = typ
							}
						}
					}
				}
			}
		case *ast.AssignStmt:
			if x.Tok == token.DEFINE {
				for i, l := range x.Lhs {
					id, ok := l.(*ast.Ident)
					if !ok || i >= len(x.Rhs) {
						continue
					}
					if ce, ok := x.Rhs[i].(*ast.CompositeLit); ok {
						if typ := goExprName(ce.Type); typ != "" {
							vars[id.Name] = typ
						}
					}
				}
			}
		case *ast.CallExpr:
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			targetType := ""
			switch recv := sel.X.(type) {
			case *ast.Ident:
				targetType = vars[recv.Name]
			case *ast.SelectorExpr:
				if base, ok := recv.X.(*ast.Ident); ok {
					if ownerType := vars[base.Name]; ownerType != "" {
						if fields := structFields[ownerType]; fields != nil {
							targetType = fields[recv.Sel.Name]
						}
					}
			}
			if targetType != "" {
				out.candidates = append(out.candidates, relationCandidate{from: from.ID, kind: RelationCalls, targetType: targetType, targetMethod: sel.Sel.Name, confidence: 1, evidence: "Go AST typed selector"})
			}
		}
		return true
	})
}

func goExprName(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.StarExpr:
		return goExprName(x.X)
	case *ast.SelectorExpr:
		return goExprName(x.X) + "." + x.Sel.Name
	case *ast.ArrayType:
		return goExprName(x.Elt)
	case *ast.IndexExpr:
		return goExprName(x.X)
	case *ast.IndexListExpr:
		return goExprName(x.X)
	}
	return ""
}
func goFuncSignature(fd *ast.FuncDecl) string {
	parts := []string{}
	if fd.Type.Params != nil {
		for _, p := range fd.Type.Params.List {
			typ := goExprName(p.Type)
			count := len(p.Names)
			if count == 0 {
				count = 1
			}
			for i := 0; i < count; i++ {
				parts = append(parts, typ)
			}
		}
	}
	return "(" + strings.Join(parts, ",") + ")"
}
