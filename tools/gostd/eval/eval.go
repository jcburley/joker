package eval

import (
	"fmt"
	. "go/ast"
	"go/token"
)

type Config struct {
	StringType, IntType, FloatType, CharType Expr
}

func (cfg *Config) Eval(ty, e Node) (ts Node, val interface{}) {
	var valType Expr
	switch v := e.(type) {
	case *BasicLit:
		switch v.Kind {
		case token.STRING:
			valType = cfg.StringType
		case token.INT:
			valType = cfg.IntType
		case token.FLOAT:
			valType = cfg.FloatType
		case token.CHAR:
			valType = cfg.CharType
		default:
			panic(fmt.Sprintf("unknown kind=%s for %T %+v", v.Kind, e, e))
		}
	default:
		panic(fmt.Sprintf("unknown node %T %+v", e, e))
	}
	if ty == nil {
		ts = valType
	} else {
		ts = ty
	}
	return ts, 1
}
