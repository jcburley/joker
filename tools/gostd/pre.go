package main

import (
	"fmt"
	. "go/ast"
	"strings"
)

func genGoPreArray(fn *funcInfo, indent string, e *ArrayType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	el := e.Elt
	len := e.Len
	clType, clTypeDoc, goType, goTypeDoc, _, cl2golParam = genTypePre(fn, indent, el, paramName)
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

func genGoPreStar(fn *funcInfo, indent string, e *StarExpr, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	el := e.X
	clType, clTypeDoc, goType, goTypeDoc, _, cl2golParam = genTypePre(fn, indent, el, paramName)
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

func genGoPreSelected(fn *funcInfo, indent, fullPkgName, baseTypeName, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	clType, clTypeDoc, goType, goTypeDoc = fullPkgNameAsGoType(fn, fullPkgName, baseTypeName)
	cl2golParam = "*" + paramName // genType generates functions that return pointers to objects, to avoid copying-sync.Mutex issues
	return
}

func genGoPreNamed(fn *funcInfo, indent, typeName, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	return genGoPreSelected(fn, indent, fn.sourceFile.pkgDirUnix, typeName, paramName)
}

func genGoPreSelector(fn *funcInfo, indent string, e *SelectorExpr, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	pkgName := e.X.(*Ident).Name
	fullPathUnix := unix(fileAt(e.Pos()))
	referringFile := strings.TrimPrefix(fullPathUnix, fn.sourceFile.rootUnix+"/")
	rf, ok := goFiles[referringFile]
	if !ok {
		panic(fmt.Sprintf("genGoPreSelector: could not find referring file %s for file %s at %s",
			referringFile, fullPathUnix, whereAt(e.Pos())))
	}
	if fullPkgName, found := (*rf.spaces)[pkgName]; found {
		return genGoPreSelected(fn, indent, fullPkgName, e.Sel.Name, paramName)
	}
	panic(fmt.Sprintf("processing %s for %s: could not find %s in %s",
		whereAt(e.Pos()), whereAt(fn.fd.Pos()), pkgName, fn.sourceFile.name))
}

func genGoPreEllipsis(fn *funcInfo, indent string, e *Ellipsis, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	el := e.Elt
	clType, clTypeDoc, goType, goTypeDoc, _, cl2golParam = genTypePre(fn, indent, el, paramName)
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

func genGoPreFunc(fn *funcInfo, indent string, e *FuncType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
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

func genGoPreInterface(fn *funcInfo, indent string, e *InterfaceType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
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

func genGoPreMap(fn *funcInfo, indent string, e *MapType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
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

func genGoPreChan(fn *funcInfo, indent string, e *ChanType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
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

func genTypePre(fn *funcInfo, indent string, e Expr, paramName string) (clType, clTypeDoc, goType, goTypeDoc, goPreCode, cl2golParam string) {
	clType = fmt.Sprintf("ABEND881(pre.go: unrecognized Expr type %T at: %s)", e, unix(whereAt(e.Pos())))
	goType = fmt.Sprintf("ABEND882(pre.go: unrecognized Expr type %T at: %s)", e, unix(whereAt(e.Pos())))
	cl2golParam = paramName
	switch v := e.(type) {
	case *Ident:
		goType = v.Name
		ti := toGoExprInfo(fn.sourceFile, &e)
		clType = ti.argClojureArgType
		if ti.sourceFile != nil { // not a builtin
			if isPrivate(v.Name) {
				clType = fmt.Sprintf("ABEND044(pre.go: unsupported built-in type %s)", v.Name)
				clTypeDoc = v.Name
			} else {
				clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreNamed(fn, indent, v.Name, paramName)
			}
		}
		if clTypeDoc == "" {
			clTypeDoc = clType
		}
		if goTypeDoc == "" {
			goTypeDoc = goType
		}
	case *ArrayType:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreArray(fn, indent, v, paramName)
	case *StarExpr:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreStar(fn, indent, v, paramName)
	case *SelectorExpr:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreSelector(fn, indent, v, paramName)
	case *Ellipsis:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreEllipsis(fn, indent, v, paramName)
	case *FuncType:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreFunc(fn, indent, v, paramName)
	case *InterfaceType:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreInterface(fn, indent, v, paramName)
	case *MapType:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreMap(fn, indent, v, paramName)
	case *ChanType:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreChan(fn, indent, v, paramName)
	}
	return
}

func genGoPre(fn *funcInfo, indent string, fl *FieldList, goFname string) (clojureParamList, clojureParamListDoc,
	clojureGoParams, goParamList, goParamListDoc, goPreCode, goParams string) {
	if fl == nil {
		return
	}
	fields := flattenFieldList(fl)
	for _, field := range fields {
		p := field.name
		resVar := ""
		if p == nil {
			resVar = genSym("")
		} else {
			resVar = p.Name
		}
		clType, clTypeDoc, goType, goTypeDoc, preCode, cl2golParam := genTypePre(fn, indent, field.field.Type, "_"+resVar)

		if clojureParamList != "" {
			clojureParamList += ", "
		}
		if clType != "" {
			clojureParamList += "^" + clType + " "
		}
		clojureParamList += "_" + paramNameAsClojure(resVar)

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
	return
}
