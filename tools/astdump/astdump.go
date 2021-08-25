package main

import (
	"fmt"
	. "go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"strings"
)

func usage() {
	fmt.Println(`Usage: astdump <dirs>...`)
}

var Fset *token.FileSet

var packagesByPath = map[string]*Package{}

func GetPackageFromPath(path string) *Package {
	if pkg, found := packagesByPath[path]; found {
		return pkg
	}
	return nil
}

func SetPackagePath(path string, pkg *Package) {
	if q, found := packagesByPath[path]; found {
		panic(fmt.Sprintf("already have %s for %s, now want it to be %s", q, path, pkg))
	}
}

var typeCheckerConfig *types.Config
var typeCheckerInfo *types.Info

type importerFunc func(path string) (*types.Package, error)

func (f importerFunc) Import(path string) (*types.Package, error) {
	return f(path)
}

func myImporter(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}
	pkg := GetPackageFromPath(path)
	files := []*File{}
	for _, f := range pkg.Files {
		files = append(files, f)
	}
	return typeCheckerConfig.Check(path, Fset, files, typeCheckerInfo)
}

func main() {
	Fset := token.NewFileSet()
	allDirs := false
	for _, dir := range os.Args[1:] {
		if !allDirs {
			if dir == "--" {
				allDirs = true
				continue
			}
			if dir[0] == '-' {
				fmt.Fprintf(os.Stderr, "Unrecognized option: %s\n", dir)
				usage()
				os.Exit(1)
			}
		}
		pkgs, err := parser.ParseDir(Fset, dir,
			func(info fs.FileInfo) bool {
				if strings.HasSuffix(info.Name(), "_test.go") {
					return false
				}
				b, e := build.Default.MatchFile(dir, info.Name())
				return b && e == nil
			},
			parser.ParseComments|parser.AllErrors)
		if err != nil {
			fmt.Fprintf(os.Stderr, "astdump: cannot parse directory %s: %s\n", dir, err)
			os.Exit(2)
		}

		for p, v := range pkgs {
			SetPackagePath(dir, v)
			fmt.Printf("--------\nastdump: Dumping %s:\n", p)
			Print(Fset, v)
			fmt.Println("--------")
		}
	}

	typeCheckerConfig = &types.Config{
		IgnoreFuncBodies: true,
		FakeImportC:      true,
		Importer:         importer.Default(),
	}
	typeCheckerInfo = &types.Info{
		Types: map[Expr]types.TypeAndValue{},
		Defs:  map[*Ident]types.Object{},
	}
}
