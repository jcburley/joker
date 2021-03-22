package main

import (
	"fmt"
	. "go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"strings"
)

func usage() {
	fmt.Println(`Usage: astdump <dirs>...`)
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
			fmt.Printf("--------\nastdump: Dumping %s:\n", p)
			Print(Fset, v)
			fmt.Println("--------")
		}
	}
}
