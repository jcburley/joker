package main

import (
	"fmt"
	. "github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/gowalk"
	. "github.com/candid82/joker/tools/gostd/jtypes"
	"github.com/candid82/joker/tools/gostd/types"
	. "github.com/candid82/joker/tools/gostd/utils"
	. "go/ast"
	"strings"
)

func genGoPostSelected(fn *gowalk.FuncInfo, indent, captureName, fullTypeName, onlyIf string) (cl, clDoc, gol, goc, out string) {
	clDoc = gowalk.FullTypeNameAsClojure(fn.SourceFile.Package.NsRoot, fullTypeName)
	if _, ok := gowalk.GoTypes[fullTypeName]; ok {
		gol = fullTypeName
		out = "MakeGoObject(" + captureName + ")"
	} else {
		clDoc = fmt.Sprintf("ABEND042(post.go: cannot find typename %s)", fullTypeName)
		gol = "..."
		out = captureName
	}
	return
}

func genGoPostNamed(fn *gowalk.FuncInfo, indent, captureName, typeName, onlyIf string) (cl, clDoc, gol, goc, out string) {
	return genGoPostSelected(fn, indent, captureName, fn.SourceFile.Package.Dir.String()+"."+typeName, onlyIf)
}

func genGoPostSelector(fn *gowalk.FuncInfo, indent, captureName string, e *SelectorExpr, onlyIf string) (cl, clDoc, gol, goc, out string) {
	pkgName := e.X.(*Ident).Name
	fullPathUnix := Unix(FileAt(e.Pos()))
	referringFile := strings.TrimPrefix(fullPathUnix, fn.SourceFile.Package.Root.String()+"/")
	rf, ok := GoFiles[referringFile]
	if !ok {
		panic(fmt.Sprintf("genGoPostSelector: could not find referring file %s for file %s at %s",
			referringFile, fullPathUnix, WhereAt(e.Pos())))
	}
	if fullPkgName, found := (*rf.Spaces)[pkgName]; found {
		return genGoPostSelected(fn, indent, captureName, fullPkgName.String()+"."+e.Sel.Name, onlyIf)
	}
	panic(fmt.Sprintf("processing %s for %s: could not find %s in %s",
		WhereAt(e.Pos()), WhereAt(fn.Fd.Pos()), pkgName, fn.SourceFile.Name))
}

// func tryThis(s string) struct { a int; b string } {
//	return struct { a int; b string }{ 5, "hey" }
// }

// Joker: { :a ^Int, :b ^String }
// Go: struct { a int; b string }
func genGoPostArray(fn *gowalk.FuncInfo, indent, captureName string, e Expr, onlyIf string) (cl, clDoc, gol, goc, out string) {
	cl, clDoc, gol, goc, out = genGoPostExpr(fn, indent, fmt.Sprintf("ABEND333(post.go: should not show up: %s)", captureName), e, onlyIf)
	out = "MakeGoObject(" + captureName + ")"
	if cl != "" {
		s := strings.Split(cl, "/")
		s[len(s)-1] = "arrayOf" + s[len(s)-1]
		cl = strings.Join(s, "/")
	}
	if clDoc != "" {
		s := strings.Split(clDoc, "/")
		s[len(s)-1] = "arrayOf" + s[len(s)-1]
		clDoc = strings.Join(s, "/")
	}
	gol = "[]" + gol
	return
}

func genGoPostStar(fn *gowalk.FuncInfo, indent, captureName string, e Expr, onlyIf string) (cl, clDoc, gol, goc, out string) {
	cl, clDoc, gol, goc, out = genGoPostExpr(fn, indent, fmt.Sprintf("ABEND333(post.go: should not show up: %s)", captureName), e, onlyIf)
	out = "MakeGoObject(" + captureName + ")"
	if cl != "" {
		cl = "(ref-to " + cl + ")"
	}
	clDoc = "(ref-to " + clDoc + ")"
	gol = "*" + gol
	return
}

func maybeNil(expr, captureName string) string {
	return "func () Object { if (" + expr + ") == nil { return NIL } else { return " + captureName + " } }()"
}

func genGoPostExpr(fn *gowalk.FuncInfo, indent, captureName string, e Expr, onlyIf string) (cl, clDoc, gol, goc, out string) {
	switch v := e.(type) {
	case *Ident:
		gol = v.Name
		jti := JokerTypeInfo(e)
		if jti == nil || jti.ConvertToClojure == "" {
			out = fmt.Sprintf("ABEND043(post.go: unsupported built-in type %s)", v.Name)
		} else {
			out = "Make" + fmt.Sprintf(jti.ConvertToClojure, captureName, "")
		}
		if jti.Nullable {
			out = maybeNil(captureName, out)
		}
		cl = jti.ArgExtractFunc
		clDoc = jti.ArgClojureArgType
	case *ArrayType:
		cl, clDoc, gol, goc, out = genGoPostArray(fn, indent, captureName, v.Elt, onlyIf)
	case *StarExpr:
		cl, clDoc, gol, goc, out = genGoPostStar(fn, indent, captureName, v.X, onlyIf)
	case *SelectorExpr:
		cl, clDoc, gol, goc, out = genGoPostSelector(fn, indent, captureName, v, onlyIf)
	case *InterfaceType:
		out = "MakeGoObjectIfNeeded(" + captureName + ")"
		cl = "Object"
	case *MapType, *ChanType:
		out = "MakeGoObject(" + captureName + ")"
		cl = "GoObject"
	default:
		cl = fmt.Sprintf("ABEND883(post.go: unrecognized Expr type %T at: %s)", e, Unix(WhereAt(e.Pos())))
		gol = "..."
		out = captureName
	}

	if gol == "" {
		ty, tyName := types.TypeLookup(e)
		if ty == nil {
			gol = tyName + "ABEND000(post.go: no type info found)"
		} else {
			gol = ty.RelativeGoName(e.Pos())
		}
	}

	if clDoc == "" {
		clDoc = cl
	}
	return
}

const resultName = "_res"

func genGoPostItem(fn *gowalk.FuncInfo, indent, captureName string, f *Field, onlyIf string) (captureVar, cl, clDoc, gol, goc, out string, useful bool) {
	captureVar = captureName
	if captureName == "" || captureName == "_" {
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
func genGoPostList(fn *gowalk.FuncInfo, indent string, fl *FieldList) (cl, clDoc, gol, goc, out string) {
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

func genGoPost(fn *gowalk.FuncInfo, indent string, t *FuncType) (goResultAssign, clojureReturnType, clojureReturnTypeForDoc, goReturnTypeForDoc, goReturnCode string) {
	fl := t.Results
	if fl == nil || fl.List == nil {
		return
	}
	clojureReturnType, clojureReturnTypeForDoc, goReturnTypeForDoc, goReturnCode, goResultAssign = genGoPostList(fn, indent, fl)
	return
}
