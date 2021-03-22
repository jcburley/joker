package eval

import (
	"fmt"
	. "go/ast"
	"go/token"
	"strconv"
)

type Config struct {
	StringType, IntType, FloatType, CharType Expr
}

var stringType = &Ident{Name: "string"}
var intType = &Ident{Name: "int"}
var floatType = &Ident{Name: "float"}
var charType = &Ident{Name: "char"}

func (cfg *Config) Eval(ty, e Node) (ts Node, val interface{}) {
	if ty == nil {
		switch v := e.(type) {
		case *BasicLit:
			switch v.Kind {
			case token.STRING:
				ts = stringType
			case token.INT:
				ts = intType
			case token.FLOAT:
				ts = floatType
			case token.CHAR:
				ts = charType
			default:
				panic(fmt.Sprintf("unknown kind=%s for %T %+v", v.Kind, e, e))
			}
		default:
			panic(fmt.Sprintf("unknown node %T %+v", e, e))
		}
	} else {
		ts = ty
	}
	v, err := strconv.Atoi(e.(*BasicLit).Value)
	if err != nil {
		panic(fmt.Sprintf("cannot convert %T %+v: %s", e, e, err))
	}
	return ts, v
}
