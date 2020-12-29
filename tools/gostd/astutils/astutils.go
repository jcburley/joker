package astutils

// Helpers for wrangling Go AST.

import (
	"fmt"
	. "go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"
)

type FieldItem struct {
	Name  *Ident
	Field *Field
}

func FlattenFieldList(fl *FieldList) (items []FieldItem) {
	items = []FieldItem{}
	if fl == nil {
		return
	}
	for _, f := range fl.List {
		if f.Names == nil {
			items = append(items, FieldItem{nil, f})
			continue
		}
		for _, n := range f.Names {
			items = append(items, FieldItem{n, f})
		}
	}
	return
}

func TypeAsString(f *Field) string {
	return "hey"
}

func FieldListAsString(fl *FieldList, needParens bool, fn func(f *Field) string) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}
	items := FlattenFieldList(fl)
	if len(items) < 2 {
		return fn(items[0].Field)
	}

	types := make([]string, len(items))
	for ix, it := range items {
		types[ix] = fn(it.Field)
	}

	res := strings.Join(types, ", ")

	if needParens {
		return "(" + res + ")"
	}

	return res
}

func IsBuiltin(name string) bool {
	return types.Universe.Lookup(name) != nil
}

func IsExportedType(f *Expr) bool {
	switch td := (*f).(type) {
	case *Ident:
		return IsExported(td.Name)
	case *ArrayType:
		return IsExportedType(&td.Elt)
	case *StarExpr:
		return IsExportedType(&td.X)
	default:
		panic(fmt.Sprintf("unsupported expr type %T", f))
	}
}

func EvalExpr(e Expr) interface{} {
	switch v := e.(type) {
	case *BasicLit:
		switch v.Kind {
		case token.STRING:
			return v.Value
		case token.INT:
			res, err := strconv.Atoi(v.Value)
			if err != nil {
				panic(err)
			}
			return res
		default:
			panic(fmt.Sprintf("unsupported BasicLit type %T", v.Kind))
		}
	case *ParenExpr:
		return EvalExpr(v.X)
	}
	return nil
}

func IntExprToString(e Expr) (real, doc string) {
	if e == nil {
		return
	}

	res := EvalExpr(e)
	switch r := res.(type) {
	case int:
		real = fmt.Sprintf("%d", r)
	default:
		real = fmt.Sprintf("ABEND229(non-int expression %T at %s)", res, WhereAt(e.Pos()))
	}

	doc = IntExprToDocString(e)

	return
}

func IntExprToDocString(e Expr) string {
	switch v := e.(type) {
	case *BasicLit:
		return v.Value
	case *Ident:
		return v.Name
	}
	return "???"
}

var WhereAt func(p token.Pos) string
