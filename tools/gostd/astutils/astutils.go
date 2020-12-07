package astutils

// Helpers for wrangling Go AST.

import (
	"fmt"
	. "go/ast"
	"go/types"
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
