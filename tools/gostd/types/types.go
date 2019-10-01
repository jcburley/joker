package types

import (
	"fmt"
	. "github.com/candid82/joker/tools/gostd/utils"
	. "go/ast"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
)

type TypeInfo struct {
	Type             Expr   // nil until first reference (not definition) seen
	FullName         string // Clojure name (e.g. "a.b.c/Typename")
	LocalName        string // Local, or base, name (e.g. "Typename")
	Definition       *TypeDefInfo
	SimpleIdentifier bool // Just a name, not *name, []name, etc.
}

// Maps the "definitive" (first-found) referencing Expr for a type to type info
var types = map[Expr]*TypeInfo{}

// Maps a non-definitive referencing Expr for a type to the definitive referencing Expr for the same type
var typeAliases = map[Expr]Expr{}

// Maps the full (Clojure) name (e.g. "a.b.c/typename") for a type to the definitive Expr for the same type
var typesByFullName = map[string]Expr{}

// Info from the definition of the type (if any)
type TypeDefInfo struct {
	Doc    string
	DefPos token.Pos
}

var typeDefinitionsByFullName = map[string]*TypeDefInfo{}

func TypeDefine(ts *TypeSpec, parentDoc *CommentGroup) *TypeDefInfo {
	prefix := ClojureNamespaceForPos(Fset.Position(ts.Name.NamePos)) + "/"
	tln := ts.Name.Name
	tfn := prefix + tln
	if tdi, ok := typeDefinitionsByFullName[tfn]; ok {
		panic(fmt.Sprintf("already defined type %s at %s and again at %s", tfn, WhereAt(tdi.DefPos), WhereAt(ts.Name.NamePos)))
	}

	doc := ts.Doc // Try block comments for this specific decl
	if doc == nil {
		doc = ts.Comment // Use line comments if no preceding block comments are available
	}
	if doc == nil {
		doc = parentDoc // Use 'var'/'const' statement block comments as last resort
	}

	tdi := &TypeDefInfo{
		Doc:    CommentGroupAsString(doc),
		DefPos: ts.Name.NamePos,
	}
	typeDefinitionsByFullName[tfn] = tdi
	if e, ok := typesByFullName[tfn]; ok {
		types[e].Definition = tdi
	}
	return tdi
}

func TypeLookup(e Expr) *TypeInfo {
	if ti, ok := types[e]; ok {
		return ti
	}
	if ta, ok := typeAliases[e]; ok {
		return types[ta]
	}
	tfn, tln, simple := typeNames(e, true)
	if te, ok := typesByFullName[tfn]; ok {
		typeAliases[te] = e
		return types[te]
	}
	ti := &TypeInfo{
		Type:             e,
		FullName:         tfn,
		LocalName:        tln,
		SimpleIdentifier: simple,
	}
	types[e] = ti
	typesByFullName[tfn] = e
	if tdi, ok := typeDefinitionsByFullName[tfn]; ok {
		ti.Definition = tdi
	}
	return ti
}

func typeKeyForSort(k string) string {
	if strings.HasPrefix(k, "*") {
		return k[1:] + "*"
	}
	if strings.HasPrefix(k, "[]") {
		return k[2:] + "[]"
	}
	return k
}

func SortedTypes(m map[*TypeInfo]struct{}, f func(ti *TypeInfo)) {
	var keys []string
	for k, _ := range m {
		if k != nil {
			keys = append(keys, k.FullName)
		}
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return typeKeyForSort(keys[i]) < typeKeyForSort(keys[j])
	})
	for _, k := range keys {
		f(types[typesByFullName[k]])
	}
}

func typeNames(e Expr, root bool) (full, local string, simple bool) {
	prefix := ""
	if root {
		prefix = ClojureNamespaceForExpr(e) + "/"
	}
	switch x := e.(type) {
	case *Ident:
		full = prefix + x.Name
		local = x.Name
		simple = true
	case *ArrayType:
		elFull, elLocal, _ := typeNames(x.Elt, false)
		full = "[]" + prefix + elFull
		local = "[]" + elLocal
	case *StarExpr:
		elFull, elLocal, _ := typeNames(x.X, false)
		full = "*" + prefix + elFull
		local = "*" + elLocal
	}
	return
}

func (ti *TypeInfo) TypeReflected() string {
	t := ""
	suffix := ""
	prefix := "_" + filepath.Base(GoPackageForExpr(ti.Type)) + "."
	switch x := ti.Type.(type) {
	case *Ident:
		t = "*" + prefix + x.Name
		suffix = ".Elem()"
	case *StarExpr:
		t = "*" + prefix + x.X.(*Ident).Name
	default:
		panic(fmt.Sprintf("unrecognized expr %T", x))
	}
	return fmt.Sprintf("_reflect.TypeOf((%s)(nil))%s", t, suffix)
}

func (ti *TypeInfo) TypeKey() string {
	t := ""
	suffix := ""
	prefix := GoPackageForExpr(ti.Type)
	switch x := ti.Type.(type) {
	case *Ident:
		t = "*" + prefix + x.Name
		suffix = ".Elem()"
	case *StarExpr:
		t = "*" + prefix + x.X.(*Ident).Name
	default:
		panic(fmt.Sprintf("unrecognized expr %T", x))
	}
	return fmt.Sprintf("_reflect.TypeOf((%s)(nil))%s", t, suffix)
}

func (ti *TypeInfo) TypeMappingsName() string {
	res := ""
	switch x := ti.Type.(type) {
	case *Ident:
		res += x.Name
	case *ArrayType:
		res += "ArrayOf_" + x.Elt.(*Ident).Name
	case *StarExpr:
		res += "PtrTo_" + x.X.(*Ident).Name
	default:
		panic(fmt.Sprintf("typeName: unrecognized expr %T", x))
	}
	return "info_" + res
}
