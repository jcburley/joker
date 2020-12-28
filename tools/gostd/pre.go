package main

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/astutils"
	"github.com/candid82/joker/tools/gostd/genutils"
	. "go/ast"
	"strconv"
	"strings"
)

func genTypePre(fn *FuncInfo, indent string, e Expr, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, goPreCode, cl2golParam string) {
	ti := TypeInfoForExpr(e)

	pkgBaseName := fn.AddToImports(ti)
	goName := fmt.Sprintf(ti.GoPattern(), genutils.CombineGoName(pkgBaseName, ti.GoBaseName()))

	clType, clTypeDoc, goType, goTypeDoc = ti.ClojureName(), ti.ClojureNameDoc(e), goName, ti.GoNameDoc(e)
	if clType != "" {
		clType = assertRuntime("Extract", "ExtractGoObject", clType)
	}
	if fn.Fd == nil || fn.Fd.Recv != nil {
		cvt := ti.ConvertFromClojure()
		if cvt == "" {
			cvt = fmt.Sprintf("%%s.(GoObject).O.(%s)%%.s", goName)
		} else {
			cvt = assertRuntime("", "", cvt)
		}
		argNumAsString := strconv.Itoa(argNum)
		goPreCode = paramName + " := " + fmt.Sprintf(cvt, "SeqNth(_argList, "+argNumAsString+")", strconv.Quote("Arg["+argNumAsString+"] ("+paramName+"): %s"))
	}
	if ti.IsPassedByAddress() {
		cl2golParam = "*" + paramName
	} else {
		cl2golParam = paramName
	}

	return
}

func genGoPre(fn *FuncInfo, indent string, fl *FieldList, goFname string) (clojureParamList, clojureParamListDoc,
	clojureGoParams, goParamList, goParamListDoc, goPreCode, goParams string, min, max int) {
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
		clType, clTypeDoc, goType, goTypeDoc, preCode, cl2golParam := genTypePre(fn, indent, field.Field.Type, resVar, argNum)
		if goType == "unsafe.ArbitraryType" {
			goType = "interface{}"
			clType = "GoObject"
			clTypeDoc = "GoObject"
		}

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
