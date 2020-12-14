package main

import (
	"fmt"
	. "go/ast"
	"go/token"
	"strconv"
)

func evalExpr(e Expr) interface{} {
	switch v := e.(type) {
	case *BasicLit:
		switch v.Kind {
		case token.STRING:
			return v.Value
		case token.INT:
			res, err := strconv.Atoi(v.Value)
			Check(err)
			return res
		default:
			panic(fmt.Sprintf("unsupported BasicLit type %T", v.Kind))
		}
	case *ParenExpr:
		return evalExpr(v.X)
	}
	return nil
}
