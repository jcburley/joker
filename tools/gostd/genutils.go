package main

import (
	"fmt"
	"strings"
)

func paramNameAsClojure(n string) string {
	return n
}

func paramNameAsGo(p string) string {
	return p
}

func typeToGoExtractFuncName(t string) string {
	return strings.ReplaceAll(strings.ReplaceAll(t, ".", "_"), "/", "__")
}

func funcNameAsGoPrivate(f string) string {
	// s := strings.ToLower(f[0:1]) + f[1:]
	// if token.Lookup(s).IsKeyword() || gotypes.Universe.Lookup(s) != nil {
	// 	s = "_" + s
	// }
	return "__" + strings.ToLower(f[0:1]) + f[1:]
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
func clojureReturnTypeForGenerateCustom(in_cl, in_gol string) (cl, gol string) {
	switch in_cl {
	case "String", "Int", "Byte", "Double", "Boolean", "Time", "Error":
		cl = `^"` + in_cl + `"`
	default:
		cl = ""
		gol = "Object"
	}
	return
}
