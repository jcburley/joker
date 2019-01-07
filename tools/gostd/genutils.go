package main

import (
	"fmt"
	. "go/ast"
	"strconv"
	"strings"
	"unicode"
)

func whereAt(p token.Pos) string {
	return fmt.Sprintf("%s", fset.Position(p).String())
}

func fileAt(p token.Pos) string {
	return token.Position{Filename: fset.Position(p).Filename,
		Offset: 0, Line: 0, Column: 0}.String()
}

func unix(p string) string {
	return filepath.ToSlash(p)
}

func commentGroupInQuotes(doc *CommentGroup, jokIn, jokOut, goIn, goOut string) string {
	var d string
	if doc != nil {
		d = doc.Text()
	}
	if goIn != "" {
		if d != "" {
			d = strings.Trim(d, " \t\n") + "\n\n"
		}
		d += "Go input arguments: " + goIn
	}
	if goOut != "" {
		if d != "" {
			d = strings.Trim(d, " \t\n") + "\n\n"
		}
		d += "Go return type: " + goOut
	}
	if jokIn != "" {
		if d != "" {
			d = strings.Trim(d, " \t\n") + "\n\n"
		}
		d += "Joker input arguments: " + jokIn
	}
	if jokOut != "" {
		if d != "" {
			d = strings.Trim(d, " \t\n") + "\n\n"
		}
		d += "Joker return type: " + jokOut
	}
	return `  ` + strings.Trim(strconv.Quote(d), " \t\n") + "\n"
}

func paramNameAsClojure(n string) string {
	return n
}

func paramNameAsGo(p string) string {
	return p
}

func funcNameAsGoPrivate(f string) string {
	return strings.ToLower(f[0:1]) + f[1:]
}

func isPrivate(p string) bool {
	return !unicode.IsUpper(rune(p[0]))
}

func reverseJoin(a []string, infix string) string {
	j := ""
	for idx := len(a) - 1; idx >= 0; idx-- {
		if idx != len(a)-1 {
			j += infix
		}
		j += a[idx]
	}
	return j
}

var genSymIndex = map[string]int{}

func genSym(pre string) string {
	var idx int
	if i, ok := genSymIndex[pre]; ok {
		idx = i + 1
	} else {
		idx = 1
	}
	genSymIndex[pre] = idx
	return fmt.Sprintf("%s%d", pre, idx)
}

func genSymReset() {
	genSymIndex = map[string]int{}
}

func exprIsUseful(rtn string) bool {
	return rtn != "NIL"
}

// Generates code that, at run time, tests each of the onlyIf's and, if all true, returns the expr; else returns NIL.
func wrapOnlyIfs(onlyIf string, e string) string {
	if len(onlyIf) == 0 {
		return e
	}
	return "func() Object { if " + onlyIf + " { return " + e + " } else { return NIL } }()"
}

// Add one level of indent to each line
func indentedCode(c string) string {
	return "\t" + strings.Replace(c, "\n", "\n\t", -1)
}

func wrapStmtOnlyIfs(indent, v, t, e string, onlyIf string, c string, out *string) string {
	if len(onlyIf) == 0 {
		*out = v
		return indent + v + " := " + e + "\n" + c
	}
	*out = "_obj" + v
	return indent + "var " + *out + " Object\n" +
		indent + "if " + onlyIf + " {\n" +
		indent + "\t" + v + " := " + e + "\n" +
		strings.TrimRight(indentedCode(c), "\t") +
		indent + "\t" + *out + " = Object(" + v + ")\n" +
		indent + "} else {\n" +
		indent + "\t" + *out + " = NIL\n" +
		indent + "}\n"
}
