package main

import (
	"fmt"
	. "go/ast"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

/* The transformation code, below, takes an approach that is new for me.

   Instead of each transformation point having its own transform
   routine(s), as is customary, I'm trying an approach in which the
   transform is driven by the input and multiple outputs are
   generated, where appropriate, for further processing and/or
   insertion into the ultimate transformation points.

   The primary reason for this is that the input is complicated and
   (generally) being supported to a greater extent as enhancements are
   made. I want to maintain coherence among the various transformation
   insertions, so it's less likely that a change made for one
   insertion point (to support a new input form, or modify an existing
   one) won't have corresponding changes made to other forms relying
   on the same essential input, which could lead to coding errors.

   This approach also should make it easier to see how the different
   snippets of code, relating to one particular aspect of the input,
   relate to each other, because the code will be in the same place.

   Further, it should be easier to make and track decisions (such as
   what will be the names of temporary variables) by doing it in one
   place, rather than having to make a "decision pass" first, memoize
   the results, and pass them around to the various transformation
   routines.

   However, I'm concerned that the resulting code will be too
   complicated for that to be sufficiently helpful. If I was
   proficient in a constraint/unification-based transformation
   language, I'd look at that instead, because it would allow me to
   express that e.g. "func foo(args) (returns) { ...do things with
   args...; call foo in some fashion; ...do things with returns... }"
   not only have specific transformations for each of the variables
   involved, but that they are also constrained in some fashion
   (e.g. whatever names are picked for unnamed 'returns' values are
   the same in both "returns" and "do things with returns"; whatever
   types are involved in both "args" and "returns" are properly
   processed in "do things with args" and "do things with returns",
   respectively; and so on).

   Now that I've refactored the code to achieve this, I'll start
   adding transformations and see how it goes. Might revert to
   old-fashioned use of custom transformation code per point (sharing
   code where appropriate, of course) if it gets too hairy.

*/

var nonEmptyLineRegexp *regexp.Regexp

/*
   (defn <clojureReturnType> <godecl.Name>
     <docstring>
     {:added "1.0"
      :go "<cl2golcall>"}  ; cl2golcall := <conversion?>(<cl2gol>+<clojureGoParams>)
     [clojureParamList])   ; clojureParamList

   func <goFname>(<goParamList>) <goReturnType> {  // goParamList
           <goCode>                                // goCode := <goPreCode>+"\t"+<goResultAssign>+"_"+pkg+"."+<godecl.Name>+"("+<goParams>+")\n"+<goPostCode>
   }

*/

type funcCode struct {
	clojureParamList        string
	clojureParamListDoc     string
	goParamList             string
	goParamListDoc          string
	clojureGoParams         string
	goCode                  string
	clojureReturnType       string
	clojureReturnTypeForDoc string
	goReturnTypeForDoc      string
}

/* IMPORTANT: The public functions listed herein should be only those
   defined in joker/core/custom-runtime.go.

   That's how gostd knows to not actually generate calls to
   as-yet-unimplemented (or stubbed-out) functions, saving the
   developer the hassle of getting most of the way through a build
   before hitting undefined-func errors.
*/
var customRuntimeImplemented = map[string]struct{}{
	"ConvertToArrayOfByte":   {},
	"ConvertToArrayOfInt":    {},
	"ConvertToArrayOfString": {},
}

func genGoCall(pkgBaseName, goFname, goParams string) string {
	return "_" + pkgBaseName + "." + goFname + "(" + goParams + ")\n"
}

func genFuncCode(fn *funcInfo, pkgBaseName, pkgDirUnix string, d *FuncDecl, goFname string) (fc funcCode) {
	var goPreCode, goParams, goResultAssign, goPostCode string

	fc.clojureParamList, fc.clojureParamListDoc, fc.clojureGoParams, fc.goParamList, fc.goParamListDoc, goPreCode, goParams =
		genGoPre(fn, "\t", d.Type.Params, goFname)
	goCall := genGoCall(pkgBaseName, d.Name.Name, goParams)
	goResultAssign, fc.clojureReturnType, fc.clojureReturnTypeForDoc, fc.goReturnTypeForDoc, goPostCode = genGoPost(fn, "\t", d)

	if goPostCode == "" && goResultAssign == "" {
		goPostCode = "\t...ABEND675: TODO...\n"
	}

	fc.goCode = goPreCode + // Optional block of pre-code
		"\t" + goResultAssign + goCall + // [results := ]fn-to-call([args...])
		goPostCode // Optional block of post-code
	return
}

var abendRegexp *regexp.Regexp

var abends = map[string]int{}

func trackAbends(a string) {
	subMatches := abendRegexp.FindAllStringSubmatch(a, -1)
	//	fmt.Printf("trackAbends: %v %s => %#v\n", abendRegexp, a, subMatches)
	for _, m := range subMatches {
		if len(m) != 2 {
			panic(fmt.Sprintf("len(%v) != 2", m))
		}
		n := m[1]
		if _, ok := abends[n]; !ok {
			abends[n] = 0
		}
		abends[n] += 1
	}
}

func printAbends(m map[string]int) {
	type ac struct {
		abendCode  string
		abendCount int
	}
	a := []ac{}
	for k, v := range m {
		a = append(a, ac{abendCode: k, abendCount: v})
	}
	sort.Slice(a,
		func(i, j int) bool {
			if a[i].abendCount == a[j].abendCount {
				return a[i].abendCode < a[j].abendCode
			}
			return a[i].abendCount > a[j].abendCount
		})
	for _, v := range a {
		fmt.Printf(" %s(%d)", v.abendCode, v.abendCount)
	}
}

func genFunction(fn *funcInfo) {
	genSymReset()
	d := fn.fd
	pkgDirUnix := fn.sourceFile.pkgDirUnix
	pkgBaseName := filepath.Base(pkgDirUnix)

	const clojureTemplate = `
(defn %s%s
%s  {:added "1.0"
   :go "%s"}
  [%s])
`
	goFname := funcNameAsGoPrivate(d.Name.Name)
	fc := genFuncCode(fn, pkgBaseName, pkgDirUnix, d, goFname)
	clojureReturnType, goReturnType := clojureReturnTypeForGenerateCustom(fc.clojureReturnType, fc.goReturnTypeForDoc)

	var cl2gol string
	if clojureReturnType == "" {
		cl2gol = goFname
	} else {
		clojureReturnType += " "
		cl2gol = pkgBaseName + "." + d.Name.Name
		if _, found := packagesInfo[pkgDirUnix]; !found {
			panic(fmt.Sprintf("Cannot find package %s", pkgDirUnix))
		}
	}
	cl2golCall := cl2gol + fc.clojureGoParams

	clojureFn := fmt.Sprintf(clojureTemplate, clojureReturnType, d.Name.Name,
		commentGroupInQuotes(d.Doc, fc.clojureParamListDoc, fc.clojureReturnTypeForDoc,
			fc.goParamListDoc, fc.goReturnTypeForDoc),
		cl2golCall, fc.clojureParamList)

	const goTemplate = `
func %s(%s) %s {
%s}
`

	goFn := fmt.Sprintf(goTemplate, goFname, fc.goParamList, goReturnType, fc.goCode)
	if clojureReturnType != "" && !strings.Contains(clojureFn, "ABEND") && !strings.Contains(goFn, "ABEND") {
		goFn = ""
	}

	if strings.Contains(clojureFn, "ABEND") || strings.Contains(goFn, "ABEND") {
		clojureFn = nonEmptyLineRegexp.ReplaceAllString(clojureFn, `;; $1`)
		goFn = nonEmptyLineRegexp.ReplaceAllString(goFn, `// $1`)
		trackAbends(clojureFn)
		trackAbends(goFn)
	} else {
		generatedFunctions++
		packagesInfo[pkgDirUnix].nonEmpty = true
		if clojureReturnType == "" {
			packagesInfo[pkgDirUnix].importsNative[pkgDirUnix] = exists
		}
		if clojureReturnType != "" || fn.refersToSelf {
			packagesInfo[pkgDirUnix].importsAutoGen[pkgDirUnix] = exists
		}
	}

	clojureCode[pkgDirUnix].functions[d.Name.Name] = fnCodeInfo{fn.sourceFile, clojureFn}

	if goFn != "" {
		goCode[pkgDirUnix].functions[d.Name.Name] = fnCodeInfo{fn.sourceFile, goFn}
	}
}

func maybeImplicitConvert(typeName string, td *TypeSpec) string {
	var declType string
	var argType string
	switch t := td.Type.(type) {
	case *Ident:
		switch t.Name {
		case "string":
			argType = "String"
			declType = "String"
		case "int":
			argType = "Int"
			declType = "Int"
		case "byte":
			argType = "Int"
			declType = "Byte"
		case "bool":
			argType = "Bool"
			declType = "Bool"
		case "int8":
			argType = "Int"
			declType = "Byte"
		case "int16":
			argType = "Int"
			declType = "Int16"
		case "uint":
			argType = "Number"
			declType = "UInt"
		case "uint8":
			argType = "Int"
			declType = "UInt8"
		case "uint16":
			argType = "Int"
			declType = "UInt16"
		case "int32":
			argType = "Int"
			declType = "Int32"
		case "uint32":
			argType = "Number"
			declType = "UInt32"
		case "int64":
			argType = "Number"
			declType = "Int64"
		case "uintptr":
			argType = "Number"
			declType = "UIntPtr"
		case "float32":
			argType = "Double"
			declType = "ABEND007(find these)"
		case "float64":
			argType = "Double"
			declType = "ABEND007(find these)"
		case "complex64":
			argType = "Double"
			declType = "ABEND007(find these)"
		case "complex128":
			argType = "Double"
			declType = "ABEND007(find these)"
		}
	}
	if declType == "" {
		return ""
	}
	const exTemplate = `case %s:
		v := _%s(Extract%s(args, index))
		return &v
	`
	return fmt.Sprintf(exTemplate, argType, typeName, declType)
}

func genType(t string, ti *typeInfo) {
	pkgDirUnix := ti.sourceFile.pkgDirUnix
	if pi, found := packagesInfo[pkgDirUnix]; !found {
		return // no public functions available for package, so don't try to generate type info either
	} else if !pi.nonEmpty {
		return // no functions generated
	}

	packagesInfo[pkgDirUnix].importsNative[pkgDirUnix] = exists

	clojureCode[pkgDirUnix].types[t] = ti
	goCode[pkgDirUnix].types[t] = ti

	const goExtractTemplate = `
func ExtractGoObject%s(args []Object, index int) *_%s {
	a := args[index]
	switch o := a.(type) {
	case GoObject:
		switch r := o.O.(type) {
		case _%s:
			return &r
		case *_%s:
			return r
		}
	%s}
	panic(RT.NewArgTypeError(index, a, "GoObject[%s]"))
}
`

	typeName := path.Base(t)
	baseTypeName := ti.td.Name.Name

	others := maybeImplicitConvert(typeName, ti.td)
	ti.goCode = fmt.Sprintf(goExtractTemplate, baseTypeName, typeName, typeName, typeName, others, t)

	const clojureTemplate = `
(defn ^"GoObject" %s.
  "Constructor for %s"
  {:added "1.0"
   :go "_Construct%s(_v)"}
  [^Object _v])
`

	ti.clojureCode = fmt.Sprintf(clojureTemplate, baseTypeName, typeName, baseTypeName)

	const goConstructTemplate = `
%sfunc _Construct%s(_v Object) _%s {
	switch _o := _v.(type) {
	case GoObject:
		switch _g := _o.O.(type) {
		case _%s:
			return _g
		}
	%s
	}
	panic(RT.NewArgTypeError(0, _v, "%s"))
}
`

	nonGoObject, expectedObjectDoc, helperFunc := nonGoObjectCase(typeName, baseTypeName, ti)
	goConstructor := fmt.Sprintf(goConstructTemplate, helperFunc, baseTypeName, typeName, typeName, nonGoObject, expectedObjectDoc)

	if strings.Contains(ti.clojureCode, "ABEND") || strings.Contains(goConstructor, "ABEND") {
		ti.clojureCode = nonEmptyLineRegexp.ReplaceAllString(ti.clojureCode, `;; $1`)
		goConstructor = nonEmptyLineRegexp.ReplaceAllString(goConstructor, `// $1`)
		trackAbends(ti.clojureCode)
		trackAbends(goConstructor)
	}

	ti.goCode += goConstructor
}

func nonGoObjectCase(typeName, baseTypeName string, ti *typeInfo) (string, string, string) {
	const nonGoObjectCaseTemplate = `%s:
		return %s`

	nonGoObjectType, nonGoObjectTypeDoc, extractClojureObject, helperFunc := nonGoObjectTypeFor(typeName, baseTypeName, ti)

	return fmt.Sprintf(nonGoObjectCaseTemplate, nonGoObjectType, extractClojureObject), fmt.Sprintf("GoObject[%s] or %s", typeName, nonGoObjectTypeDoc), helperFunc
}

func nonGoObjectTypeFor(typeName, baseTypeName string, ti *typeInfo) (nonGoObjectType, nonGoObjectTypeDoc, extractClojureObject, helperFunc string) {
	switch t := ti.td.Type.(type) {
	case *Ident:
		nonGoObjectType, nonGoObjectTypeDoc, extractClojureObject = simpleTypeFor(t.Name)
		extractClojureObject = "_" + typeName + "(_o" + extractClojureObject + ")"
		if nonGoObjectType != "" {
			return
		}
	case *StructType:
		helperFName := "_mapTo" + baseTypeName
		return "case *ArrayMap, *HashMap", "Map", helperFName + "(_o)", mapToType(helperFName, "_"+typeName, ti.td.Type)
	}
	return "default", "whatever", fmt.Sprintf("_%s(_o.ABEND674(unknown underlying type %T for %s))", typeName, ti.td.Type, ti.td.Name), ""
}

func simpleTypeFor(name string) (nonGoObjectType, nonGoObjectTypeDoc, extractClojureObject string) {
	switch name {
	case "string":
		return "case String", "String", ".S"
	case "bool":
		return "case Bool", "Bool", ".Bool().B"
	case "int", "byte", "int8", "int16", "uint", "uint8", "uin16", "int32", "uint32":
		return "case Number", "Number", ".Int().I"
	case "int64":
		return "case Number", "Number", ".BigInt().Int64()"
	case "uint64", "uintptr":
		return "case Number", "Number", ".BigInt().Uint64()"
	}
	return
}

func mapToType(helperFName, typeName string, ty Expr) string {
	const hFunc = `func %s(o Map) %s {
	return %s{
%s	}
}

`
	return fmt.Sprintf(hFunc, helperFName, typeName, typeName, "")
}
