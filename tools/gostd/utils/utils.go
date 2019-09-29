package utils

import (
	"fmt"
	. "go/ast"
	"go/token"
	"path/filepath"
	. "strings"
)

var Fset *token.FileSet

func WhereAt(p token.Pos) string {
	return fmt.Sprintf("%s", Fset.Position(p).String())
}

type mapping struct {
	prefix  string
	cljRoot string
}

var mappings = []mapping{}

func AddMapping(dir string, root string) {
	for _, m := range mappings {
		if HasPrefix(dir, m.prefix) {
			panic(fmt.Sprintf("duplicate mapping %s and %s", dir, m.prefix))
		}
	}
	mappings = append(mappings, mapping{dir, root})
}

func GoPackageForFilename(dirName string) (pkg, prefix string) {
	for _, m := range mappings {
		if HasPrefix(dirName, m.prefix) {
			return dirName[len(m.prefix)+1:], m.cljRoot
		}
	}
	panic(fmt.Sprintf("no mapping for %s", dirName))
}

func GoPackageForExpr(e Expr) string {
	pkg, _ := GoPackageForFilename(filepath.Dir(Fset.Position(e.Pos()).Filename))
	return pkg
}

func ClojureNamespaceForExpr(e Expr) string {
	pkg, root := GoPackageForFilename(filepath.Dir(Fset.Position(e.Pos()).Filename))
	return root + ReplaceAll(pkg, "/", ".")
}
