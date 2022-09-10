package main

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/genutils"
	"go/types"
	"strings"
)

func maybeNil(expr, captureName string) string {
	return "func () Object { if (" + expr + ") == nil { return NIL } else { return " + captureName + " } }()"
}

func genGoPostExpr(captureName string, v *types.Var) (cl, clDoc, gol, out, conversion string) {
	ty := v.Type()
	ti := TypeInfoForType(ty)
	if ti == nil {
		return "", "", "", fmt.Sprintf("ABEND750(unsupported type for %q)", captureName), ""
	}
	if ti.AsClojureObject() == "" {
		out = fmt.Sprintf("MakeGoObjectIfNeeded(%s)", captureName)
	} else {
		out = "Make" + fmt.Sprintf(ti.AsClojureObject(), captureName, "")
	}
	if ti.IsNullable() {
		out = maybeNil(captureName, out)
	}
	cl = ti.ClojureName()
	clDoc = ti.ClojureNameDocForType(v.Pkg())
	gol = ti.GoNameDocForType(v.Pkg())
	conversion = ti.PromoteType()

	return
}

const resultName = "_res"

func genGoPostItem(captureName string, v *types.Var) (captureVar, cl, clDoc, gol, out, conversion string) {
	captureVar = captureName
	if captureName == "" || captureName == "_" {
		captureVar = genutils.GenSym(resultName)
	}
	cl, clDoc, gol, out, conversion = genGoPostExpr(captureVar, v)
	if captureName != "" && captureName != resultName {
		gol = genutils.ParamNameAsGo(captureName) + " " + gol
	}
	return
}

// Caller generates "outGOCALL;goc" while saving cl and gol for type info (they go into .joke as metadata and docstrings)
func genGoPostList(indent string, tuple *types.Tuple) (cl, clDoc, gol, goc, out, conversion string) {
	captureVars := []string{}
	clType := []string{}
	clTypeDoc := []string{}
	golType := []string{}
	goCode := []string{}

	result := resultName
	args := tuple.Len()
	useful := args > 0
	multipleCaptures := args > 1
	for argNum := 0; argNum < args; argNum++ {
		field := tuple.At(argNum)
		n := field.Name()
		captureName := result
		if multipleCaptures {
			captureName = n
		}
		captureVar, clNew, clDocNew, golNew, outNew, conversionNew := genGoPostItem(captureName, field)
		gocNew := ""
		if multipleCaptures {
			gocNew += indent + result + " = " + result + ".Conjoin(" + outNew + ")\n"
		} else {
			result = outNew
		}
		if n != "" {
			clDocNew += " " + n
		}
		captureVars = append(captureVars, captureVar)
		clType = append(clType, clNew)
		clTypeDoc = append(clTypeDoc, "^"+clDocNew)
		golType = append(golType, golNew)
		goCode = append(goCode, gocNew)
		if conversion == "" {
			conversion = conversionNew
		}
	}

	out = strings.Join(captureVars, ", ")
	if out != "" {
		out += " := "
	}

	cl = strings.Join(clType, " ")
	if len(clType) > 1 && cl != "" {
		cl = "[" + cl + "]"
	}

	clDoc = strings.Join(clTypeDoc, ", ")
	if len(clTypeDoc) > 1 && clDoc != "" {
		clDoc = "[" + clDoc + "]"
	}

	gol = strings.Join(golType, ", ")
	if len(golType) > 1 && gol != "" {
		gol = "(" + gol + ")"
	}

	goc = strings.Join(goCode, "")

	if multipleCaptures {
		goc = indent + result + " := EmptyVector()\n" + goc + indent + "return " + result + "\n"
		conversion = ""
	} else if useful {
		if goc == "" && result == resultName {
			out = "return " // No code generated, so no need to use intermediary
		} else {
			goc += indent + "return " + result + "\n"
		}
	} else {
		// An Error() method (from 'error' in an interface{}). Capture and wrap the string.
		out = result + " := "
		goc = indent + "return MakeString(" + result + ")\n"
	}

	return
}

func genGoPost(indent string, t *types.Signature) (goResultAssign, clojureReturnType, clojureReturnTypeForDoc, goReturnTypeForDoc, goReturnCode, conversion string) {
	res := t.Results()
	if res == nil || res.Len() == 0 {
		return
	}
	clojureReturnType, clojureReturnTypeForDoc, goReturnTypeForDoc, goReturnCode, goResultAssign, conversion = genGoPostList(indent, res)
	return
}
