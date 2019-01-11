package main

import (
	"fmt"
	. "go/ast"
	"strings"
)

func genGoPostSelected(fn *funcInfo, indent, captureName, fullTypeName, onlyIf string) (cl, gol, goc, out string) {
	if _, ok := types[fullTypeName]; ok {
		cl = fullTypeNameAsClojure(fullTypeName)
		gol = fullTypeName
		out = "MakeGoObject(" + captureName + ")"
	} else {
		cl = fmt.Sprintf("ABEND042(cannot find typename %s)", fullTypeName)
		gol = "..."
		out = captureName
	}
	return
}

func genGoPostNamed(fn *funcInfo, indent, captureName, typeName, onlyIf string) (cl, gol, goc, out string) {
	return genGoPostSelected(fn, indent, captureName, fn.sourceFile.pkgDirUnix+"."+typeName, onlyIf)
}

func genGoPostSelector(fn *funcInfo, indent, captureName string, e *SelectorExpr, onlyIf string) (cl, gol, goc, out string) {
	pkgName := e.X.(*Ident).Name
	referringFile := strings.TrimPrefix(fileAt(e.Pos()), fn.sourceFile.rootUnix+"/")
	rf, ok := goFiles[referringFile]
	if !ok {
		panic(fmt.Sprintf("genGoPostSelector: could not find referring file %s for expression at %s",
			referringFile, whereAt(e.Pos())))
	}
	if fullPkgName, found := (*rf.spaces)[pkgName]; found {
		return genGoPostSelected(fn, indent, captureName, fullPkgName+"."+e.Sel.Name, onlyIf)
	}
	panic(fmt.Sprintf("processing %s for %s: could not find %s in %s",
		whereAt(e.Pos()), whereAt(fn.fd.Pos()), pkgName, fn.sourceFile.name))
}

// func tryThis(s string) struct { a int; b string } {
//	return struct { a int; b string }{ 5, "hey" }
// }

// Joker: { :a ^Int, :b ^String }
// Go: struct { a int; b string }
func genGoPostStruct(fn *funcInfo, indent, captureName string, fl *FieldList, onlyIf string) (cl, gol, goc, out string) {
	tmpmap := "_map" + genSym("")
	useful := false
	for _, f := range fl.List {
		for _, p := range f.Names {
			if isPrivate(p.Name) {
				continue // Skipping non-exported fields
			}
			var clType, golType, more_goc string
			clType, golType, more_goc, out =
				genGoPostExpr(fn, indent, captureName+"."+p.Name, f.Type, "")
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
			if golType != "" {
				gol += golType
			}
		}
	}
	cl = "{" + cl + "}"
	gol = "struct {" + gol + "}"
	if useful {
		goc = wrapStmtOnlyIfs(indent, tmpmap, "ArrayMap", "EmptyArrayMap()", onlyIf, goc, &out)
	} else {
		goc = ""
		out = "NIL"
	}
	return
}

func genGoPostArray(fn *funcInfo, indent, captureName string, el Expr, onlyIf string) (cl, gol, goc, out string) {
	tmp := genSym("")
	tmpvec := "_vec" + tmp
	tmpelem := "_elem" + tmp

	var goc_pre string
	cl, gol, goc_pre, out = genGoPostExpr(fn, indent+"\t", tmpelem, el, "")
	useful := exprIsUseful(out)
	cl = "(vector-of " + cl + ")"
	gol = "[]" + gol

	if useful {
		goc = indent + "for _, " + tmpelem + " := range " + captureName + " {\n"
		goc += goc_pre
		goc += indent + "\t" + tmpvec + " = " + tmpvec + ".Conjoin(" + out + ")\n"
		goc += indent + "}\n"
		goc = wrapStmtOnlyIfs(indent, tmpvec, "Vector", "EmptyVector", onlyIf, goc, &out)
	} else {
		goc = ""
	}
	return
}

func genGoPostStar(fn *funcInfo, indent, captureName string, e Expr, onlyIf string) (cl, gol, goc, out string) {
	cl, gol, goc, out = genGoPostExpr(fn, indent, fmt.Sprintf("ABEND333(should not show up: %s)", captureName), e, onlyIf)
	out = "MakeGoObject(" + captureName + ")"
	cl = "(atom-of " + cl + ")"
	gol = "*" + gol
	return
}

func maybeNil(expr, captureName string) string {
	return "func () Object { if (" + expr + ") == nil { return NIL } else { return " + captureName + " } }()"
}

func genGoPostExpr(fn *funcInfo, indent, captureName string, e Expr, onlyIf string) (cl, gol, goc, out string) {
	switch v := e.(type) {
	case *Ident:
		gol = v.Name
		switch v.Name {
		case "string":
			cl = "String"
			out = "MakeString(" + captureName + ")"
		case "int":
			cl = "Int"
			out = "MakeInt(" + captureName + ")"
		case "int16", "uint", "uint16", "int32", "uint32", "int64", "byte": // TODO: Does Joker always have 64-bit signed ints?
			cl = ""
			out = "MakeInt(int(" + captureName + "))"
		case "bool":
			cl = "Bool"
			out = "MakeBool(" + captureName + ")"
		case "error":
			cl = "Error"
			out = maybeNil(captureName, "MakeError("+captureName+")") // TODO: Test this against the MakeError() added to joker/core/object.go
		default:
			if isPrivate(v.Name) {
				cl = fmt.Sprintf("ABEND043(unsupported built-in type %s)", v.Name)
				gol = "..."
				out = captureName
			} else {
				cl, _, goc, out = genGoPostNamed(fn, indent, captureName, v.Name, onlyIf)
			}
		}
	case *ArrayType:
		cl, gol, goc, out = genGoPostArray(fn, indent, captureName, v.Elt, onlyIf)
	case *StarExpr:
		cl, gol, goc, out = genGoPostStar(fn, indent, captureName, v.X, onlyIf)
	case *SelectorExpr:
		cl, gol, goc, out = genGoPostSelector(fn, indent, captureName, v, onlyIf)
	case *StructType:
		cl, gol, goc, out = genGoPostStruct(fn, indent, captureName, v.Fields, onlyIf)
	default:
		cl = fmt.Sprintf("ABEND883(unrecognized Expr type %T at: %s)", e, unix(whereAt(e.Pos())))
		gol = "..."
		out = captureName
	}
	return
}

const resultName = "_res"

func genGoPostItem(fn *funcInfo, indent, captureName string, f *Field, onlyIf string) (captureVar, cl, gol, goc, out string, useful bool) {
	captureVar = captureName
	if captureName == "" {
		captureVar = genSym(resultName)
	}
	cl, gol, goc, out = genGoPostExpr(fn, indent, captureVar, f.Type, onlyIf)
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
func genGoPostList(fn *funcInfo, indent string, fl FieldList) (cl, gol, goc, out string) {
	useful := false
	captureVars := []string{}
	clType := []string{}
	golType := []string{}
	goCode := []string{}

	result := resultName
	multipleCaptures := len(fl.List) > 1 || (fl.List[0].Names != nil && len(fl.List[0].Names) > 1)
	for _, f := range fl.List {
		names := []string{}
		if f.Names == nil {
			names = append(names, "")
		} else {
			for _, n := range f.Names {
				names = append(names, n.Name)
			}
		}
		for _, n := range names {
			captureName := result
			if multipleCaptures {
				captureName = n
			}
			captureVar, clNew, golNew, gocNew, outNew, usefulItem := genGoPostItem(fn, indent, captureName, f, "")
			useful = useful || usefulItem
			if multipleCaptures {
				gocNew += indent + result + " = " + result + ".Conjoin(" + outNew + ")\n"
			} else {
				result = outNew
			}
			captureVars = append(captureVars, captureVar)
			clType = append(clType, clNew)
			golType = append(golType, golNew)
			goCode = append(goCode, gocNew)
		}
	}

	out = strings.Join(captureVars, ", ")
	if out != "" {
		out += " := "
	}

	cl = strings.Join(clType, " ")
	if len(clType) > 1 && cl != "" {
		cl = "[" + cl + "]"
	}

	gol = strings.Join(golType, ", ")
	if len(golType) > 1 && gol != "" {
		gol = "(" + gol + ")"
	}

	goc = strings.Join(goCode, "")

	if multipleCaptures {
		if useful {
			goc = indent + result + " := EmptyVector\n" + goc + indent + "return " + result + "\n"
		} else {
			goc = indent + "ABEND123(no public information returned)\n"
		}
	} else {
		if goc == "" && result == resultName {
			out = "return " // No code generated, so no need to use intermediary
		} else {
			goc += indent + "return " + result + "\n"
		}
		if !useful {
			goc += indent + "ABEND124(no public information returned)\n"
		}
	}

	return
}

func genGoPost(fn *funcInfo, indent string, d *FuncDecl) (goResultAssign, clojureReturnTypeForDoc, goReturnTypeForDoc string, goReturnCode string) {
	fl := d.Type.Results
	if fl == nil || fl.List == nil {
		return
	}
	clojureReturnTypeForDoc, goReturnTypeForDoc, goReturnCode, goResultAssign = genGoPostList(fn, indent, *fl)
	return
}
