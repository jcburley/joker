package godb

import (
	"fmt"
	. "go/ast"
	"go/token"
	"path/filepath"
	. "strings"
)

func Resolve(n Node) Node {
	switch n.(type) {
	case *Ident:
	}
	return nil
}

var Fset *token.FileSet

func WhereAt(p token.Pos) string {
	return fmt.Sprintf("%s", Fset.Position(p).String())
}

func FileAt(p token.Pos) string {
	return token.Position{Filename: Fset.Position(p).Filename,
		Offset: 0, Line: 0, Column: 0}.String()
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

func goPackageForDirname(dirName string) (pkg, prefix string) {
	for _, m := range mappings {
		if HasPrefix(dirName, m.prefix) {
			return dirName[len(m.prefix)+1:], m.cljRoot
		}
	}
	return "", mappings[0].cljRoot
}

func GoPackageForExpr(e Expr) string {
	dirName := filepath.Dir(Fset.Position(e.Pos()).Filename)
	pkg, _ := goPackageForDirname(dirName)
	if pkg == "" {
		panic(fmt.Sprintf("no mapping for %s", dirName))
	}
	return pkg
}

func GoPackageForTypeSpec(ts *TypeSpec) string {
	dirName := filepath.Dir(Fset.Position(ts.Pos()).Filename)
	pkg, _ := goPackageForDirname(dirName)
	if pkg == "" {
		panic(fmt.Sprintf("no mapping for %s", dirName))
	}
	return pkg
}

func ClojureNamespaceForPos(p token.Position) string {
	dirName := filepath.Dir(p.Filename)
	pkg, root := goPackageForDirname(dirName)
	if pkg == "" {
		panic(fmt.Sprintf("no mapping for %s", dirName))
	}
	return root + ReplaceAll(pkg, (string)(filepath.Separator), ".")
}

func ClojureNamespaceForExpr(e Expr) string {
	return ClojureNamespaceForPos(Fset.Position(e.Pos()))
}

func ClojureNamespaceForDirname(d string) string {
	pkg, root := goPackageForDirname(d)
	if pkg == "" {
		pkg = root + d
	}
	return ReplaceAll(pkg, (string)(filepath.Separator), ".")
}

func GoPackageBaseName(e Expr) string {
	return filepath.Base(filepath.Dir(Fset.Position(e.Pos()).Filename))
}
