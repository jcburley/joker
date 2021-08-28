package main

import (
	"flag"
	"fmt"
	. "go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"sort"
	"strings"
)

var dumpAST = flag.Bool("ast", true, "whether to dump the AST that results from parsing")
var dumpTypes = flag.Bool("types", false, "whether to dump the go/types output (after any AST output)")

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
		panic(fmt.Sprintf("already have %v for %s, now want it to be %v", q, path, pkg))
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
	flag.Parse()

	Fset := token.NewFileSet()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
		return
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

	if !(*dumpAST || *dumpTypes) {
		fmt.Fprintln(os.Stderr, "(not much to do!!)")
	}

	for _, dir := range flag.Args() {
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
			pkg := types.NewPackage(dir, p)
			files := []*File{}
			for _, f := range v.Files {
				files = append(files, f)
			}
			chk := types.NewChecker(typeCheckerConfig, Fset, pkg, typeCheckerInfo)
			if e := chk.Files(files); e != nil {
				fmt.Fprintf(os.Stderr, "chk.Files(%v) returned error %s\n", files, e)
			}
			if *dumpAST {
				fmt.Printf("--------\nastdump: Dumping %s:\n", p)
				Print(Fset, v)
				fmt.Println("--------")
			}
		}
	}

	if !*dumpTypes {
		return
	}

	type L struct {
		p token.Pos
		k *Ident
	}
	lines := []L{}
	for k, _ := range typeCheckerInfo.Defs {
		p := k.Pos()
		lines = append(lines, L{p: p, k: k})
	}
	sort.SliceStable(lines, func(i, j int) bool {
		if lines[i].p < lines[j].p {
			return true
		}
		if lines[i].p > lines[j].p {
			return false
		}
		return lines[i].k.Name < lines[j].k.Name
	})
	for _, k := range lines {
		s := fmt.Sprintf("%s: %s => %s", Fset.Position(k.p), k.k, typeCheckerInfo.Defs[k.k])
		fmt.Fprintln(os.Stderr, s)
	}
}

func init() {
	var flagUsage = flag.Usage
	flag.Usage = func() {
		flagUsage()
		fmt.Fprintf(os.Stderr, "  dir...\n\tone or more directories to parse/dump.\n")
	}
}
