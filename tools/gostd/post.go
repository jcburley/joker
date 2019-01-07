package main

import (
	"fmt"
	. "go/ast"
	"strings"
)

func genGoPostSelected(fn *funcInfo, indent, captureName, fullTypeName, onlyIf string) (jok, gol, goc, out string) {
	if _, ok := types[fullTypeName]; ok {
		jok = fullTypeName
		gol = fullTypeName
		runtime := "ConvertFrom" + fullTypeName
		out = runtime + "(" + captureName + ")"
		if _, ok := customRuntimeImplemented[runtime]; !ok {
			if !strings.Contains(out, "ABEND") {
				out = "ABEND911(custom-runtime routine not implemented: " + out + ")"
			}
		}
	} else {
		jok = fmt.Sprintf("ABEND042(cannot find typename %s)", fullTypeName)
		gol = "..."
		out = captureName
	}
	return
}

func genGoPostNamed(fn *funcInfo, indent, captureName, typeName, onlyIf string) (jok, gol, goc, out string) {
	return genGoPostSelected(fn, indent, captureName, fn.sourceFile.pkgDirUnix+"."+typeName, onlyIf)
}

func genGoPostSelector(fn *funcInfo, indent, captureName string, e *SelectorExpr, onlyIf string) (jok, gol, goc, out string) {
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
func genGoPostStruct(fn *funcInfo, indent, captureName string, fl *FieldList, onlyIf string) (jok, gol, goc, out string) {
	tmpmap := "_map" + genSym("")
	useful := false
	for _, f := range fl.List {
		for _, p := range f.Names {
			if isPrivate(p.Name) {
				continue // Skipping non-exported fields
			}
			var joktype, goltype, more_goc string
			joktype, goltype, more_goc, out =
				genGoPostExpr(fn, indent, captureName+"."+p.Name, f.Type, "")
			if useful || exprIsUseful(out) {
				useful = true
			}
			goc += more_goc
			goc += indent + tmpmap +
				".Add(MakeKeyword(\"" + p.Name + "\"), " + out + ")\n"
			if jok != "" {
				jok += ", "
			}
			if gol != "" {
				gol += "; "
			}
			if p == nil {
				jok += "_ "
			} else {
				jok += ":" + p.Name + " "
				gol += p.Name + " "
			}
			if joktype != "" {
				jok += "^" + joktype
			}
			if goltype != "" {
				gol += goltype
			}
		}
	}
	jok = "{" + jok + "}"
	gol = "struct {" + gol + "}"
	if useful {
		goc = wrapStmtOnlyIfs(indent, tmpmap, "ArrayMap", "EmptyArrayMap()", onlyIf, goc, &out)
	} else {
		goc = ""
		out = "NIL"
	}
	return
}

func genGoPostArray(fn *funcInfo, indent, captureName string, el Expr, onlyIf string) (jok, gol, goc, out string) {
	tmp := genSym("")
	tmpvec := "_vec" + tmp
	tmpelem := "_elem" + tmp

	var goc_pre string
	jok, gol, goc_pre, out = genGoPostExpr(fn, indent+"\t", tmpelem, el, "")
	useful := exprIsUseful(out)
	jok = "(vector-of " + jok + ")"
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

// TODO: Maybe return a ref or something Joker (someday) supports? flag.String() is useful only as it returns a ref;
// whereas net.LookupMX() returns []*MX, and these are not only populated, it's unclear there's any utility in
// modifying them (it could just as well return []MX AFAICT).
func genGoPostStar(fn *funcInfo, indent, captureName string, e Expr, onlyIf string) (jok, gol, goc, out string) {
	if onlyIf == "" {
		onlyIf = captureName + " != nil"
	} else {
		onlyIf = captureName + " != nil && " + onlyIf
	}
	jok, gol, goc, out = genGoPostExpr(fn, indent, "(*"+captureName+")", e, onlyIf)
	gol = "*" + gol
	return
}

func maybeNil(expr, captureName string) string {
	return "func () Object { if (" + expr + ") == nil { return NIL } else { return " + captureName + " } }()"
}

func genGoPostExpr(fn *funcInfo, indent, captureName string, e Expr, onlyIf string) (jok, gol, goc, out string) {
	switch v := e.(type) {
	case *Ident:
		switch v.Name {
		case "string":
			jok = "String"
			gol = "string"
			out = "MakeString(" + captureName + ")"
		case "int":
			jok = "Int"
			gol = "int"
			out = "MakeInt(" + captureName + ")"
		case "int16", "uint", "uint16", "int32", "uint32", "int64", "byte": // TODO: Does Joker always have 64-bit signed ints?
			jok = "Int"
			gol = "int"
			out = "MakeInt(int(" + captureName + "))"
		case "bool":
			jok = "Bool"
			gol = "bool"
			out = "MakeBool(" + captureName + ")"
		case "error":
			jok = "Error"
			gol = "error"
			out = maybeNil(captureName, "MakeError("+captureName+")") // TODO: Test this against the MakeError() added to joker/core/object.go
		default:
			if isPrivate(v.Name) {
				jok = fmt.Sprintf("ABEND043(unsupported built-in type %s)", v.Name)
				gol = "..."
				out = captureName
			} else {
				jok, _, goc, out = genGoPostNamed(fn, indent, captureName, v.Name, onlyIf)
				gol = v.Name // This is as far as Go needs to go for a type signature
			}
		}
	case *ArrayType:
		jok, gol, goc, out = genGoPostArray(fn, indent, captureName, v.Elt, onlyIf)
	case *StarExpr:
		jok, gol, goc, out = genGoPostStar(fn, indent, captureName, v.X, onlyIf)
	case *SelectorExpr:
		jok, gol, goc, out = genGoPostSelector(fn, indent, captureName, v, onlyIf)
	case *StructType:
		jok, gol, goc, out = genGoPostStruct(fn, indent, captureName, v.Fields, onlyIf)
	default:
		jok = fmt.Sprintf("ABEND883(unrecognized Expr type %T at: %s)", e, unix(whereAt(e.Pos())))
		gol = "..."
		out = captureName
	}
	return
}

const resultName = "_res"

func genGoPostItem(fn *funcInfo, indent, captureName string, f *Field, onlyIf string) (captureVar, jok, gol, goc, out string, useful bool) {
	captureVar = captureName
	if captureName == "" {
		captureVar = genSym(resultName)
	}
	jok, gol, goc, out = genGoPostExpr(fn, indent, captureVar, f.Type, onlyIf)
	if captureName != "" && captureName != resultName {
		gol = paramNameAsGo(captureName) + " " + gol
	}
	useful = exprIsUseful(out)
	if !useful {
		captureVar = "_"
	}
	return
}

// Caller generates "outGOCALL;goc" while saving jok and gol for type info (they go into .joke as metadata and docstrings)
func genGoPostList(fn *funcInfo, indent string, fl FieldList) (jok, gol, goc, out string) {
	useful := false
	captureVars := []string{}
	jokType := []string{}
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
			captureVar, jokNew, golNew, gocNew, outNew, usefulItem := genGoPostItem(fn, indent, captureName, f, "")
			useful = useful || usefulItem
			if multipleCaptures {
				gocNew += indent + result + " = " + result + ".Conjoin(" + outNew + ")\n"
			} else {
				result = outNew
			}
			captureVars = append(captureVars, captureVar)
			jokType = append(jokType, jokNew)
			golType = append(golType, golNew)
			goCode = append(goCode, gocNew)
		}
	}

	out = strings.Join(captureVars, ", ")
	if out != "" {
		out += " := "
	}

	jok = strings.Join(jokType, " ")
	if len(jokType) > 1 && jok != "" {
		jok = "[" + jok + "]"
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

func genGoPost(fn *funcInfo, indent string, d *FuncDecl) (goResultAssign, jokerReturnTypeForDoc, goReturnTypeForDoc string, goReturnCode string) {
	fl := d.Type.Results
	if fl == nil || fl.List == nil {
		return
	}
	jokerReturnTypeForDoc, goReturnTypeForDoc, goReturnCode, goResultAssign = genGoPostList(fn, indent, *fl)
	return
}

// Return a form of the return type as supported by generate-std.joke,
// or empty string if not supported (which will trigger attempting to
// generate appropriate code for *_native.go). gol either passes
// through or "Object" is returned for it if jok is returned as empty.
func jokerReturnTypeForGenerateCustom(in_jok, in_gol string) (jok, gol string) {
	switch in_jok {
	case "String", "Int", "Byte", "Double", "Bool", "Time", "Error": // TODO: Have tested only String so far
		jok = `^"` + in_jok + `"`
	default:
		jok = ""
		gol = "Object"
	}
	return
}
