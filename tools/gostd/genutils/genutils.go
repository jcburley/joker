package genutils

// Helpers for bridging between Go AST and generated Go and Clojure code.

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/astutils"
	. "go/ast"
	"os"
	"sort"
	"strconv"
	"strings"
)

func ParamNameAsClojure(n string) string {
	return n
}

func ParamNameAsGo(p string) string {
	return p
}

func FuncNameAsGoPrivate(f string) string {
	// s := strings.ToLower(f[0:1]) + f[1:]
	// if token.Lookup(s).IsKeyword() || gotypes.Universe.Lookup(s) != nil {
	// 	s = "_" + s
	// }
	return "_f_" + strings.ToLower(f[0:1]) + f[1:]
}

func FullTypeNameAsClojure(nsRoot, t string) string {
	if t[0] == '_' {
		t = t[1:]
	}
	return nsRoot + strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(t, ".", ":"), "/", "."), ":", "/")
}

var genSymIndex = map[string]int{}

func GenSym(pre string) string {
	var idx int
	if i, ok := genSymIndex[pre]; ok {
		idx = i + 1
	} else {
		idx = 1
	}
	genSymIndex[pre] = idx
	return fmt.Sprintf("%s%d", pre, idx)
}

func GenSymReset() {
	genSymIndex = map[string]int{}
}

func ExprIsUseful(rtn string) bool {
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
	return strings.ReplaceAll("\t"+strings.ReplaceAll(c, "\n", "\n\t"), "\t\n", "\n")
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

// Return a form of the return type as supported by generate-std.joke,
// or empty string if not supported (which will trigger attempting to
// generate appropriate code for *_native.go). gol either passes
// through or "Object" is returned for it if cl is returned as empty.
func ClojureReturnTypeForGenerateCustom(in_cl, in_gol string) (cl, gol string) {
	switch in_cl {
	case "String", "Int", "Byte", "Double", "Boolean", "Time", "Error":
		cl = `^"` + in_cl + `"`
	default:
		cl = ""
		gol = "Object"
	}
	return
}

func CombineGoName(pkg, name string) string {
	if pkg == "" || astutils.IsBuiltin(name) {
		return name
	}
	return pkg + "." + name
}

func CombineClojureName(ns, name string) string {
	if ns == "" {
		return name
	}
	return ns + "/" + name
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
			d = strings.Trim(d, " \t\n") + "\n\n"
		}
		d += "Go input arguments: " + goIn
	}
	if goOut != "" {
		if d != "" {
			d = strings.Trim(d, " \t\n") + "\n\n"
		}
		d += "Go returns: " + goOut
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
		d += "Joker returns: " + jokOut
	}
	return strings.Trim(strconv.Quote(d), " \t\n")
}

var outs map[string]struct{}

func StartSortedStdout() {
	outs = map[string]struct{}{}
}

func AddSortedStdout(s string) {
	outs[s] = struct{}{}
}

func EndSortedStdout() {
	var keys []string
	for k, _ := range outs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		os.Stdout.WriteString(k)
	}
}
