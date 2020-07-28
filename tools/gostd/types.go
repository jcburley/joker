package main

import (
	//	"github.com/candid82/joker/tools/gostd/gtypes"
	"github.com/candid82/joker/tools/gostd/jtypes"
	. "go/ast"
	//	"strings"
)

var typeMap = map[string]jtypes.Info{}

func JokerTypeInfoForExpr(e Expr) jtypes.Info {
	switch td := e.(type) {
	case *Ident:
		if jti, found := typeMap[td.Name]; found {
			return jti
		}
	}
	return jtypes.Nil
}

func init() {
	typeMap["string"] = jtypes.String
}
