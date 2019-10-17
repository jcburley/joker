package types

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/godb"
	. "github.com/candid82/joker/tools/gostd/godb"
	. "github.com/candid82/joker/tools/gostd/utils"
	. "go/ast"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
)

const Concrete = ^uint(0) /* MaxUint */

var NumExprHits uint
var NumAliasHits uint
var NumFullNameHits uint

// Info from the definition of the type (if any)
type Type struct {
	Type           Expr      // The actual type (if any)
	TypeSpec       *TypeSpec // The definition of the named type (if any)
	FullName       string    // Clojure name (e.g. "a.b.c/Typename")
	LocalName      string    // Local, or base, name (e.g. "Typename" or "*Typename")
	IsExported     bool
	Doc            string
	DefPos         token.Pos
	GoFile         *GoFile
	GoPrefix       string // Currently either "" or "*" (for reference types)
	GoPackage      string // E.g. a/b/c
	GoName         string // Base name of type (LocalName without any prefix)
	underlyingType *Type
	Ord            uint // Slot in []*GoTypeInfo and position of case statement in big switch in goswitch.go
	Specificity    uint // Concrete means concrete type; else # of methods defined for interface{} (abstract) type
}

var typesByFullName = map[string]*Type{}

func specificityOfInterface(ts *InterfaceType) uint {
	var sp uint
	for _, m := range ts.Methods.List {
		if m.Names != nil {
			sp += (uint)(len(m.Names))
			continue
		}
		ts := godb.Resolve(m.Type)
		if ts == nil {
			continue
		}
		sp += specificityOfInterface(ts.(*TypeSpec).Type.(*InterfaceType))
	}
	return sp
}

func specificity(ts *TypeSpec) uint {
	if iface, ok := ts.Type.(*InterfaceType); ok {
		return specificityOfInterface(iface)
	}
	return Concrete
}

func TypeDefine(ts *TypeSpec, gf *godb.GoFile, parentDoc *CommentGroup) []*Type {
	if len(allTypesSorted) > 0 {
		panic("Attempt to define new type after having sorted all types!!")
	}
	prefix := ClojureNamespaceForPos(Fset.Position(ts.Name.NamePos)) + "/"
	tln := ts.Name.Name
	tfn := prefix + tln
	if tdi, ok := typesByFullName[tfn]; ok {
		panic(fmt.Sprintf("already defined type %s at %s and again at %s", tfn, WhereAt(tdi.DefPos), WhereAt(ts.Name.NamePos)))
	}

	doc := ts.Doc // Try block comments for this specific decl
	if doc == nil {
		doc = ts.Comment // Use line comments if no preceding block comments are available
	}
	if doc == nil {
		doc = parentDoc // Use 'var'/'const' statement block comments as last resort
	}

	tdi := &Type{
		Type:        ts.Type,
		TypeSpec:    ts,
		FullName:    tfn,
		LocalName:   tln,
		IsExported:  IsExported(tln),
		Doc:         CommentGroupAsString(doc),
		DefPos:      ts.Name.NamePos,
		GoFile:      gf,
		GoPrefix:    "",
		GoPackage:   GoPackageForTypeSpec(ts),
		GoName:      tln,
		Specificity: specificity(ts),
	}
	if ts.Type != nil {
		tdiByExpr, found := typesByExpr[ts.Type]
		if found && tdiByExpr != tdi {
			panic(fmt.Sprintf("different expr for type %s", tfn))
		}
		typesByExpr[ts.Type] = tdi
	}
	typesByFullName[tfn] = tdi

	if tdi.Specificity == Concrete {
		// Concrete types all get reference-to versions.
		tfnPtr := "*" + tfn
		tdiPtr := &Type{
			Type:           &StarExpr{X: ts.Type},
			FullName:       tfnPtr,
			LocalName:      "*" + tln,
			IsExported:     tdi.IsExported,
			Doc:            "",
			GoPrefix:       "*",
			GoPackage:      tdi.GoPackage,
			GoName:         tln,
			underlyingType: tdi,
			Specificity:    Concrete,
		}
		tdiByExpr, found := typesByExpr[tdiPtr.Type]
		if found && tdiByExpr != tdiPtr {
			panic(fmt.Sprintf("different expr for type %s", tfn))
		}
		typesByExpr[ts.Type] = tdiPtr
		typesByFullName[tfnPtr] = tdiPtr
		return []*Type{tdi, tdiPtr}
	}

	return []*Type{tdi}
}

// Maps type-defining Expr to exactly one struct describing that type
var typesByExpr = map[Expr]*Type{}

func TypeLookup(e Expr) *Type {
	if tdi, ok := typesByExpr[e]; ok {
		NumExprHits++
		return tdi
	}
	tfn, _, _ := typeNames(e, true)
	if tdi, ok := typesByFullName[tfn]; ok {
		if tdi.Type == nil {
			panic(fmt.Sprintf("nil Type for %s", tfn))
		}
		NumFullNameHits++
		typesByExpr[e] = tdi
		return tdi
	}
	return nil
}

var allTypesSorted = []*Type{}

// This establishes the order in which types are matched by 'case' statements in the "big switch" in goswitch.go. Once established,
// new types cannot be discovered/added.
func SortAll() {
	if len(allTypesSorted) > 0 {
		panic("Attempt to sort all types type after having already sorted all types!!")
	}
	for _, t := range typesByFullName {
		if t.IsExported {
			allTypesSorted = append(allTypesSorted, t)
		}
	}
	sort.SliceStable(allTypesSorted, func(i, j int) bool {
		if allTypesSorted[i].Specificity != allTypesSorted[j].Specificity {
			return allTypesSorted[i].Specificity > allTypesSorted[j].Specificity
		}
		return allTypesSorted[i].FullName < allTypesSorted[j].FullName
	})
	for ord, t := range allTypesSorted {
		t.Ord = (uint)(ord)
		ord++
	}
}

func AllSorted() []*Type {
	return allTypesSorted
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

func SortedTypeDefinitions(m map[*Type]struct{}, f func(ti *Type)) {
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
		f(typesByFullName[k])
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
			tdi := typesByFullName[full]
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
		len := exprToString(x.Len)
		full = "[" + len + "]" + prefix + elFull
		local = "[" + len + "]" + elLocal
	case *StarExpr:
		elFull, elLocal, _ := typeNames(x.X, false)
		full = "*" + prefix + elFull
		local = "*" + elLocal
	}
	return
}

func exprToString(e Expr) string {
	if e == nil {
		return ""
	}
	switch v := e.(type) {
	case *Ellipsis:
		return "..." + exprToString(v.Elt)
	}
	return fmt.Sprintf("%v", e)
}

func (tdi *Type) TypeReflected() (packageImport, pattern string) {
	t := ""
	suffix := ".Elem()"
	if tdiu := tdi.underlyingType; tdiu != nil {
		t = "_" + filepath.Base(tdiu.GoPackage) + "." + tdiu.LocalName
		suffix = ""
	} else {
		t = "_" + filepath.Base(tdi.GoPackage) + "." + tdi.LocalName
	}
	return "reflect", fmt.Sprintf("%%s.TypeOf((*%s)(nil))%s", t, suffix)
}

func (tdi *Type) TypeMappingsName() string {
	if !tdi.IsExported {
		return ""
	}
	if tdi.underlyingType != nil {
		return "info_PtrTo_" + tdi.underlyingType.LocalName
	}
	return "info_" + tdi.LocalName
}
