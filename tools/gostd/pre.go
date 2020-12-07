package main

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/astutils"
	"github.com/candid82/joker/tools/gostd/genutils"
	. "github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/paths"
	. "go/ast"
	"strconv"
	"strings"
)

func genGoPreArray(fn *FuncInfo, indent string, e *ArrayType, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	el := e.Elt
	len := e.Len
	_, _, goType, goTypeDoc, _, cl2golParam = genTypePre(fn, indent, el, paramName, argNum)
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
	goType = "[]" + goType
	goTypeDoc = "[]" + goTypeDoc
	return
}

func genGoPreStar(fn *FuncInfo, indent string, e *StarExpr, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	el := e.X
	clType, _, goType, goTypeDoc, _, cl2golParam = genTypePre(fn, indent, el, paramName, argNum)
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
	//	clTypeDoc = "(ref-to " + clTypeDoc + ")"
	goType = "*" + goType
	goTypeDoc = "*" + goTypeDoc
	return
}

func genGoPreSelected(fn *FuncInfo, indent, fullPkgName, baseTypeName, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	clType, clTypeDoc, goType, goTypeDoc = FullPkgNameAsGoType(fn, fullPkgName, baseTypeName)
	if goType == "unsafe.ArbitraryType" {
		cl2golParam = paramName // genType generates functions that return pointers to objects, to avoid copying-sync.Mutex issues
	} else {
		cl2golParam = "*" + paramName // genType generates functions that return pointers to objects, to avoid copying-sync.Mutex issues
	}
	return
}

func genGoPreNamed(fn *FuncInfo, indent, typeName, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	return genGoPreSelected(fn, indent, fn.SourceFile.Package.Dir.String(), typeName, paramName, argNum)
}

func genGoPreSelector(fn *FuncInfo, indent string, e *SelectorExpr, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
	pkgName := e.X.(*Ident).Name
	fullPathUnix := paths.Unix(FileAt(e.Pos()))
	referringFile := strings.TrimPrefix(fullPathUnix, fn.SourceFile.Package.Root.String()+"/")
	rf, ok := GoFilesRelative[referringFile]
	if !ok {
		panic(fmt.Sprintf("genGoPreSelector: could not find referring file %s for file %s at %s",
			referringFile, fullPathUnix, WhereAt(e.Pos())))
	}
	if fullPkgName, found := (*rf.Spaces)[pkgName]; found {
		return genGoPreSelected(fn, indent, fullPkgName.String(), e.Sel.Name, paramName, argNum)
	}
	panic(fmt.Sprintf("processing %s for %s: could not find %s in %s",
		WhereAt(e.Pos()), WhereAt(fn.Fd.Pos()), pkgName, fn.SourceFile.Name))
}

func genGoPreEllipsis(fn *FuncInfo, indent string, e *Ellipsis, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
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

func genGoPreFunc(fn *FuncInfo, indent string, e *FuncType, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
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

func genGoPreInterface(fn *FuncInfo, indent string, e *InterfaceType, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
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

func genGoPreMap(fn *FuncInfo, indent string, e *MapType, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
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

func genGoPreChan(fn *FuncInfo, indent string, e *ChanType, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, cl2golParam string) {
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

func genTypePre(fn *FuncInfo, indent string, e Expr, paramName string, argNum int) (clType, clTypeDoc, goType, goTypeDoc, goPreCode, cl2golParam string) {
	cl2golParam = paramName
	switch v := e.(type) {
	case *Ident:
		goType = v.Name
		extractParam := ""
		ti := TypeInfoForExpr(e)
		clType = ti.ArgExtractFunc()
		clTypeDoc = ti.JokerNameDoc(e)
		if clTypeDoc == "" {
			clTypeDoc = clType
		}
		if !ti.Custom() { // a builtin
			if clType != "" {
				extractParam = fmt.Sprintf("ExtractGo%s(\"%s\", \"%s\", _argList, %d)", clType, fn.DocName, paramName, argNum)
			}
		} else {
			if !IsExported(v.Name) {
				clType = fmt.Sprintf("ABEND044(pre.go: unsupported built-in type %s)", v.Name)
				clTypeDoc = v.Name
			} else {
				clType, _, goType, goTypeDoc, cl2golParam = genGoPreNamed(fn, indent, v.Name, paramName, argNum)
			}
			if ti.JokerName() != "" {
				extractParam = fmt.Sprintf("ExtractGo_%s(\"%s\", \"%s\", _argList, %d)", genutils.TypeToGoExtractFuncName(ti.JokerName()), fn.DocName, paramName, argNum)
			}
		}
		if clTypeDoc == "" {
			clTypeDoc = clType
		}
		if goTypeDoc == "" {
			goTypeDoc = goType
		}
		if fn.Fd == nil || fn.Fd.Recv != nil {
			if extractParam == "" {
				panic(fmt.Sprintf("no arg-extraction code for %+v type (%s, %s, %s, %s) at %s @%p %+v", v, goType, goTypeDoc, clType, clTypeDoc, WhereAt(v.Pos()), ti, *ti.JokerTypeInfo()))
			}
			goPreCode = paramName + " := " + extractParam
		}
	case *ArrayType:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreArray(fn, indent, v, paramName, argNum)
	case *StarExpr:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreStar(fn, indent, v, paramName, argNum)
	case *SelectorExpr:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreSelector(fn, indent, v, paramName, argNum)
	case *Ellipsis:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreEllipsis(fn, indent, v, paramName, argNum)
	case *FuncType:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreFunc(fn, indent, v, paramName, argNum)
	case *InterfaceType:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreInterface(fn, indent, v, paramName, argNum)
	case *MapType:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreMap(fn, indent, v, paramName, argNum)
	case *ChanType:
		clType, clTypeDoc, goType, goTypeDoc, cl2golParam = genGoPreChan(fn, indent, v, paramName, argNum)
	}
	if (fn.Fd == nil || fn.Fd.Recv != nil) && goPreCode == "" {
		goPreCode = fmt.Sprintf("ABEND644(pre.go: unsupported built-in type %T for %s at: %s)", e, paramName, paths.Unix(WhereAt(e.Pos())))
	}

	if clType == "" || clTypeDoc == "" {
		ti := TypeInfoForExpr(e)
		if clType == "" {
			clType = ti.ArgExtractFunc()
		}
		if clType == "" {
			clType = "Object"
		}
		if clTypeDoc == "" {
			clTypeDoc = ti.JokerNameDoc(e)
		}
		if clTypeDoc == "" {
			clTypeDoc = clType
		}
		//		fmt.Printf("pre.go/genTypePre: e @%p == %+v ti@%p == %+v clType=%s clTypeDoc=%s\n", e, e, ti, ti, clType, clTypeDoc)
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
