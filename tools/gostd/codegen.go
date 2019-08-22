package main

import (
	"fmt"
	. "go/ast"
	"path"
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

	fc.clojureParamList, fc.clojureParamListDoc, fc.clojureGoParams, fc.goParamList, fc.goParamListDoc, goPreCode, goParams, _, _ =
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

func genReceiverCode(fn *funcInfo, goFname string) string {
	const arityTemplate = `
	%sCheckGoArity("%s", args, %d, %d)
	`

	cljParamList, cljParamListDoc, cljGoParams, paramList, paramListDoc, preCode, params, min, max := genGoPre(fn, "\t", fn.fd.Type.Params, goFname)
	if strings.Contains(paramListDoc, "ABEND") {
		return paramListDoc
	}
	if strings.Contains(paramList, "ABEND") {
		return paramList
	}
	if strings.Contains(cljParamListDoc, "ABEND") {
		return cljParamListDoc
	}
	if strings.Contains(cljParamList, "ABEND") {
		return cljParamList
	}
	if strings.Contains(cljGoParams, "ABEND") {
		return cljGoParams
	}

	receiverName := fn.fd.Name.Name
	call := fmt.Sprintf("o.O.(%s).%s(%s)", fn.receiverId, receiverName, params)

	resultAssign, cljReturnType, cljReturnTypeForDoc, returnTypeForDoc, postCode := genGoPost(fn, "\t", fn.fd)
	if strings.Contains(returnTypeForDoc, "ABEND") {
		return returnTypeForDoc
	}
	if strings.Contains(cljReturnType, "ABEND") {
		return cljReturnType
	}
	if strings.Contains(cljReturnTypeForDoc, "ABEND") {
		return cljReturnTypeForDoc
	}
	if postCode == "" && resultAssign == "" {
		return "\t...ABEND275: TODO...\n"
	}
	finishPreCode := ""
	if preCode != "" {
		finishPreCode = "\n\t"
	}
	maybeAssignArgList := ""
	if max > 0 {
		maybeAssignArgList = "_argList := "
	}
	arity := fmt.Sprintf(arityTemplate[1:], maybeAssignArgList, fn.docName, min, max)
	return arity + preCode + finishPreCode + resultAssign + call + "\n" + postCode
}

func typeKey(pkgPrefix string, fl *Field) string {
	t := ""
	suffix := ""
	switch x := fl.Type.(type) {
	case *Ident:
		t = "*" + pkgPrefix + x.Name
		suffix = ".Elem()"
	case *StarExpr:
		t = "*" + pkgPrefix + x.X.(*Ident).Name
	default:
		panic(fmt.Sprintf("typeInfoName: unrecognized expr %T", x))
	}
	return fmt.Sprintf("_reflect.TypeOf((%s)(nil))%s", t, suffix)
}

func typeInfoName(fl *Field) string {
	res := ""
	switch x := fl.Type.(type) {
	case *Ident:
		res += x.Name
	case *ArrayType:
		res += "ArrayOf_" + x.Elt.(*Ident).Name
	case *StarExpr:
		res += "PtrTo_" + x.X.(*Ident).Name
	default:
		panic(fmt.Sprintf("typeInfoName: unrecognized expr %T", x))
	}
	return "members_" + res
}

func genReceiver(fn *funcInfo) {
	genSymReset()
	pkgDirUnix := fn.sourceFile.pkgDirUnix
	pkgBaseName := fn.sourceFile.pkgBaseName

	const goTemplate = `
func %s(o GoObject, args Object) Object {
%s}
`

	goFname := funcNameAsGoPrivate(fn.name)

	clojureFn := ""

	goFn := fmt.Sprintf(goTemplate, goFname, genReceiverCode(fn, goFname))

	if strings.Contains(clojureFn, "ABEND") || strings.Contains(goFn, "ABEND") {
		clojureFn = nonEmptyLineRegexp.ReplaceAllString(clojureFn, `;; $1`)
		goFn = nonEmptyLineRegexp.ReplaceAllString(goFn, `// $1`)
		trackAbends(clojureFn)
		trackAbends(goFn)
	} else {
		numGeneratedFunctions++
		numGeneratedReceivers++
		packagesInfo[pkgDirUnix].nonEmpty = true
		addImport(packagesInfo[pkgDirUnix].importsNative, ".", "github.com/candid82/joker/core", false)
		addImport(packagesInfo[pkgDirUnix].importsNative, "_"+pkgBaseName, pkgDirUnix, false)
		addImport(packagesInfo[pkgDirUnix].importsNative, "_reflect", "reflect", false)
		for _, r := range fn.fd.Recv.List {
			tin := typeInfoName(r)
			goCode[pkgDirUnix].initTypes[typeKey("_"+pkgBaseName+".", r)] = tin
			if _, ok := goCode[pkgDirUnix].initVars[tin]; !ok {
				goCode[pkgDirUnix].initVars[tin] = map[string]string{}
			}
			goCode[pkgDirUnix].initVars[tin][fn.fd.Name.Name] = goFname
		}
	}

	if goFn != "" {
		goCode[pkgDirUnix].functions[goFname] = fnCodeInfo{fn.sourceFile, goFn}
	}
}

func genStandalone(fn *funcInfo) {
	genSymReset()
	d := fn.fd
	pkgDirUnix := fn.sourceFile.pkgDirUnix
	pkgBaseName := fn.sourceFile.pkgBaseName

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
		numGeneratedFunctions++
		numGeneratedStandalones++
		packagesInfo[pkgDirUnix].nonEmpty = true
		if clojureReturnType == "" {
			addImport(packagesInfo[pkgDirUnix].importsNative, ".", "github.com/candid82/joker/core", false)
			addImport(packagesInfo[pkgDirUnix].importsNative, "_"+pkgBaseName, pkgDirUnix, false)
		}
		if clojureReturnType != "" || fn.refersToSelf {
			addImport(packagesInfo[pkgDirUnix].importsAutoGen, "", pkgDirUnix, false)
		}
	}

	clojureCode[pkgDirUnix].functions[d.Name.Name] = fnCodeInfo{fn.sourceFile, clojureFn}

	if goFn != "" {
		goCode[pkgDirUnix].functions[d.Name.Name] = fnCodeInfo{fn.sourceFile, goFn}
	}
}

func genConstant(ci *constantInfo) {
	genSymReset()
	pkgDirUnix := ci.sourceFile.pkgDirUnix

	clojureCode[pkgDirUnix].constants[ci.name.Name] = ci

	addImport(packagesInfo[pkgDirUnix].importsAutoGen, "", pkgDirUnix, false)
}

func maybeImplicitConvert(src *goFile, typeName string, ts *TypeSpec) string {
	t := toGoTypeInfo(src, ts)
	if t == nil || t.custom {
		return ""
	}
	argType := t.argClojureArgType
	declType := t.argExtractFunc
	if declType == "" {
		return ""
	}
	const exTemplate = `case %s:
		v := _%s(Extract%s(args, index))
		return &v
	`
	return fmt.Sprintf(exTemplate, argType, typeName, declType)
}

func addressOf(ptrTo string) string {
	if ptrTo == "*" {
		return "&"
	}
	return ""
}

func maybeDeref(ptrTo string) string {
	if ptrTo == "*" {
		return ""
	}
	return "*"
}

func goTypeExtractor(t string, ti *goTypeInfo) string {
	const template = `
func %s(rcvr, arg string, args *ArraySeq, n int) (res %s) {
	a := CheckGoNth(rcvr, "%s", arg, args, n).O
	res, ok := a.(%s)
	if !ok {
		panic(RT.NewError(%s.Sprintf("Argument %%d passed to %%s should be type GoObject[%s], but is GoObject[%%s]",
			n, rcvr, GoTypeToString(%s.TypeOf(a)))))
	}
	return
}
`

	mangled := typeToGoExtractFuncName(ti.argClojureArgType)
	localType := "_" + ti.sourceFile.pkgBaseName + "." + ti.localName
	typeDoc := ti.argClojureArgType // "path.filepath.Mode"

	fmtLocal := addImport(packagesInfo[ti.sourceFile.pkgDirUnix].importsNative, "", "fmt", true)
	reflectLocal := addImport(packagesInfo[ti.sourceFile.pkgDirUnix].importsNative, "", "reflect", true)

	fnName := "ExtractGo_" + mangled
	resType := localType
	resTypeDoc := typeDoc // or similar
	resType += ""         // repeated here
	fmtLocal += ""        //
	resTypeDoc += ""      // repeated here
	reflectLocal += ""    //

	return fmt.Sprintf(template, fnName, resType, resTypeDoc, resType, fmtLocal, resTypeDoc, reflectLocal)
}

func genType(t string, ti *goTypeInfo) {
	td := ti.td
	if isPrivate(td.Name.Name) {
		return // Do not generate anything for private types
	}
	pkgDirUnix := ti.sourceFile.pkgDirUnix
	pkgBaseName := ti.sourceFile.pkgBaseName
	if pi, found := packagesInfo[pkgDirUnix]; !found {
		return // no public functions available for package, so don't try to generate type info either
	} else if !pi.nonEmpty {
		return // no functions generated
	}

	addImport(packagesInfo[pkgDirUnix].importsNative, ".", "github.com/candid82/joker/core", false)
	addImport(packagesInfo[pkgDirUnix].importsNative, "_"+pkgBaseName, pkgDirUnix, false)

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
	baseTypeName := td.Name.Name

	others := maybeImplicitConvert(ti.sourceFile, typeName, td)
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
%sfunc _Construct%s(_v Object) %s_%s {
	switch _o := _v.(type) {
	case GoObject:
		switch _g := _o.O.(type) {
		case _%s:
			return %s_g
		case *_%s:
			return %s_g
		}
	%s
	}
	panic(RT.NewArgTypeError(0, _v, "%s"))
}
`

	nonGoObject, expectedObjectDoc, helperFunc, ptrTo := nonGoObjectCase(ti, typeName, baseTypeName)
	goConstructor := fmt.Sprintf(goConstructTemplate, helperFunc, baseTypeName, ptrTo, typeName, typeName, addressOf(ptrTo), typeName, maybeDeref(ptrTo), nonGoObject, expectedObjectDoc)

	if strings.Contains(ti.clojureCode, "ABEND") || strings.Contains(goConstructor, "ABEND") {
		ti.clojureCode = nonEmptyLineRegexp.ReplaceAllString(ti.clojureCode, `;; $1`)
		goConstructor = nonEmptyLineRegexp.ReplaceAllString(goConstructor, `// $1`)
		trackAbends(ti.clojureCode)
		trackAbends(goConstructor)
	} else {
		numGeneratedTypes++
		promoteImports(ti)
	}

	ti.goCode += goConstructor

	ti.goCode += goTypeExtractor(t, ti)
}

func promoteImports(ti *goTypeInfo) {
	for _, imp := range ti.requiredImports.fullNames {
		addImport(packagesInfo[ti.sourceFile.pkgDirUnix].importsNative, imp.local, imp.full, false)
	}
}

func nonGoObjectCase(ti *goTypeInfo, typeName, baseTypeName string) (nonGoObjectCase, nonGoObjectCaseDoc, helperFunc, ptrTo string) {
	const nonGoObjectCaseTemplate = `%s:
		return %s`

	nonGoObjectTypes, nonGoObjectTypeDocs, extractClojureObjects, helperFuncs, ptrTo := nonGoObjectTypeFor(ti, typeName, baseTypeName)

	nonGoObjectCasePrefix := ""
	nonGoObjectCase = ""
	for i := 0; i < len(nonGoObjectTypes); i++ {
		nonGoObjectCase += nonGoObjectCasePrefix + fmt.Sprintf(nonGoObjectCaseTemplate, nonGoObjectTypes[i], extractClojureObjects[i])
		nonGoObjectCasePrefix = `
	`
	}

	return nonGoObjectCase,
		fmt.Sprintf("GoObject[%s] or: %s", typeName, strings.Join(nonGoObjectTypeDocs, " or ")),
		strings.Join(helperFuncs, ""),
		ptrTo
}

func nonGoObjectTypeFor(ti *goTypeInfo, typeName, baseTypeName string) (nonGoObjectTypes, nonGoObjectTypeDocs, extractClojureObjects, helperFuncs []string, ptrTo string) {
	switch t := ti.td.Type.(type) {
	case *Ident:
		nonGoObjectType, nonGoObjectTypeDoc, extractClojureObject := simpleTypeFor(ti.sourceFile.pkgDirUnix, t.Name, &ti.td.Type)
		extractClojureObject = "_" + typeName + "(_o" + extractClojureObject + ")"
		nonGoObjectTypes = []string{nonGoObjectType}
		nonGoObjectTypeDocs = []string{nonGoObjectTypeDoc}
		extractClojureObjects = []string{extractClojureObject}
		if nonGoObjectType != "" {
			return
		}
	case *StructType:
		uniqueTypeName := "_" + typeName
		mapHelperFName := "_mapTo" + baseTypeName
		vectorHelperFName := "_vectorTo" + baseTypeName
		return []string{"case *ArrayMap, *HashMap", "case *Vector"},
			[]string{"Map", "Vector"},
			[]string{mapHelperFName + "(_o.(Map))", vectorHelperFName + "(_o)"},
			[]string{mapToType(ti, mapHelperFName, uniqueTypeName, t),
				vectorToType(ti, vectorHelperFName, uniqueTypeName, t)},
			"*"
	case *ArrayType:
	}
	return []string{"default"},
		[]string{"whatever"},
		[]string{fmt.Sprintf("_%s(_o.ABEND674(codegen.go: unknown underlying type %T for %s))",
			typeName, ti.td.Type, ti.td.Name)},
		[]string{""},
		""
}

func simpleTypeFor(pkgDirUnix, name string, e *Expr) (nonGoObjectType, nonGoObjectTypeDoc, extractClojureObject string) {
	v := toGoTypeNameInfo(pkgDirUnix, name, e)
	nonGoObjectType = "case " + v.argClojureType
	nonGoObjectTypeDoc = v.argClojureType
	extractClojureObject = v.argFromClojureObject
	if v.unsupported || v.argClojureType == "" || extractClojureObject == "" {
		nonGoObjectType += fmt.Sprintf(" /* ABEND171(missing go object type or clojure-object extraction for %s) */", v.fullGoName)
	}
	return
}

func mapToType(ti *goTypeInfo, helperFName, typeName string, ty *StructType) string {
	const hFunc = `func %s(o Map) *%s {
	return &%s{%s}
}

`
	return fmt.Sprintf(hFunc, helperFName, typeName, typeName, "")
}

func vectorToType(ti *goTypeInfo, helperFName, typeName string, ty *StructType) string {
	const hFunc = `func %s(o *Vector) *%s {
	return &%s{%s}
}

`

	elToType := elementsToType(ti, ty, vectorElementToType)
	if elToType != "" {
		elToType = `
		` + elToType + `
	`
	}

	return fmt.Sprintf(hFunc, helperFName, typeName, typeName, elToType)
}

func elementsToType(ti *goTypeInfo, ty *StructType, toType func(ti *goTypeInfo, i int, name string, f *Field) string) string {
	els := []string{}
	i := 0
	for _, f := range ty.Fields.List {
		for _, p := range f.Names {
			fieldName := p.Name
			if fieldName == "" || isPrivate(fieldName) {
				continue
			}
			els = append(els, fmt.Sprintf("%s: %s,", fieldName, toType(ti, i, p.Name, f)))
			i++
		}
	}
	return strings.Join(els, `
		`)
}

func vectorElementToType(ti *goTypeInfo, i int, name string, f *Field) string {
	return elementToType(ti, fmt.Sprintf("o.Nth(%d)", i), &f.Type)
}

func elementToType(ti *goTypeInfo, el string, e *Expr) string {
	v := toGoExprInfo(ti.sourceFile, e)
	if v.unsupported {
		return v.fullGoName
	}
	if v.convertFromClojure != "" {
		addRequiredImports(ti, v.convertFromClojureImports)
		return fmt.Sprintf(v.convertFromClojure, el)
	}
	return fmt.Sprintf("ABEND048(codegen.go: no conversion from Clojure for %s (%s))",
		v.fullGoName, toGoExprString(ti.sourceFile, v.underlyingType))
}

// Add the list of imports to those required if this type's constructor can be emitted (no ABENDs).
func addRequiredImports(ti *goTypeInfo, imports []packageImport) {
	for _, imp := range imports {
		addImport(ti.requiredImports, imp.local, imp.full, false)
	}
}
