package main

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/genutils"
	"go/types"
	"strings"
)

func genTypePreFunc(fn *FuncInfo, ty types.Type, paramName string, isVariadic bool) (clType, clTypeDoc, goType, goTypeDoc, goPreCode, cl2golParam, newResVar string) {
	ti := TypeInfoForType(ty)

	pkgBaseName := fn.AddToImports(ti)
	goEffectiveBaseName := ti.GoEffectiveBaseName()
	if ti.IsArbitraryType() {
		// unsafe.ArbitraryType becomes interface{}, so omit the package name.
		goType = fmt.Sprintf(ti.GoPattern(), goEffectiveBaseName)
	} else {
		goType = fmt.Sprintf(ti.GoPattern(), genutils.CombineGoName(pkgBaseName, goEffectiveBaseName))
	}

	clType, clTypeDoc, goTypeDoc = ti.ClojureEffectiveName(), ti.ClojureNameDocForType(nil), ti.GoNameDocForType(ty)

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
	if isVariadic {
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
	tuple := fn.Ft.Params()
	args := tuple.Len()
	for argNum := 0; argNum < args; argNum++ {
		field := tuple.At(argNum)
		name := field.Name()
		resVar := ""
		resVarDoc := ""
		if name == "" {
			resVar = genutils.GenSym("__arg")
			resVarDoc = resVar
		} else {
			resVar = "_v_" + name
			resVarDoc = name
		}
		clType, clTypeDoc, goType, goTypeDoc, preCode, cl2golParam, newResVar := genTypePreFunc(fn, field.Type(), resVar, argNum == args-1 && fn.Ft.Variadic())

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

func genTypePreReceiver(fn *FuncInfo, ty types.Type, paramName string, argNum int, isVariadic bool) (goPreCode, resExpr string) {
	ti := TypeInfoForType(ty)
	resExpr = paramName

	clType := ti.ClojureEffectiveName()
	if isVariadic {
		if strings.Contains(clType, "/") {
			clType += "_s"
		} else {
			clType += "s"
		}
	}

	apiImportName := fn.AddApiToImports(clType)
	api := determineRuntime("ReceiverArgAs", "ReceiverArgAs_ns_", apiImportName, clType)
	goPreCode = fmt.Sprintf("\t%s := %s(%q, myName, _argList, %d)\n", paramName, api, paramName, argNum)

	if ti.IsPassedByAddress() {
		resExpr = "*" + resExpr
	}

	if isVariadic {
		resExpr += "..."
		if ti.IsPassedByAddress() {
			resExpr = fmt.Sprintf("ABEND748(cannot combine \"...\" with passed-by-reference types as in %q)", resExpr)
		}
	}

	return
}

func genGoPreReceiver(fn *FuncInfo) (goPreCode, goParams string, min, max int) {
	tuple := fn.Ft.Params()
	args := tuple.Len()
	for argNum := 0; argNum < args; argNum++ {
		field := tuple.At(argNum)
		name := field.Name()
		resVar := ""
		if name == "" {
			resVar = genutils.GenSym("__arg")
		} else {
			resVar = "_v_" + name
		}
		preCode, resExpr := genTypePreReceiver(fn, field.Type(), resVar, argNum, argNum == args-1 && fn.Ft.Variadic())

		goPreCode += preCode

		if goParams != "" {
			goParams += ", "
		}
		goParams += genutils.ParamNameAsGo(resExpr)
	}
	min = args
	max = args
	return
}
