package astutils

// Helpers for wrangling Go AST.

import (
	"bytes"
	"fmt"
	. "go/ast"
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

func AnyNamesExported(names []*Ident) bool {
	if names == nil {
		return false
	}
	for _, n := range names {
		if IsExported(n.Name) {
			return true
		}
	}
	return false
}

var TypeCheckerInfo *types.Info

func IntExprToString(e Expr) (real, doc string) {
	if e == nil {
		return
	}

	if typeAndValue, found := TypeCheckerInfo.Types[e]; found {
		typ, val := types.Default(typeAndValue.Type), typeAndValue.Value
		if typ.String() == "int" {
			real = val.ExactString()
			doc = real
			return
		}
	}
	real = fmt.Sprintf("ABEND333(asutils.go/IntExprToString: Not an integer constant: %s)", ExprToString(e))
	doc = real
	return
}

func ExprToString(e Expr) string {
	if e == nil {
		return "<nil>"
	}

	if typeAndValue, found := TypeCheckerInfo.Types[e]; found {
		typ, val := typeAndValue.Type, typeAndValue.Value
		if typ == nil {
			if val == nil {
				return "<<nil>>"
			}
			return strconv.Quote(val.String())
		}
		if val == nil {
			return strconv.Quote(typ.String())
		}
		return fmt.Sprintf("%q (type %q)", typ.String(), val.String())
	}
	return fmt.Sprintf("ABEND334(astutils.go/ExprToString: Cannot find expression %q)", e)
}

func TypePathname(ty types.Type) string {
	buf := new(bytes.Buffer)
	types.WriteType(buf, ty, nil)
	return buf.String()
}

func TypePathnameFromExpr(e Expr) string {
	tav := TypeCheckerInfo.Types[e].Type
	return TypePathname(tav)
}
