package main

import (
	"fmt"
	. "go/ast"
	"strings"
)

func genGoPreArray(fn *funcInfo, indent string, e *ArrayType, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	el := e.Elt
	len := e.Len
	clType, clTypeDoc, goType, goTypeDoc, _, cl2golParam = genTypePre(fn, indent, el, paramName, argNum)
	runtime := "ConvertToArrayOf" + goType
	cl2golParam = runtime + "(" + cl2golParam + ")"
	if len != nil {
		cl2golParam = "ABEND901(pre.go: specific-length arrays not supported: " + cl2golParam + ")"
	} else if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(cl2golParam, "ABEND") {
			cl2golParam = "ABEND902(pre.go: custom-runtime routine not implemented: " + cl2golParam + ")"
		}
	} else if _, ok := el.(*Ident); !ok {
		cl2golParam = "ABEND910(pre.go: arrays of things other than identifiers not supported: " + cl2golParam + ")"
	}
	clType = "Object"
	clTypeDoc = "(vector-of " + clTypeDoc + ")"
	goType = "[]" + goType
	goTypeDoc = "[]" + goTypeDoc
	return
}

func genGoPreStar(fn *funcInfo, indent string, e *StarExpr, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	el := e.X
	clType, clTypeDoc, goType, goTypeDoc, _, cl2golParam = genTypePre(fn, indent, el, paramName, argNum)
	if cl2golParam[0] == '*' {
		cl2golParam = cl2golParam[1:]
	} else {
		runtime := "ConvertToIndirectOf" + goType
		cl2golParam = runtime + "(" + cl2golParam + ")"
		if _, ok := customRuntimeImplemented[runtime]; !ok {
			if !strings.Contains(cl2golParam, "ABEND") {
				cl2golParam = "ABEND903(pre.go: custom-runtime routine not implemented: " + cl2golParam + ")"
			}
		}
		clType = "Object"
	}
	clTypeDoc = "(atom-of " + clTypeDoc + ")"
	goType = "*" + goType
	goTypeDoc = "*" + goTypeDoc
	return
}

func genGoPreSelected(fn *funcInfo, indent, fullPkgName, baseTypeName, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	clType, clTypeDoc, goType, goTypeDoc = fullPkgNameAsGoType(fn.sourceFile, fullPkgName, baseTypeName)
	cl2golParam = "*" + paramName // genType generates functions that return pointers to objects, to avoid copying-sync.Mutex issues
	return
}

func genGoPreNamed(fn *funcInfo, indent, typeName, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	return genGoPreSelected(fn, indent, fn.sourceFile.pkgDirUnix, typeName, paramName, argNum)
}

func genGoPreSelector(fn *funcInfo, indent string, e *SelectorExpr, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	pkgName := e.X.(*Ident).Name
	fullPathUnix := unix(fileAt(e.Pos()))
	referringFile := strings.TrimPrefix(fullPathUnix, fn.sourceFile.rootUnix+"/")
	rf, ok := goFiles[referringFile]
	if !ok {
		panic(fmt.Sprintf("genGoPreSelector: could not find referring file %s for file %s at %s",
			referringFile, fullPathUnix, whereAt(e.Pos())))
	}
	if fullPkgName, found := (*rf.spaces)[pkgName]; found {
		return genGoPreSelected(fn, indent, fullPkgName, e.Sel.Name, paramName, argNum)
	}
	panic(fmt.Sprintf("processing %s for %s: could not find %s in %s",
		whereAt(e.Pos()), whereAt(fn.fd.Pos()), pkgName, fn.sourceFile.name))
}

func genGoPreEllipsis(fn *funcInfo, indent string, e *Ellipsis, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	el := e.Elt
	clType, clTypeDoc, goType, goTypeDoc, _, cl2golParam = genTypePre(fn, indent, el, paramName, argNum)
	runtime := "ConvertToEllipsisHaHa" + goType
	cl2golParam = runtime + "(" + cl2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(cl2golParam, "ABEND") {
			cl2golParam = "ABEND905(pre.go: custom-runtime routine not implemented: " + cl2golParam + ")"
		}
	}
	clTypeDoc = "(ellipsis-somehow " + clType + ")"
	goType = "..." + goType
	goTypeDoc = "..." + goTypeDoc
	return
}

func genGoPreFunc(fn *funcInfo, indent string, e *FuncType, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	clType = "fn"
	goType = "func"
	runtime := "ConvertToFuncTypeHaHa" + goType
	cl2golParam = runtime + "(" + cl2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(cl2golParam, "ABEND") {
			cl2golParam = "ABEND906(pre.go: custom-runtime routine not implemented: " + cl2golParam + ")"
		}
	}
	clTypeDoc = clType
	goTypeDoc = goType
	return
}

func genGoPreInterface(fn *funcInfo, indent string, e *InterfaceType, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	clType = "<protocol-or-something>"
	goType = "interface {}"
	runtime := "ConvertToInterfaceTypeHaHa"
	cl2golParam = runtime + "(" + cl2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(cl2golParam, "ABEND") {
			cl2golParam = "ABEND907(pre.go: custom-runtime routine not implemented: " + cl2golParam + ")"
		}
	}
	clTypeDoc = clType
	goTypeDoc = goType
	return
}

func genGoPreMap(fn *funcInfo, indent string, e *MapType, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	clType = "{}"
	goType = "map[]"
	runtime := "ConvertToMapTypeHaHa"
	cl2golParam = runtime + "(" + cl2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(cl2golParam, "ABEND") {
			cl2golParam = "ABEND908(pre.go: custom-runtime routine not implemented: " + cl2golParam + ")"
		}
	}
	clTypeDoc = clType
	goTypeDoc = goType
	return
}

func genGoPreChan(fn *funcInfo, indent string, e *ChanType, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	clType = "<no-idea-about-chan-yet>"
	goType = "<-chan"
	runtime := "ConvertToChanTypeHaHa"
	cl2golParam = runtime + "(" + cl2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(cl2golParam, "ABEND") {
			cl2golParam = "ABEND909(pre.go: custom-runtime routine not implemented: " + cl2golParam + ")"
		}
	}
	clTypeDoc = clType
	goTypeDoc = goType
	return
}

func genTypePre(fn *funcInfo, indent string, e Expr, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, goPreCode, cl2golParam string) {
	cl2golParam = paramName
	extractParam := ""
	ti := toGoExprInfo(fn.sourceFile, &e)
	goTypeDoc = ti.goFullName()
	if ti.isLocallyDefined(fn.sourceFile) {
		goType = ti.goBaseName()
	} else {
		goType = goTypeDoc
	}
	if ti.sourceFile != nil && fn.sourceFile.pkgDirUnix != ti.sourceFile.pkgDirUnix {
		goType = "ABEND986(genutils.go: imports not yet supported: " + goTypeDoc + ")"
		goTypeDoc = goType
	}
	clType = ti.fullClojureName
	clTypeDoc = clType
	if fn.fd.Recv != nil {
		argType := ti.argExtractFunc
		if argType != "" {
			extractParam = fmt.Sprintf("ExtractGo%s(\"%s\", \"%s\", _argList, %d)", typeToGoExtractFuncName(argType), fn.docName, paramName, argNum)
		}
		goPreCode = paramName + " := " + extractParam
	}
	if fn.fd.Recv != nil && goPreCode == "" {
		goPreCode = fmt.Sprintf("ABEND644(pre.go: unsupported built-in type %T for %s at: %s)", e, paramName, unix(whereAt(e.Pos())))
	}
	return
}

func genGoPre(fn *funcInfo, indent string, fl *FieldList, goFname string) (clojureParamList, clojureParamListDoc,
	clojureGoParams, goParamList, goParamListDoc, goPreCode, goParams string, min, max int) {
	if fl == nil {
		return
	}
	fields := flattenFieldList(fl)
	for argNum, field := range fields {
		p := field.name
		resVar := ""
		if p == nil {
			resVar = genSym("")
		} else {
			resVar = p.Name
		}
		clType, clTypeDoc, goType, goTypeDoc, preCode, cl2golParam := genTypePre(fn, indent, field.field.Type, resVar, argNum)

		if clojureParamList != "" {
			clojureParamList += ", "
		}
		if clType != "" {
			clojureParamList += "^" + clType + " "
		}
		clojureParamList += paramNameAsClojure(resVar)

		if clojureParamListDoc != "" {
			clojureParamListDoc += ", "
		}
		if clTypeDoc != "" {
			clojureParamListDoc += "^" + clTypeDoc + " "
		}
		clojureParamListDoc += paramNameAsClojure(resVar)

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
		goParamList += paramNameAsGo(resVar)
		if goType != "" {
			goParamList += " " + goType
		}

		if goParamListDoc != "" {
			goParamListDoc += ", "
		}
		goParamListDoc += paramNameAsGo(resVar)
		if goTypeDoc != "" {
			goParamListDoc += " " + goTypeDoc
		}

		if goParams != "" {
			goParams += ", "
		}
		goParams += paramNameAsGo(resVar)
	}
	clojureGoParams = "(" + clojureGoParams + ")"
	clojureParamListDoc = "[" + clojureParamListDoc + "]"
	if strings.Contains(goParamListDoc, " ") || strings.Contains(goParamListDoc, ",") {
		goParamListDoc = "(" + goParamListDoc + ")"
	}
	min = len(fields)
	max = len(fields)
	return
}
