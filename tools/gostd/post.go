package main

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/astutils"
	"github.com/candid82/joker/tools/gostd/genutils"
	. "go/ast"
	"strings"
)

func maybeNil(expr, captureName string) string {
	return "func () Object { if (" + expr + ") == nil { return NIL } else { return " + captureName + " } }()"
}

func genGoPostExpr(fn *FuncInfo, indent, captureName string, e Expr, onlyIf string) (cl, clDoc, gol, goc, out, conversion string) {
	ti := TypeInfoForExpr(e)
	if ti.AsClojureObject() == "" {
		out = fmt.Sprintf("MakeGoObject(%s)", captureName)
		cl = "GoObject"
	} else {
		out = "Make" + fmt.Sprintf(ti.AsClojureObject(), captureName, "")
		cl = ti.ArgExtractFunc()
		clDoc = ti.ArgClojureArgType()
	}
	if ti.IsNullable() {
		out = maybeNil(captureName, out)
	}
	cl = ti.ClojureName()
	clDoc = ti.ClojureNameDoc(e)
	gol = ti.GoNameDoc(e)
	conversion = ti.PromoteType()

	return
}

const resultName = "_res"

func genGoPostItem(fn *FuncInfo, indent, captureName string, f *Field, onlyIf string) (captureVar, cl, clDoc, gol, goc, out, conversion string, useful bool) {
	captureVar = captureName
	if captureName == "" || captureName == "_" {
		captureVar = genutils.GenSym(resultName)
	}
	cl, clDoc, gol, goc, out, conversion = genGoPostExpr(fn, indent, captureVar, f.Type, onlyIf)
	if captureName != "" && captureName != resultName {
		gol = genutils.ParamNameAsGo(captureName) + " " + gol
	}
	useful = genutils.ExprIsUseful(out)
	if !useful {
		captureVar = "_"
	}
	return
}

// Caller generates "outGOCALL;goc" while saving cl and gol for type info (they go into .joke as metadata and docstrings)
func genGoPostList(fn *FuncInfo, indent string, fl *FieldList) (cl, clDoc, gol, goc, out, conversion string) {
	useful := false
	captureVars := []string{}
	clType := []string{}
	clTypeDoc := []string{}
	golType := []string{}
	goCode := []string{}

	result := resultName
	fields := astutils.FlattenFieldList(fl)
	multipleCaptures := len(fields) > 1
	for _, field := range fields {
		n := ""
		if field.Name != nil {
			n = field.Name.Name
		}
		captureName := result
		if multipleCaptures {
			captureName = n
		}
		captureVar, clNew, clDocNew, golNew, gocNew, outNew, conversionNew, usefulItem := genGoPostItem(fn, indent, captureName, field.Field, "")
		useful = useful || usefulItem
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
		if useful {
			goc = indent + result + " := EmptyVector()\n" + goc + indent + "return " + result + "\n"
		} else {
			goc = indent + "ABEND123(post.go: no public information returned)\n"
		}
		conversion = ""
	} else {
		if goc == "" && result == resultName {
			out = "return " // No code generated, so no need to use intermediary
		} else {
			goc += indent + "return " + result + "\n"
		}
		if !useful {
			goc += indent + "ABEND124(post.go: no public information returned)\n"
		}
	}

	return
}

func genGoPost(fn *FuncInfo, indent string, t *FuncType) (goResultAssign, clojureReturnType, clojureReturnTypeForDoc, goReturnTypeForDoc, goReturnCode, conversion string) {
	fl := t.Results
	if fl == nil || fl.List == nil {
		return
	}
	clojureReturnType, clojureReturnTypeForDoc, goReturnTypeForDoc, goReturnCode, goResultAssign, conversion = genGoPostList(fn, indent, fl)
	return
}
