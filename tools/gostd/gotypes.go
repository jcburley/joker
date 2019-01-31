package main

import (
	"fmt"
	. "go/ast"
)

type goTypeInfo struct {
	fullName             string
	argClojureType       string
	argFromClojureObject string
}

var goBuiltinTypes = map[string]*goTypeInfo{}
var goTypes = map[string]*goTypeInfo{}

func toGoTypeInfo(ts *TypeSpec) *goTypeInfo {
	return toGoExprInfo(ts.Type)
}

func toGoExprInfo(e Expr) *goTypeInfo {
	fullName := fmt.Sprintf("<notfound>%T</notfound>", e)
	switch td := e.(type) {
	case *Ident:
		fullName = td.Name
		v := goBuiltinTypes[td.Name]
		if v != nil {
			return v
		}
	case *ArrayType:
		return goArrayType(td.Len, td.Elt)
	case *StarExpr:
		return goStarExpr(td.X)
	}
	v := &goTypeInfo{
		fullName:             fullName,
		argClojureType:       "",
		argFromClojureObject: "",
	}
	goTypes[v.fullName] = v
	return v
}

func toGoExprString(e Expr) string {
	t := toGoExprInfo(e)
	if t != nil {
		return t.fullName
	}
	return fmt.Sprintf("%T", e)
}

func goArrayType(len Expr, elt Expr) *goTypeInfo {
	var fullName string
	ev := toGoExprInfo(elt)
	en := toGoExprString(elt)
	if len == nil {
		fullName = "[]" + en
	} else {
		fullName = "..." + en
	}
	if v, ok := goTypes[fullName]; ok {
		return v
	}
	v := &goTypeInfo{
		fullName:             fullName,
		argClojureType:       ev.argClojureType,
		argFromClojureObject: ev.argFromClojureObject,
	}
	goTypes[fullName] = v
	return v
}

func goStarExpr(x Expr) *goTypeInfo {
	ex := toGoExprInfo(x)
	fullName := "*" + ex.fullName
	if v, ok := goTypes[fullName]; ok {
		return v
	}
	v := &goTypeInfo{
		fullName:             fullName,
		argClojureType:       ex.argClojureType,
		argFromClojureObject: ex.argFromClojureObject,
	}
	goTypes[fullName] = v
	return v
}

func init() {
	goBuiltinTypes["string"] = &goTypeInfo{
		fullName:             "string",
		argClojureType:       "String",
		argFromClojureObject: ".S",
	}
	goBuiltinTypes["bool"] = &goTypeInfo{
		fullName:             "bool",
		argClojureType:       "Boolean",
		argFromClojureObject: ".Boolean().B",
	}
	goBuiltinTypes["int"] = &goTypeInfo{
		fullName:             "int",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
	}
	goBuiltinTypes["byte"] = &goTypeInfo{
		fullName:             "byte",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
	}
	goBuiltinTypes["int8"] = &goTypeInfo{
		fullName:             "int8",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
	}
	goBuiltinTypes["int16"] = &goTypeInfo{
		fullName:             "int16",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
	}
	goBuiltinTypes["uint"] = &goTypeInfo{
		fullName:             "uint",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
	}
	goBuiltinTypes["uint8"] = &goTypeInfo{
		fullName:             "uint8",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
	}
	goBuiltinTypes["uint16"] = &goTypeInfo{
		fullName:             "uint16",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
	}
	goBuiltinTypes["int32"] = &goTypeInfo{
		fullName:             "int32",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
	}
	goBuiltinTypes["uint32"] = &goTypeInfo{
		fullName:             "uint32",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
	}
	goBuiltinTypes["int64"] = &goTypeInfo{
		fullName:             "int64",
		argClojureType:       "Number",
		argFromClojureObject: ".BigInt().Int64()",
	}
	goBuiltinTypes["uint64"] = &goTypeInfo{
		fullName:             "uint64",
		argClojureType:       "Number",
		argFromClojureObject: ".BigInt().Uint64()",
	}
	goBuiltinTypes["uintptr"] = &goTypeInfo{
		fullName:             "uintptr",
		argClojureType:       "Number",
		argFromClojureObject: ".BigInt().Uint64()",
	}
}
