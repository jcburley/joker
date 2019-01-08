package main

import (
	"fmt"
	. "go/ast"
	"strings"
)

func genGoPreArray(fn *funcInfo, indent string, e *ArrayType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	el := e.Elt
	len := e.Len
	clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genTypePre(fn, indent, el, paramName)
	runtime := "ConvertToArrayOf" + goType
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if len != nil {
		jok2golParam = "ABEND901(specific-length arrays not supported: " + jok2golParam + ")"
	} else if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND902(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	} else if _, ok := el.(*Ident); !ok {
		jok2golParam = "ABEND910(arrays of things other than identifiers not supported: " + jok2golParam + ")"
	}
	clType = "Object"
	clTypeDoc = "(vector-of " + clTypeDoc + ")"
	goType = "[]" + goType
	goTypeDoc = goType
	return
}

func genGoPreStar(fn *funcInfo, indent string, e *StarExpr, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	el := e.X
	clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genTypePre(fn, indent, el, paramName)
	runtime := "ConvertToIndirectOf" + goType
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND903(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	}
	clType = "Object"
	clTypeDoc = "(atom-of " + clTypeDoc + ")"
	goType = "*" + goType
	goTypeDoc = goType
	return
}

func genGoPreSelected(fn *funcInfo, indent, fullTypeName, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	clType = fullTypeNameAsClojure(fullTypeName)
	clTypeDoc = clType
	goType = fullTypeName
	goTypeDoc = goType
	runtime := "ConvertTo" + fullTypeName
	jok2golParam = runtime + "(" + paramName + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND904(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	} else if _, ok := types[fullTypeName]; !ok {
		jok2golParam = fmt.Sprintf("ABEND045(cannot find typename %s)", fullTypeName)
	}
	return
}

func genGoPreNamed(fn *funcInfo, indent, typeName, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	return genGoPreSelected(fn, indent, fn.sourceFile.pkgDirUnix+"."+typeName, paramName)
}

func genGoPreSelector(fn *funcInfo, indent string, e *SelectorExpr, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	pkgName := e.X.(*Ident).Name
	referringFile := strings.TrimPrefix(fileAt(e.Pos()), fn.sourceFile.rootUnix+"/")
	rf, ok := goFiles[referringFile]
	if !ok {
		panic(fmt.Sprintf("genGoPreSelector: could not find referring file %s for expression at %s",
			referringFile, whereAt(e.Pos())))
	}
	if fullPkgName, found := (*rf.spaces)[pkgName]; found {
		fullTypeName := fullPkgName + "." + e.Sel.Name
		return genGoPreSelected(fn, indent, fullTypeName, paramName)
	}
	panic(fmt.Sprintf("processing %s for %s: could not find %s in %s",
		whereAt(e.Pos()), whereAt(fn.fd.Pos()), pkgName, fn.sourceFile.name))
}

func genGoPreEllipsis(fn *funcInfo, indent string, e *Ellipsis, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	el := e.Elt
	clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genTypePre(fn, indent, el, paramName)
	runtime := "ConvertToEllipsisHaHa" + goType
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND905(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	}
	clTypeDoc = "(ellipsis-somehow " + clType + ")"
	goType = "..." + goType
	goTypeDoc = goType
	return
}

func genGoPreFunc(fn *funcInfo, indent string, e *FuncType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	clType = "fn"
	goType = "func"
	runtime := "ConvertToFuncTypeHaHa" + goType
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND906(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	}
	clTypeDoc = clType
	goTypeDoc = goType
	return
}

func genGoPreInterface(fn *funcInfo, indent string, e *InterfaceType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	clType = "<protocol-or-something>"
	goType = "interface {}"
	runtime := "ConvertToInterfaceTypeHaHa"
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND907(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	}
	clTypeDoc = clType
	goTypeDoc = goType
	return
}

func genGoPreMap(fn *funcInfo, indent string, e *MapType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	clType = "{}"
	goType = "map[]"
	runtime := "ConvertToMapTypeHaHa"
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND908(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	}
	clTypeDoc = clType
	goTypeDoc = goType
	return
}

func genGoPreChan(fn *funcInfo, indent string, e *ChanType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	clType = "<no-idea-about-chan-yet>"
	goType = "<-chan"
	runtime := "ConvertToChanTypeHaHa"
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND909(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	}
	clTypeDoc = clType
	goTypeDoc = goType
	return
}

func genTypePre(fn *funcInfo, indent string, e Expr, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	clType = fmt.Sprintf("ABEND881(unrecognized Expr type %T at: %s)", e, unix(whereAt(e.Pos())))
	goType = fmt.Sprintf("ABEND882(unrecognized Expr type %T at: %s)", e, unix(whereAt(e.Pos())))
	jok2golParam = paramName
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
				clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreNamed(fn, indent, v.Name, paramName)
			}
		}
		if clTypeDoc == "" {
			clTypeDoc = clType
		}
		if goTypeDoc == "" {
			goTypeDoc = goType
		}
	case *ArrayType:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreArray(fn, indent, v, paramName)
	case *StarExpr:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreStar(fn, indent, v, paramName)
	case *SelectorExpr:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreSelector(fn, indent, v, paramName)
	case *Ellipsis:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreEllipsis(fn, indent, v, paramName)
	case *FuncType:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreFunc(fn, indent, v, paramName)
	case *InterfaceType:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreInterface(fn, indent, v, paramName)
	case *MapType:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreMap(fn, indent, v, paramName)
	case *ChanType:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreChan(fn, indent, v, paramName)
	}
	return
}

func genGoPre(fn *funcInfo, indent string, fl *FieldList, goFname string) (jokerParamList, jokerParamListDoc,
	jokerGoParams, goParamList, goParamListDoc, goPreCode, goParams string) {
	if fl == nil {
		return
	}
	for _, f := range fl.List {
		for _, p := range f.Names {
			clType, clTypeDoc, goType, goTypeDoc, jok2golParam := genTypePre(fn, indent, f.Type, "_"+p.Name)

			if jokerParamList != "" {
				jokerParamList += ", "
			}
			if clType != "" {
				jokerParamList += "^" + clType + " "
			}
			jokerParamList += "_" + paramNameAsClojure(p.Name)

			if jokerParamListDoc != "" {
				jokerParamListDoc += ", "
			}
			if clTypeDoc != "" {
				jokerParamListDoc += "^" + clTypeDoc + " "
			}
			jokerParamListDoc += paramNameAsClojure(p.Name)

			if jokerGoParams != "" {
				jokerGoParams += ", "
			}
			jokerGoParams += jok2golParam

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
	jokerGoParams = "(" + jokerGoParams + ")"
	jokerParamListDoc = "[" + jokerParamListDoc + "]"
	if strings.Contains(goParamListDoc, " ") || strings.Contains(goParamListDoc, ",") {
		goParamListDoc = "(" + goParamListDoc + ")"
	}
	return
}
