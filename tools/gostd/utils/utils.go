package utils

import (
	"fmt"
	. "go/ast"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	. "strings"
)

func Check(e error) {
	if e != nil {
		panic(e)
	}
}

var Fset *token.FileSet

// func SortedStringMap(m map[string]string, f func(key, value string)) {
// 	var keys []string
// 	for k, _ := range m {
// 		keys = append(keys, k)
// 	}
// 	sort.Strings(keys)
// 	for _, k := range keys {
// 		f(k, m[k])
// 	}
// }

// func ReverseJoin(a []string, infix string) string {
// 	j := ""
// 	for idx := len(a) - 1; idx >= 0; idx-- {
// 		if idx != len(a)-1 {
// 			j += infix
// 		}
// 		j += a[idx]
// 	}
// 	return j
// }

func WhereAt(p token.Pos) string {
	return fmt.Sprintf("%s", Fset.Position(p).String())
}

func FileAt(p token.Pos) string {
	return token.Position{Filename: Fset.Position(p).Filename,
		Offset: 0, Line: 0, Column: 0}.String()
}

func Unix(p string) string {
	return filepath.ToSlash(p)
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

func CommentGroupAsString(d *CommentGroup) string {
	s := ""
	if d != nil {
		s = d.Text()
	}
	return s
}

func CommentGroupInQuotes(doc *CommentGroup, jokIn, jokOut, goIn, goOut string) string {
	var d string
	if doc != nil {
		d = doc.Text()
	}
	if goIn != "" {
		if d != "" {
			d = Trim(d, " \t\n") + "\n\n"
		}
		d += "Go input arguments: " + goIn
	}
	if goOut != "" {
		if d != "" {
			d = Trim(d, " \t\n") + "\n\n"
		}
		d += "Go return type: " + goOut
	}
	if jokIn != "" {
		if d != "" {
			d = Trim(d, " \t\n") + "\n\n"
		}
		d += "Joker input arguments: " + jokIn
	}
	if jokOut != "" {
		if d != "" {
			d = Trim(d, " \t\n") + "\n\n"
		}
		d += "Joker return type: " + jokOut
	}
	return Trim(strconv.Quote(d), " \t\n")
}

type FieldItem struct {
	Name  *Ident
	Field *Field
}

func FlattenFieldList(fl *FieldList) (items []FieldItem) {
	items = []FieldItem{}
	if fl == nil {
		return
	}
	for _, f := range fl.List {
		if f.Names == nil {
			items = append(items, FieldItem{nil, f})
			continue
		}
		for _, n := range f.Names {
			items = append(items, FieldItem{n, f})
		}
	}
	return
}

var outs map[string]struct{}

func StartSortedOutput() {
	outs = map[string]struct{}{}
}

func AddSortedOutput(s string) {
	outs[s] = struct{}{}
}

func EndSortedOutput() {
	var keys []string
	for k, _ := range outs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		os.Stdout.WriteString(k)
	}
}
