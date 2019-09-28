package types

import (
	"fmt"
	. "github.com/candid82/joker/tools/gostd/utils"
	. "go/ast"
	"sort"
)

type TypeInfo struct {
	Type     Expr
	FullName string
}

// Maps the "definitive" (first-found) Expr for a type to type info
var types = map[Expr]*TypeInfo{}

// Maps a non-definitive Expr for a type to the definitive Expr for the same type
var typeAliases = map[Expr]Expr{}

// Maps the full name for a type to the definitive Expr for the same type
var typesByFullName = map[string]Expr{}

func TypeLookup(e Expr) *TypeInfo {
	if ti, ok := types[e]; ok {
		return ti
	}
	if ta, ok := typeAliases[e]; ok {
		return types[ta]
	}
	tfn := typeFullName(e)
	if te, ok := typesByFullName[tfn]; ok {
		typeAliases[te] = e
		return types[te]
	}
	ti := &TypeInfo{
		Type:     e,
		FullName: tfn}
	types[e] = ti
	typesByFullName[tfn] = e
	return ti
}

func SortedTypes(m map[*TypeInfo]struct{}, f func(ti *TypeInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k.FullName)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(types[typesByFullName[k]])
	}
}

// func typeName(e Expr) string {
// 	res := ""
// 	switch x := e.(type) {
// 	case *Ident:
// 		res += x.Name
// 	case *ArrayType:
// 		res += "ArrayOf_" + x.Elt.(*Ident).Name
// 	case *StarExpr:
// 		res += "PtrTo_" + x.X.(*Ident).Name
// 	default:
// 		panic(fmt.Sprintf("typeName: unrecognized expr %T", x))
// 	}
// 	return "info_" + res
// }

// r, "go.std."+pkgDirUnix+"/"
func typeFullName(e Expr) string {
	res := ""
	prefix := GoPackageForExpr(e)
	switch x := e.(type) {
	case *Ident:
		res += prefix + x.Name
	case *ArrayType:
		res += "[]" + prefix + x.Elt.(*Ident).Name
	case *StarExpr:
		res += "*" + prefix + x.X.(*Ident).Name
	default:
		panic(fmt.Sprintf("typeFullName: unrecognized expr %T", x))
	}
	return res
}

func (ti *TypeInfo) TypeReflected() string {
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
		panic(fmt.Sprintf("typeKey: unrecognized expr %T", x))
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
		panic(fmt.Sprintf("typeKey: unrecognized expr %T", x))
	}
	return fmt.Sprintf("_reflect.TypeOf((%s)(nil))%s", t, suffix)
}
