package utils

import (
	"fmt"
	. "go/ast"
	"go/token"
	. "strings"
)

var Fset *token.FileSet

func WhereAt(p token.Pos) string {
	return fmt.Sprintf("%s", Fset.Position(p).String())
}

type mapping struct {
	prefix      string
	jokerPrefix string
}

var mappings = []mapping{}

func AddMapping(dir string, prefix string) {
	for _, m := range mappings {
		if HasPrefix(dir, m.prefix) {
			panic(fmt.Sprintf("duplicate mapping %s and %s", dir, m.prefix))
		}
	}
	mappings = append(mappings, mapping{dir, prefix})
}

func GoPackageForFilename(fileName string) string {
	for _, m := range mappings {
		if HasPrefix(fileName, m.prefix) {
			return fileName[len(m.prefix)+1:]
		}
	}
	panic(fmt.Sprintf("no mapping for %s", fileName))
}

func GoPackageForExpr(e Expr) string {
	return GoPackageForFilename(Fset.Position(e.Pos()).Filename)
}
