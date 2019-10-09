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

var NumExprHits uint
var NumAliasHits uint
var NumFullNameHits uint

type TypeInfo struct {
	Type             Expr // nil until first reference (not definition) seen
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
	TypeSpec  *TypeSpec
	FullName  string // Clojure name (e.g. "a.b.c/Typename")
	LocalName string // Local, or base, name (e.g. "Typename")
	IsPrivate bool
	Doc       string
	DefPos    token.Pos
	goPackage string
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
		TypeSpec:  ts,
		FullName:  tfn,
		LocalName: tln,
		IsPrivate: IsPrivate(tln),
		Doc:       CommentGroupAsString(doc),
		DefPos:    ts.Name.NamePos,
		goPackage: GoPackageForTypeSpec(ts),
	}
	typeDefinitionsByFullName[tfn] = tdi
	return tdi
}

func TypeLookup(e Expr) *TypeInfo {
	if ti, ok := types[e]; ok {
		NumExprHits++
		return ti
	}
	if ta, ok := typeAliases[e]; ok {
		NumAliasHits++
		return types[ta]
	}
	tfn, _, simple := typeNames(e, true)
	if te, ok := typesByFullName[tfn]; ok {
		NumFullNameHits++
		typeAliases[te] = e
		return types[te]
	}
	ti := &TypeInfo{
		Type:             e,
		Definition:       typeDefinitionsByFullName[tfn],
		SimpleIdentifier: simple,
	}
	types[e] = ti
	typesByFullName[tfn] = e
	return ti
}

func (ti *TypeInfo) FullName() string {
	return ti.Definition.FullName
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

func SortedTypeDefinitions(m map[*TypeDefInfo]struct{}, f func(ti *TypeDefInfo)) {
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
		f(typeDefinitionsByFullName[k])
	}
}

func SortedTypes(m map[*TypeInfo]struct{}, f func(ti *TypeInfo)) {
	var keys []string
	for k, _ := range m {
		if k != nil {
			keys = append(keys, k.FullName())
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
		o := x.Obj
		if o != nil && o.Kind == Typ {
			tdi := typeDefinitionsByFullName[full]
			if o.Name != local || (tdi != nil && o.Decl.(*TypeSpec) != tdi.TypeSpec) {
				Print(Fset, x)
				var ts *TypeSpec
				if tdi != nil {
					ts = tdi.TypeSpec
				}
				panic(fmt.Sprintf("mismatch name=%s != %s or ts %p != %p!", o.Name, local, o.Decl.(*TypeSpec), ts))
			}
		} else {
			// Strangely, not all *Ident's referring to defined types have x.Obj populated! Can't figure out what's
			// different about them, though maybe it's just that they're for only those receivers currently being
			// code-generated?
		}
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

func (tdi *TypeDefInfo) TypeReflected() (packageImport, pattern string) {
	t := "_" + filepath.Base(tdi.goPackage) + "." + tdi.LocalName
	return "reflect", fmt.Sprintf("%%s.TypeOf((*%s)(nil)).Elem()", t)
}

// currently unused
func (ti *TypeInfo) typeKey() string {
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

func (tdi *TypeDefInfo) TypeMappingsName() string {
	if IsPrivate(tdi.LocalName) {
		return ""
	}
	res := "info_" + tdi.LocalName
	return res
}

func (ti *TypeInfo) TypeMappingsName() string {
	if IsPrivate(ti.Definition.LocalName) {
		return ""
	}
	res := "info_"
	switch x := ti.Type.(type) {
	case *Ident:
		res += x.Name
	case *ArrayType:
		res = ""
	case *StarExpr:
		res += "PtrTo_" + x.X.(*Ident).Name
	default:
		panic(fmt.Sprintf("unrecognized expr %T", x))
	}
	return res
}
