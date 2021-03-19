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
		real = fmt.Sprintf("ABEND229(non-int expression %T at %s)", e, WhereAt(e.Pos()))
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

// Return arbitrary, yet somewhat-identifiable and "unique", string for an expr.
func ExprToString(e Expr) string {
	if e == nil {
		return ""
	}
	switch v := e.(type) {
	case *Ident:
		return v.Name
	case *SelectorExpr:
		return ExprToString(v.X) + "." + v.Sel.Name
	case *ArrayType:
		return "[" + ExprToString(v.Len) + "]" + ExprToString(v.Elt)
	case *StarExpr:
		return "*" + ExprToString(v.X)
	case *ChanType:
		dir := "chan"
		switch d := v.Dir & (SEND | RECV); d {
		case SEND:
			dir = "chan<-"
		case RECV:
			dir = "<-chan"
		case SEND | RECV:
			dir = "chan"
		default:
			panic(fmt.Sprintf("unrecognized channel direction %d", d))
		}
		return dir + " " + ExprToString(v.Value)
	case *InterfaceType:
		incomplete := ""
		if v.Incomplete {
			incomplete = ",..."
		}
		fl := FieldListToString(v.Methods, "{", "}")
		return "interface" + fl + incomplete
	case *MapType:
		return "map[" + ExprToString(v.Key) + "]" + ExprToString(v.Value)
	case *StructType:
		incomplete := ""
		if v.Incomplete {
			incomplete = ",..."
		}
		return "struct" + FieldListToString(v.Fields, "{", "}") + incomplete
	case *FuncType:
		return "func" + FieldListToString(v.Params, "(", ")") + FieldListToString(v.Results, "(", ")")
	case *Ellipsis:
		return "..." + ExprToString(v.Elt)
	case *BasicLit:
		return BasicLitToString(v)
	case *BinaryExpr:
		return ExprToString(v.X) + v.Op.String() + ExprToString(v.Y)
	case *ParenExpr:
		return "(" + ExprToString(v.X) + ")"
	case *CallExpr:
		ellipsis := ""
		if v.Ellipsis != token.NoPos {
			if v.Args != nil && len(v.Args) != 0 {
				ellipsis = ",..."
			} else {
				ellipsis = "..."
			}
		}
		return ExprToString(v.Fun) + "(" + ExprArrayToString(v.Args) + ellipsis + ")"
	case *CompositeLit:
		incomplete := ""
		if v.Incomplete {
			if v.Elts != nil {
				incomplete = ",..."
			} else {
				incomplete = "..."
			}
		}
		return "{" + ExprArrayToString(v.Elts) + incomplete + "}"
	default:
		panic(fmt.Sprintf("unrecognized expr type %T", v))
	}
}

func ExprArrayToString(ea []Expr) string {
	rl := []string{}
	for _, e := range ea {
		rl = append(rl, ExprToString(e))
	}
	return strings.Join(rl, ",")
}

func FieldListToString(fl *FieldList, open, close string) string {
	f := FlattenFieldList(fl)
	rl := []string{}
	for _, it := range f {
		s := ""
		if it.Name != nil {
			s = it.Name.Name + " "
		}
		s += fieldToString(it.Field)
		rl = append(rl, s)
	}
	r := strings.Join(rl, ",")
	if fl != nil && fl.Opening != token.NoPos {
		if open == "" {
			panic(fmt.Sprintf("no opening for expr at %s", WhereAt(fl.Pos())))
		}
		r = open + r
	}
	if fl != nil && fl.Closing != token.NoPos {
		if close == "" {
			panic(fmt.Sprintf("no closing for expr at %s", WhereAt(fl.Pos())))
		}
		r += close
	}
	return r
}

// Private, as this ignores f.Names, which the caller must handle.
func fieldToString(f *Field) string {
	s := ExprToString(f.Type)
	if f.Tag != nil {
		s += BasicLitToString(f.Tag)
	}
	return s
}

func BasicLitToString(b *BasicLit) string {
	return b.Kind.String() + "(" + b.Value + ")"
}

var WhereAt func(p token.Pos) string
