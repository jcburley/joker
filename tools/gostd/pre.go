package main

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/astutils"
	"github.com/candid82/joker/tools/gostd/genutils"
	. "go/ast"
	"strconv"
	"strings"
)

func genTypePreFunc(fn *FuncInfo, e Expr, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, goPreCode, cl2golParam, newResVar string) {
	actualType, isEllipsis := e.(*Ellipsis)
	if isEllipsis {
		e = actualType.Elt
	}

	ti := TypeInfoForExpr(e)

	pkgBaseName := fn.AddToImports(ti)
	goEffectiveBaseName := ti.GoEffectiveBaseName()
	if ti.IsArbitraryType() {
		// unsafe.ArbitraryType becomes interface{}, so omit the package name.
		goType = fmt.Sprintf(ti.GoPattern(), goEffectiveBaseName)
	} else {
		goType = fmt.Sprintf(ti.GoPattern(), genutils.CombineGoName(pkgBaseName, goEffectiveBaseName))
	}

	clType, clTypeDoc, goTypeDoc = ti.ClojureExtractString(), ti.ClojureNameDoc(e), ti.GoNameDoc(e)

	if clType != "" {
		clType = "^" + assertRuntime("Extract", "Extract_ns_", clType)
		clTypeDoc = "^" + clTypeDoc
	}

	if ti.IsPassedByAddress() {
		cl2golParam = "*" + paramName
	} else {
		cl2golParam = paramName
	}

	newResVar = paramName
	if isEllipsis {
		clType = "& " + clType
		clTypeDoc = "& " + clTypeDoc
		goType = "..." + goType
		goTypeDoc = "..." + goTypeDoc
		cl2golParam += "..."
		newResVar += "..."
		if ti.IsPassedByAddress() {
			goType = fmt.Sprintf("ABEND748(cannot combine \"...\" with passed-by-reference types as in %q)", goType)
		}
	}

	return
}

func genGoPreFunc(fn *FuncInfo) (clojureParamList, clojureParamListDoc,
	clojureGoParams, goParamList, goParamListDoc, goPreCode, goParams string) {
	if fn.Ft.Params == nil {
		return
	}
	fields := astutils.FlattenFieldList(fn.Ft.Params)
	for argNum, field := range fields {
		p := field.Name
		resVar := ""
		resVarDoc := ""
		if p == nil {
			resVar = genutils.GenSym("__arg")
			resVarDoc = resVar
		} else {
			resVar = "_v_" + p.Name
			resVarDoc = p.Name
		}
		clType, clTypeDoc, goType, goTypeDoc, preCode, cl2golParam, newResVar := genTypePreFunc(fn, field.Field.Type, resVar, argNum)

		if clojureParamList != "" {
			clojureParamList += ", "
		}
		if clType != "" {
			clojureParamList += clType + " "
		}
		clojureParamList += genutils.ParamNameAsClojure(resVar)

		if clojureParamListDoc != "" {
			clojureParamListDoc += ", "
		}
		if clTypeDoc != "" {
			clojureParamListDoc += clTypeDoc + " "
		}
		clojureParamListDoc += genutils.ParamNameAsClojure(resVarDoc)

		if preCode != "" {
			if goPreCode != "" {
				goPreCode += "\n\t"
			}
			goPreCode += preCode
		}

		if clojureGoParams != "" {
			clojureGoParams += ", "
		}
		clojureGoParams += cl2golParam

		if goParamList != "" {
			goParamList += ", "
		}
		goParamList += genutils.ParamNameAsGo(resVar)
		if goType != "" {
			goParamList += " " + goType
		}

		if goParamListDoc != "" {
			goParamListDoc += ", "
		}
		goParamListDoc += genutils.ParamNameAsGo(resVarDoc)
		if goTypeDoc != "" {
			goParamListDoc += " " + goTypeDoc
		}

		if goParams != "" {
			goParams += ", "
		}
		goParams += genutils.ParamNameAsGo(newResVar)
	}
	clojureGoParams = "(" + clojureGoParams + ")"
	clojureParamListDoc = "[" + clojureParamListDoc + "]"
	if strings.Contains(goParamListDoc, " ") || strings.Contains(goParamListDoc, ",") {
		goParamListDoc = "(" + goParamListDoc + ")"
	}
	clojureParamListDoc = strings.ReplaceAll(clojureParamListDoc, "__", "")
	goParamListDoc = strings.ReplaceAll(goParamListDoc, "__", "")
	return
}

func genTypePreReceiver(fn *FuncInfo, e Expr, paramName string, argNum int) (goPreCode, resExpr string) {
	actualType, isEllipsis := e.(*Ellipsis)
	if isEllipsis {
		e = actualType.Elt
	}

	ti := TypeInfoForExpr(e)
	resExpr = paramName

	clType := ti.GoApiString(isEllipsis)

	apiImportName := fn.AddApiToImports(clType)
	api := determineRuntime("ReceiverArgAs_", "ReceiverArgAs_ns_", apiImportName, clType)
	goPreCode = fmt.Sprintf("\t%s := %s(%q, myName, _argList, %d)\n", paramName, api, paramName, argNum)

	if ti.IsPassedByAddress() {
		resExpr = "*" + resExpr
	}

	if isEllipsis {
		resExpr += "..."
		if ti.IsPassedByAddress() {
			resExpr = fmt.Sprintf("ABEND748(cannot combine \"...\" with passed-by-reference types as in %q)", resExpr)
		}
	}

	return
}

func genGoPreReceiver(fn *FuncInfo) (goPreCode, goParams string, min, max int) {
	if fn.Ft.Params == nil {
		return
	}
	fields := astutils.FlattenFieldList(fn.Ft.Params)
	for argNum, field := range fields {
		p := field.Name
		resVar := ""
		if p == nil {
			resVar = genutils.GenSym("__arg")
		} else {
			resVar = "_v_" + p.Name
		}
		preCode, resExpr := genTypePreReceiver(fn, field.Field.Type, resVar, argNum)

		goPreCode += preCode

		if goParams != "" {
			goParams += ", "
		}
		goParams += genutils.ParamNameAsGo(resExpr)
	}
	min = len(fields)
	max = len(fields)
	return
}

func paramsAsSymbolVec(fl *FieldList) string {
	genutils.GenSymReset()
	fields := astutils.FlattenFieldList(fl)
	var syms []string
	for _, field := range fields {
		var p string
		if field.Name == nil {
			p = genutils.GenSym("arg")
		} else {
			p = field.Name.Name
		}
		syms = append(syms, "MakeSymbol("+strconv.Quote(p)+")")
	}
	return strings.Join(syms, ", ")
}
