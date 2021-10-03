package main

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/genutils"
	"go/types"
	"os"
	"strings"
)

func genTypePreFunc(fn *FuncInfo, v *types.Var, paramName string, isVariadic, isNativeCodeNeeded bool) (clType, clTypeDoc, goAutoGenType, goNativeType, goTypeDoc, cl2golParam, newResVar string) {
	ty := v.Type()
	if isVariadic {
		ty = ty.(*types.Slice).Elem() // "...the last parameter [of a variadic signature] must be of unnamed slice type".
	}
	ti := TypeInfoForType(ty)

	ns := ti.Namespace()
	pkgAutoGenName := ""
	if ns != "" {
		clojureStdPath := generatedPkgPrefix + strings.ReplaceAll(ns, ".", "/")
		pkgAutoGenName = fn.ImportsAutoGen.AddPackage(clojureStdPath, ns, true, fn.Pos, "pre.go/genTypePreFunc")
	}

	pkgNativeName := ""
	if isNativeCodeNeeded {
		pkgNativeName = fn.ImportsNative.AddPackage(ti.GoPackage(), "", true, v.Pos(), "pre.go/genTypePreFunc")
	}

	goEffectiveBaseName := ti.GoEffectiveBaseName()
	if ti.IsArbitraryType() {
		// unsafe.ArbitraryType becomes interface{}, so omit the package name.
		goAutoGenType = fmt.Sprintf(ti.GoPattern(), goEffectiveBaseName)
		goNativeType = goAutoGenType
	} else {
		goAutoGenType = fmt.Sprintf(ti.GoPattern(), genutils.CombineGoName(pkgAutoGenName, goEffectiveBaseName))
		goNativeType = fmt.Sprintf(ti.GoPattern(), genutils.CombineGoName(pkgNativeName, goEffectiveBaseName))
	}

	clType, clTypeDoc, goTypeDoc = ti.ClojureEffectiveName(), ti.ClojureNameDocForType(v.Pkg()), ti.GoNameDocForType(v.Pkg())

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
		goAutoGenType = "..." + goAutoGenType
		goNativeType = "..." + goNativeType
		goTypeDoc = "..." + goTypeDoc
		cl2golParam += "..."
		newResVar += "..."
		if ti.IsPassedByAddress() {
			goAutoGenType = fmt.Sprintf("ABEND748(cannot combine \"...\" with passed-by-reference types as in %q)", goAutoGenType)
			goNativeType = goAutoGenType
		}
	}

	if strings.Contains(goNativeType, "Interface") {
		fmt.Fprintf(os.Stderr, "pre.go/genTypePreFunc(%s): %s (%s) and %s (%s)\n", fn.Name, goAutoGenType, pkgAutoGenName, goNativeType, pkgNativeName)
	}

	return
}

func genGoPreFunc(fn *FuncInfo, isNativeCodeNeeded bool) (clojureParamList, clojureParamListDoc,
	clojureGoParams, goAutoGenParamList, goNativeParamList, goParamListDoc, goParams string) {
	tuple := fn.Signature.Params()
	args := tuple.Len()
	isVariadic := fn.Signature.Variadic()
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
		clType, clTypeDoc, goAutoGenType, goNativeType, goTypeDoc, cl2golParam, newResVar := genTypePreFunc(fn, field, resVar, argNum == args-1 && isVariadic, isNativeCodeNeeded)

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

		if clojureGoParams != "" {
			clojureGoParams += ", "
		}
		clojureGoParams += cl2golParam

		if goAutoGenParamList != "" {
			goAutoGenParamList += ", "
		}
		goAutoGenParamList += genutils.ParamNameAsGo(resVar)
		if goNativeParamList != "" {
			goNativeParamList += ", "
		}
		goNativeParamList += genutils.ParamNameAsGo(resVar)
		if goAutoGenType != "" {
			goAutoGenParamList += " " + goAutoGenType
		}
		if goNativeType != "" {
			goNativeParamList += " " + goNativeType
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

func genTypePreReceiver(fn *FuncInfo, v *types.Var, paramName string, argNum int, isVariadic bool) (goPreCode, resExpr string) {
	ty := v.Type()
	if isVariadic {
		ty = ty.(*types.Slice).Elem() // "...the last parameter [of a variadic signature] must be of unnamed slice type".
	}
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
	tuple := fn.Signature.Params()
	args := tuple.Len()
	isVariadic := fn.Signature.Variadic()
	for argNum := 0; argNum < args; argNum++ {
		field := tuple.At(argNum)
		name := field.Name()
		resVar := ""
		if name == "" {
			resVar = genutils.GenSym("__arg")
		} else {
			resVar = "_v_" + name
		}
		preCode, resExpr := genTypePreReceiver(fn, field, resVar, argNum, argNum == args-1 && isVariadic)

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
