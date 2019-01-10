package main

import (
	"fmt"
	. "go/ast"
	"strings"
)

func genGoPreArray(fn *funcInfo, indent string, e *ArrayType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	el := e.Elt
	len := e.Len
	clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genTypePre(fn, indent, el, paramName)
	runtime := "ConvertToArrayOf" + goType
	cl2golParam = runtime + "(" + cl2golParam + ")"
	if len != nil {
		cl2golParam = "ABEND901(specific-length arrays not supported: " + cl2golParam + ")"
	} else if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(cl2golParam, "ABEND") {
			cl2golParam = "ABEND902(custom-runtime routine not implemented: " + cl2golParam + ")"
		}
	} else if _, ok := el.(*Ident); !ok {
		cl2golParam = "ABEND910(arrays of things other than identifiers not supported: " + cl2golParam + ")"
	}
	clType = "Object"
	clTypeDoc = "(vector-of " + clTypeDoc + ")"
	goType = "[]" + goType
	goTypeDoc = goType
	return
}

func genGoPreStar(fn *funcInfo, indent string, e *StarExpr, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	el := e.X
	clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genTypePre(fn, indent, el, paramName)
	runtime := "ConvertToIndirectOf" + goType
	cl2golParam = runtime + "(" + cl2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(cl2golParam, "ABEND") {
			cl2golParam = "ABEND903(custom-runtime routine not implemented: " + cl2golParam + ")"
		}
	}
	clType = "Object"
	clTypeDoc = "(atom-of " + clTypeDoc + ")"
	goType = "*" + goType
	goTypeDoc = goType
	return
}

func genGoPreSelected(fn *funcInfo, indent, fullTypeName, baseTypeName, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	clType = "Native"
	clTypeDoc = fullTypeNameAsClojure(fullTypeName)
	goType = "_" + fn.sourceFile.pkgName + "." + baseTypeName
	goTypeDoc = fullTypeName
	cl2golParam = paramName
	return
}

func genGoPreNamed(fn *funcInfo, indent, typeName, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	return genGoPreSelected(fn, indent, fn.sourceFile.pkgDirUnix+"."+typeName, typeName, paramName)
}

func genGoPreSelector(fn *funcInfo, indent string, e *SelectorExpr, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	pkgName := e.X.(*Ident).Name
	referringFile := strings.TrimPrefix(fileAt(e.Pos()), fn.sourceFile.rootUnix+"/")
	rf, ok := goFiles[referringFile]
	if !ok {
		panic(fmt.Sprintf("genGoPreSelector: could not find referring file %s for expression at %s",
			referringFile, whereAt(e.Pos())))
	}
	if fullPkgName, found := (*rf.spaces)[pkgName]; found {
		fullTypeName := fullPkgName + "." + e.Sel.Name
		return genGoPreSelected(fn, indent, fullTypeName, e.Sel.Name, paramName)
	}
	panic(fmt.Sprintf("processing %s for %s: could not find %s in %s",
		whereAt(e.Pos()), whereAt(fn.fd.Pos()), pkgName, fn.sourceFile.name))
}

func genGoPreEllipsis(fn *funcInfo, indent string, e *Ellipsis, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	el := e.Elt
	clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genTypePre(fn, indent, el, paramName)
	runtime := "ConvertToEllipsisHaHa" + goType
	cl2golParam = runtime + "(" + cl2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(cl2golParam, "ABEND") {
			cl2golParam = "ABEND905(custom-runtime routine not implemented: " + cl2golParam + ")"
		}
	}
	clTypeDoc = "(ellipsis-somehow " + clType + ")"
	goType = "..." + goType
	goTypeDoc = goType
	return
}

func genGoPreFunc(fn *funcInfo, indent string, e *FuncType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	clType = "fn"
	goType = "func"
	runtime := "ConvertToFuncTypeHaHa" + goType
	cl2golParam = runtime + "(" + cl2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(cl2golParam, "ABEND") {
			cl2golParam = "ABEND906(custom-runtime routine not implemented: " + cl2golParam + ")"
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
			cl2golParam = "ABEND907(custom-runtime routine not implemented: " + cl2golParam + ")"
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
			cl2golParam = "ABEND908(custom-runtime routine not implemented: " + cl2golParam + ")"
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
			cl2golParam = "ABEND909(custom-runtime routine not implemented: " + cl2golParam + ")"
		}
	}
	clTypeDoc = clType
	goTypeDoc = goType
	return
}

func genTypePre(fn *funcInfo, indent string, e Expr, paramName string) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	clType = fmt.Sprintf("ABEND881(unrecognized Expr type %T at: %s)", e, unix(whereAt(e.Pos())))
	goType = fmt.Sprintf("ABEND882(unrecognized Expr type %T at: %s)", e, unix(whereAt(e.Pos())))
	cl2golParam = paramName
	switch v := e.(type) {
	case *Ident:
		goType = v.Name
		clType = fmt.Sprintf("ABEND885(unrecognized type %s at: %s)", v.Name, unix(whereAt(e.Pos())))
		switch v.Name {
		case "string":
			clType = "String"
		case "int":
			clType = "Int"
		case "byte":
			clType = "Byte"
		case "bool":
			clType = "Bool"
		case "int16":
			clType = "Int16"
		case "uint":
			clType = "UInt"
		case "uint16":
			clType = "UInt16"
		case "int32":
			clType = "Int32"
		case "uint32":
			clType = "UInt32"
		case "int64":
			clType = "Int64"
		case "error":
		default:
			if isPrivate(v.Name) {
				clType = fmt.Sprintf("ABEND044(unsupported built-in type %s)", v.Name)
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
	for _, f := range fl.List {
		for _, p := range f.Names {
			clType, clTypeDoc, goType, goTypeDoc, cl2golParam := genTypePre(fn, indent, f.Type, "_"+p.Name)

			if clojureParamList != "" {
				clojureParamList += ", "
			}
			if clType != "" {
				clojureParamList += "^" + clType + " "
			}
			clojureParamList += "_" + paramNameAsClojure(p.Name)

			if clojureParamListDoc != "" {
				clojureParamListDoc += ", "
			}
			if clTypeDoc != "" {
				clojureParamListDoc += "^" + clTypeDoc + " "
			}
			clojureParamListDoc += paramNameAsClojure(p.Name)

			if clojureGoParams != "" {
				clojureGoParams += ", "
			}
			clojureGoParams += cl2golParam

			if goParamList != "" {
				goParamList += ", "
			}
			goParamList += paramNameAsGo(p.Name)
			if goType != "" {
				goParamList += " " + goType
			}

			if goParamListDoc != "" {
				goParamListDoc += ", "
			}
			goParamListDoc += paramNameAsGo(p.Name)
			if goTypeDoc != "" {
				goParamListDoc += " " + goTypeDoc
			}

			if goParams != "" {
				goParams += ", "
			}
			goParams += paramNameAsGo(p.Name)
		}
	}
	clojureGoParams = "(" + clojureGoParams + ")"
	clojureParamListDoc = "[" + clojureParamListDoc + "]"
	if strings.Contains(goParamListDoc, " ") || strings.Contains(goParamListDoc, ",") {
		goParamListDoc = "(" + goParamListDoc + ")"
	}
	return
}
