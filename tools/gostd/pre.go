package main

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/astutils"
	"github.com/candid82/joker/tools/gostd/genutils"
	. "go/ast"
	"strconv"
	"strings"
)

func genTypePre(fn *FuncInfo, indent string, e Expr, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, goPreCode, cl2golParam, resExpr string) {
	ti := TypeInfoForExpr(e)
	resExpr = paramName

	pkgBaseName := fn.AddToImports(ti)
	goEffectiveBaseName := ti.GoEffectiveBaseName()
	if ti.IsArbitraryType() {
		// unsafe.ArbitraryType becomes interface{}, so omit the package name.
		goType = fmt.Sprintf(ti.GoPattern(), goEffectiveBaseName)
	} else {
		goType = fmt.Sprintf(ti.GoPattern(), genutils.CombineGoName(pkgBaseName, goEffectiveBaseName))
	}

	clType, clTypeDoc, goTypeDoc = ti.ClojureEffectiveName(), ti.ClojureNameDoc(e), ti.GoNameDoc(e)

	if fn.Fd == nil || fn.Fd.Recv != nil {
		apiImportName := fn.AddApiToImports(clType)
		cvt := ti.ConvertFromClojure()
		if cvt == "" {
			api := determineRuntime("ReceiverArgAs", "ReceiverArgAs_ns_", apiImportName, clType)
			goPreCode = fmt.Sprintf("%s := %s(%q, %q, _argList, %d)", paramName, api, "[RCVR]", paramName, argNum)
		} else {
			cvt = assertRuntime("", "", cvt)
			argNumAsString := strconv.Itoa(argNum)
			goPreCode = paramName + " := " +
				fmt.Sprintf(cvt,
					"SeqNth(_argList, "+argNumAsString+")",
					strconv.Quote("Arg["+argNumAsString+"] ("+paramName+"): %s"))
		}
	} else {
		if clType != "" {
			clType = assertRuntime("Extract", "Extract_ns_", clType)
		}
	}
	if ti.IsPassedByAddress() {
		if ti.IsAddressable() {
			cl2golParam = "*" + paramName
			resExpr = "*" + resExpr
		}
	} else {
		cl2golParam = paramName
	}

	return
}

func genGoPre(fn *FuncInfo, indent string, fl *FieldList, goFname string) (clojureParamList, clojureParamListDoc,
	clojureGoParams, goParamList, goParamListDoc, goPreCode, goParams, goFinalParams string, min, max int) {
	if fl == nil {
		return
	}
	fields := astutils.FlattenFieldList(fl)
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
		clType, clTypeDoc, goType, goTypeDoc, preCode, cl2golParam, resExpr := genTypePre(fn, indent, field.Field.Type, resVar, argNum)

		if clojureParamList != "" {
			clojureParamList += ", "
		}
		if clType != "" {
			clojureParamList += "^" + clType + " "
		}
		clojureParamList += genutils.ParamNameAsClojure(resVar)

		if clojureParamListDoc != "" {
			clojureParamListDoc += ", "
		}
		if clTypeDoc != "" {
			clojureParamListDoc += "^" + clTypeDoc + " "
		}
		clojureParamListDoc += genutils.ParamNameAsClojure(resVarDoc)

		if preCode != "" {
			if goPreCode != "" {
				goPreCode += "\n" + indent
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
		goParams += genutils.ParamNameAsGo(resVar)

		if goFinalParams != "" {
			goFinalParams += ", "
		}
		goFinalParams += genutils.ParamNameAsGo(resExpr)
	}
	clojureGoParams = "(" + clojureGoParams + ")"
	clojureParamListDoc = "[" + clojureParamListDoc + "]"
	if strings.Contains(goParamListDoc, " ") || strings.Contains(goParamListDoc, ",") {
		goParamListDoc = "(" + goParamListDoc + ")"
	}
	clojureParamListDoc = strings.ReplaceAll(clojureParamListDoc, "__", "")
	goParamListDoc = strings.ReplaceAll(goParamListDoc, "__", "")
	min = len(fields)
	max = len(fields)
	return
}

func genTypePreReceiver(fn *FuncInfo, e Expr, paramName string, argNum int) (goPreCode, resExpr string) {
	ti := TypeInfoForExpr(e)
	resExpr = paramName

	clType := ti.ClojureEffectiveName()

	apiImportName := fn.AddApiToImports(clType)
	api := determineRuntime("ReceiverArgAs", "ReceiverArgAs_ns_", apiImportName, clType)
	goPreCode = fmt.Sprintf("%s := %s(%q, %q, _argList, %d)", paramName, api, "[RCVR]", paramName, argNum)

	if ti.IsPassedByAddress() && ti.IsAddressable() {
		resExpr = "*" + resExpr
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

		if preCode != "" {
			if goPreCode != "" {
				goPreCode += "\n\t"
			}
			goPreCode += preCode
		}

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
