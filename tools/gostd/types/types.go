package types

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/godb"
	. "github.com/candid82/joker/tools/gostd/godb"
	. "github.com/candid82/joker/tools/gostd/utils"
	. "go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"
)

const Concrete = ^uint(0) /* MaxUint */

var NumExprHits uint
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

func define(tdi *Type) {
	name := tdi.FullName
	if existingTdi, ok := typesByFullName[name]; ok {
		panic(fmt.Sprintf("already defined type %s at %s and again at %s", name, WhereAt(existingTdi.DefPos), WhereAt(tdi.DefPos)))
	}
	typesByFullName[name] = tdi

	if tdi.Type != nil {
		tdiByExpr, found := typesByExpr[tdi.Type]
		if found && tdiByExpr != tdi {
			panic(fmt.Sprintf("different expr for type %s", name))
		}
		typesByExpr[tdi.Type] = tdi
	}

	//	fmt.Printf("define: %s\n", name)
}

func TypeDefine(ts *TypeSpec, gf *godb.GoFile, parentDoc *CommentGroup) []*Type {
	if len(allTypesSorted) > 0 {
		panic("Attempt to define new type after having sorted all types!!")
	}

	prefix := ClojureNamespaceForPos(Fset.Position(ts.Name.NamePos)) + "/"
	tln := ts.Name.Name
	tfn := prefix + tln

	doc := ts.Doc // Try block comments for this specific decl
	if doc == nil {
		doc = ts.Comment // Use line comments if no preceding block comments are available
	}
	if doc == nil {
		doc = parentDoc // Use 'var'/'const' statement block comments as last resort
	}

	types := []*Type{}

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
	define(tdi)
	types = append(types, tdi)

	tdiArrayOf := &Type{
		Type:           &ArrayType{Elt: tdi.Type},
		FullName:       "[]" + tdi.FullName,
		LocalName:      "[]" + tdi.LocalName,
		IsExported:     tdi.IsExported,
		Doc:            "",
		GoPrefix:       "[]" + tdi.GoPrefix,
		GoPackage:      tdi.GoPackage,
		GoName:         tdi.GoName,
		underlyingType: tdi,
		Specificity:    Concrete,
	}
	define(tdiArrayOf)
	types = append(types, tdiArrayOf)

	tdiArray16Of := &Type{
		Type:           &ArrayType{Elt: tdi.Type},
		FullName:       "[16]" + tdi.FullName,
		LocalName:      "[16]" + tdi.LocalName,
		IsExported:     tdi.IsExported,
		Doc:            "",
		GoPrefix:       "[16]" + tdi.GoPrefix,
		GoPackage:      tdi.GoPackage,
		GoName:         tdi.GoName,
		underlyingType: tdi,
		Specificity:    Concrete,
	}
	define(tdiArray16Of)
	types = append(types, tdiArray16Of)

	tdiArrayOfArrayOf := &Type{
		Type:           &ArrayType{Elt: tdiArrayOf.Type},
		FullName:       "[]" + tdiArrayOf.FullName,
		LocalName:      "[]" + tdiArrayOf.LocalName,
		IsExported:     tdiArrayOf.IsExported,
		Doc:            "",
		GoPrefix:       "[]" + tdiArrayOf.GoPrefix,
		GoPackage:      tdiArrayOf.GoPackage,
		GoName:         tdi.GoName,
		underlyingType: tdiArrayOf,
		Specificity:    Concrete,
	}
	define(tdiArrayOfArrayOf)
	types = append(types, tdiArrayOfArrayOf)

	if tdi.Specificity == Concrete {
		// Concrete types get reference-to and array-of-reference-to versions.
		tdiPtrTo := &Type{
			Type:           &StarExpr{X: tdi.Type},
			FullName:       "*" + tdi.FullName,
			LocalName:      "*" + tdi.LocalName,
			IsExported:     tdi.IsExported,
			Doc:            "",
			GoPrefix:       "*" + tdi.GoPrefix,
			GoPackage:      tdi.GoPackage,
			GoName:         tdi.GoName,
			underlyingType: tdi,
			Specificity:    Concrete,
		}
		define(tdiPtrTo)
		types = append(types, tdiPtrTo)

		tdiArrayOfPtrTo := &Type{
			Type:           &ArrayType{Elt: tdiPtrTo.Type},
			FullName:       "[]" + tdiPtrTo.FullName,
			LocalName:      "[]" + tdiPtrTo.LocalName,
			IsExported:     tdiPtrTo.IsExported,
			Doc:            "",
			GoPrefix:       "[]" + tdiPtrTo.GoPrefix,
			GoPackage:      tdiPtrTo.GoPackage,
			GoName:         tdi.GoName,
			underlyingType: tdi,
			Specificity:    Concrete,
		}
		define(tdiArrayOfPtrTo)
		types = append(types, tdiArrayOfPtrTo)

		tdiArrayOneOfPtrTo := &Type{
			Type:           &ArrayType{Elt: tdiPtrTo.Type},
			FullName:       "[1]" + tdiPtrTo.FullName,
			LocalName:      "[1]" + tdiPtrTo.LocalName,
			IsExported:     tdiPtrTo.IsExported,
			Doc:            "",
			GoPrefix:       "[1]" + tdiPtrTo.GoPrefix,
			GoPackage:      tdiPtrTo.GoPackage,
			GoName:         tdi.GoName,
			underlyingType: tdi,
			Specificity:    Concrete,
		}
		define(tdiArrayOneOfPtrTo)
		types = append(types, tdiArrayOneOfPtrTo)
	}

	return types
}

// Maps type-defining Expr to exactly one struct describing that type
var typesByExpr = map[Expr]*Type{}

func defineVariant(name string, innerTdi *Type, te Expr) *Type {
	if _, ok := typesByFullName[name]; ok {
		panic(fmt.Sprintf("already defined builtin type %s", name))
	}

	if _, ok := typesByExpr[te]; ok {
		panic(fmt.Sprintf("already defined builtin type %s via expr", name))
	}

	tdi := &Type{
		Type:       te,
		FullName:   name,
		LocalName:  name,
		IsExported: true,
		GoName:     name,
	}

	typesByFullName[name] = tdi
	typesByExpr[te] = tdi

	//	fmt.Printf("defineVariant: %s\n", name)

	return tdi
}

func TypeDefineBuiltin(name string) *Type {
	te := &Ident{Name: name}
	return defineVariant(name, nil, te)
}

func dynamicDefine(prefixes []string, e Expr) (ty *Type, fullName string) {
	switch v := e.(type) {
	case *Ident:
		break
	case *StarExpr:
		return dynamicDefine(append(prefixes, "*"), v.X)
	case *ArrayType:
		return dynamicDefine(append(prefixes, exprToString(v.Len)), v.Elt)
	default:
		return nil, strings.Join(prefixes, "")
	}
	// Try defining the type here.
	return dynamicDefine([]string{}, e)

	return nil, strings.Join(prefixes, "")
}

func TypeLookup(e Expr) (ty *Type, fullName string) {
	if tdi, ok := typesByExpr[e]; ok {
		NumExprHits++
		return tdi, tdi.FullName
	}
	tfn := typeName(e)
	if tdi, ok := typesByFullName[tfn]; ok {
		NumFullNameHits++
		typesByExpr[e] = tdi
		return tdi, tfn
	}

	if _, yes := e.(*Ident); yes {
	}

	var innerTdi *Type
	var innerTfn string
	switch v := e.(type) {
	case *StarExpr:
		innerTdi, innerTfn = TypeLookup(v.X)
		tfn = "*" + innerTfn
	case *ArrayType:
		innerTdi, innerTfn = TypeLookup(v.Elt)
		tfn = "[" + exprToString(v.Len) + "]" + innerTfn
	}

	return defineVariant(tfn, innerTdi, e), tfn
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

func typeName(e Expr) (full string) {
	switch x := e.(type) {
	case *Ident:
		break
	case *ArrayType:
		elFull := typeName(x.Elt)
		len := exprToString(x.Len)
		full = "[" + len + "]" + elFull
		return
	case *StarExpr:
		elFull := typeName(x.X)
		full = "*" + elFull
		return
	default:
		return
	}

	x := e.(*Ident)
	local := x.Name
	prefix := ""
	if types.Universe.Lookup(local) == nil {
		prefix = ClojureNamespaceForExpr(e) + "/"
	}
	full = prefix + local

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

	return
}

func exprToString(e Expr) string {
	if e == nil {
		return ""
	}
	switch v := e.(type) {
	case *Ellipsis:
		return "..." + exprToString(v.Elt)
	case *BasicLit:
		return v.Value
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
