package main

import (
	"fmt"
	. "github.com/candid82/joker/tools/gostd/gowalk"
	. "github.com/candid82/joker/tools/gostd/utils"
	. "go/ast"
	"strings"
)

func genGoPostSelected(fn *FuncInfo, indent, captureName, fullTypeName, onlyIf string) (cl, clDoc, gol, goc, out string) {
	clDoc = FullTypeNameAsClojure(fn.SourceFile.NsRoot, fullTypeName)
	if _, ok := GoTypes[fullTypeName]; ok {
		gol = fullTypeName
		out = "MakeGoObject(" + captureName + ")"
	} else {
		clDoc = fmt.Sprintf("ABEND042(post.go: cannot find typename %s)", fullTypeName)
		gol = "..."
		out = captureName
	}
	return
}

func genGoPostNamed(fn *FuncInfo, indent, captureName, typeName, onlyIf string) (cl, clDoc, gol, goc, out string) {
	return genGoPostSelected(fn, indent, captureName, fn.SourceFile.PkgDirUnix+"."+typeName, onlyIf)
}

func genGoPostSelector(fn *FuncInfo, indent, captureName string, e *SelectorExpr, onlyIf string) (cl, clDoc, gol, goc, out string) {
	pkgName := e.X.(*Ident).Name
	fullPathUnix := Unix(FileAt(e.Pos()))
	referringFile := strings.TrimPrefix(fullPathUnix, fn.SourceFile.RootUnix+"/")
	rf, ok := GoFiles[referringFile]
	if !ok {
		panic(fmt.Sprintf("genGoPostSelector: could not find referring file %s for file %s at %s",
			referringFile, fullPathUnix, WhereAt(e.Pos())))
	}
	if fullPkgName, found := (*rf.Spaces)[pkgName]; found {
		return genGoPostSelected(fn, indent, captureName, fullPkgName+"."+e.Sel.Name, onlyIf)
	}
	panic(fmt.Sprintf("processing %s for %s: could not find %s in %s",
		WhereAt(e.Pos()), WhereAt(fn.Fd.Pos()), pkgName, fn.SourceFile.Name))
}

// func tryThis(s string) struct { a int; b string } {
//	return struct { a int; b string }{ 5, "hey" }
// }

// Joker: { :a ^Int, :b ^String }
// Go: struct { a int; b string }
func genGoPostStruct(fn *FuncInfo, indent, captureName string, fl *FieldList, onlyIf string) (cl, clDoc, gol, goc, out string) {
	tmpmap := "_map" + genSym("")
	useful := false
	fields := FlattenFieldList(fl)
	for _, field := range fields {
		p := field.Name
		if !IsExported(p.Name) {
			continue // Skipping non-exported fields
		}
		clType, clTypeDoc, golType, more_goc, outNew :=
			genGoPostExpr(fn, indent, captureName+"."+p.Name, field.Field.Type, "")
		out = outNew
		if useful || exprIsUseful(out) {
			useful = true
		}
		goc += more_goc
		goc += indent + tmpmap +
			".Add(MakeKeyword(\"" + p.Name + "\"), " + out + ")\n"
		if cl != "" {
			cl += ", "
		}
		if gol != "" {
			gol += "; "
		}
		if p == nil {
			cl += "_ "
		} else {
			cl += ":" + p.Name + " "
			gol += p.Name + " "
		}
		if clType != "" {
			cl += "^" + clType
		}
		if clTypeDoc != "" {
			clDoc += "^" + clTypeDoc
		}
		if golType != "" {
			gol += golType
		}
	}
	if cl != "" {
		cl = "{" + cl + "}"
	}
	clDoc = "{" + clDoc + "}"
	gol = "struct {" + gol + "}"
	if useful {
		goc = wrapStmtOnlyIfs(indent, tmpmap, "ArrayMap", "EmptyArrayMap()", onlyIf, goc, &out)
	} else {
		goc = ""
		out = "NIL"
	}
	return
}

func genGoPostArray(fn *FuncInfo, indent, captureName string, el Expr, onlyIf string) (cl, clDoc, gol, goc, out string) {
	tmp := genSym("")
	tmpvec := "_vec" + tmp
	tmpelem := "_elem" + tmp

	var goc_pre string
	cl, clDoc, gol, goc_pre, out = genGoPostExpr(fn, indent+"\t", tmpelem, el, "")
	useful := exprIsUseful(out)
	if cl != "" {
		cl = "(vector-of " + cl + ")"
	}
	clDoc = "(vector-of " + clDoc + ")"
	gol = "[]" + gol

	if useful {
		goc = indent + "for _, " + tmpelem + " := range " + captureName + " {\n"
		goc += goc_pre
		goc += indent + "\t" + tmpvec + " = " + tmpvec + ".Conjoin(" + out + ")\n"
		goc += indent + "}\n"
		goc = wrapStmtOnlyIfs(indent, tmpvec, "Vector", "EmptyVector()", onlyIf, goc, &out)
	} else {
		goc = ""
	}
	return
}

func genGoPostStar(fn *FuncInfo, indent, captureName string, e Expr, onlyIf string) (cl, clDoc, gol, goc, out string) {
	cl, clDoc, gol, goc, out = genGoPostExpr(fn, indent, fmt.Sprintf("ABEND333(post.go: should not show up: %s)", captureName), e, onlyIf)
	out = "MakeGoObject(" + captureName + ")"
	if cl != "" {
		cl = "(atom-of " + cl + ")"
	}
	clDoc = "(atom-of " + clDoc + ")"
	gol = "*" + gol
	return
}

func maybeNil(expr, captureName string) string {
	return "func () Object { if (" + expr + ") == nil { return NIL } else { return " + captureName + " } }()"
}

func genGoPostExpr(fn *FuncInfo, indent, captureName string, e Expr, onlyIf string) (cl, clDoc, gol, goc, out string) {
	switch v := e.(type) {
	case *Ident:
		gol = v.Name
		ti := toGoExprInfo(fn.SourceFile, &e)
		cl = ti.ArgExtractFunc
		if ti.ConvertToClojure == "" {
			out = fmt.Sprintf("ABEND043(post.go: unsupported built-in type %s)", v.Name)
		} else {
			out = "Make" + fmt.Sprintf(ti.ConvertToClojure, captureName, "")
		}
		if ti.Nullable {
			out = maybeNil(captureName, out)
		}
		clDoc = ti.ArgClojureArgType
	case *ArrayType:
		cl, clDoc, gol, goc, out = genGoPostArray(fn, indent, captureName, v.Elt, onlyIf)
	case *StarExpr:
		cl, clDoc, gol, goc, out = genGoPostStar(fn, indent, captureName, v.X, onlyIf)
	case *SelectorExpr:
		cl, clDoc, gol, goc, out = genGoPostSelector(fn, indent, captureName, v, onlyIf)
	case *StructType:
		cl, clDoc, gol, goc, out = genGoPostStruct(fn, indent, captureName, v.Fields, onlyIf)
	default:
		cl = fmt.Sprintf("ABEND883(post.go: unrecognized Expr type %T at: %s)", e, Unix(WhereAt(e.Pos())))
		gol = "..."
		out = captureName
	}
	if clDoc == "" {
		clDoc = cl
	}
	return
}

const resultName = "_res"

func genGoPostItem(fn *FuncInfo, indent, captureName string, f *Field, onlyIf string) (captureVar, cl, clDoc, gol, goc, out string, useful bool) {
	captureVar = captureName
	if captureName == "" {
		captureVar = genSym(resultName)
	}
	cl, clDoc, gol, goc, out = genGoPostExpr(fn, indent, captureVar, f.Type, onlyIf)
	if captureName != "" && captureName != resultName {
		gol = paramNameAsGo(captureName) + " " + gol
	}
	useful = exprIsUseful(out)
	if !useful {
		captureVar = "_"
	}
	return
}

// Caller generates "outGOCALL;goc" while saving cl and gol for type info (they go into .joke as metadata and docstrings)
func genGoPostList(fn *FuncInfo, indent string, fl *FieldList) (cl, clDoc, gol, goc, out string) {
	useful := false
	captureVars := []string{}
	clType := []string{}
	clTypeDoc := []string{}
	golType := []string{}
	goCode := []string{}

	result := resultName
	fields := FlattenFieldList(fl)
	multipleCaptures := len(fields) > 1
	for _, field := range fields {
		n := ""
		if field.Name != nil {
			n = field.Name.Name
		}
		captureName := result
		if multipleCaptures {
			captureName = n
		}
		captureVar, clNew, clDocNew, golNew, gocNew, outNew, usefulItem := genGoPostItem(fn, indent, captureName, field.Field, "")
		useful = useful || usefulItem
		if multipleCaptures {
			gocNew += indent + result + " = " + result + ".Conjoin(" + outNew + ")\n"
		} else {
			result = outNew
		}
		captureVars = append(captureVars, captureVar)
		clType = append(clType, clNew)
		clTypeDoc = append(clTypeDoc, clDocNew)
		golType = append(golType, golNew)
		goCode = append(goCode, gocNew)
	}

	out = strings.Join(captureVars, ", ")
	if out != "" {
		out += " := "
	}

	cl = strings.Join(clType, " ")
	if len(clType) > 1 && cl != "" {
		cl = "[" + cl + "]"
	}

	clDoc = strings.Join(clTypeDoc, " ")
	if len(clTypeDoc) > 1 && clDoc != "" {
		clDoc = "[" + clDoc + "]"
	}

	gol = strings.Join(golType, ", ")
	if len(golType) > 1 && gol != "" {
		gol = "(" + gol + ")"
	}

	goc = strings.Join(goCode, "")

	if multipleCaptures {
		if useful {
			goc = indent + result + " := EmptyVector()\n" + goc + indent + "return " + result + "\n"
		} else {
			goc = indent + "ABEND123(post.go: no public information returned)\n"
		}
	} else {
		if goc == "" && result == resultName {
			out = "return " // No code generated, so no need to use intermediary
		} else {
			goc += indent + "return " + result + "\n"
		}
		if !useful {
			goc += indent + "ABEND124(post.go: no public information returned)\n"
		}
	}

	return
}

func genGoPost(fn *FuncInfo, indent string, d *FuncDecl) (goResultAssign, clojureReturnType, clojureReturnTypeForDoc, goReturnTypeForDoc, goReturnCode string) {
	fl := d.Type.Results
	if fl == nil || fl.List == nil {
		return
	}
	clojureReturnType, clojureReturnTypeForDoc, goReturnTypeForDoc, goReturnCode, goResultAssign = genGoPostList(fn, indent, fl)
	return
}
