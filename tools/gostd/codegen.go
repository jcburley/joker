package main

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/abends"
	"github.com/candid82/joker/tools/gostd/godb"
	. "github.com/candid82/joker/tools/gostd/gowalk"
	"github.com/candid82/joker/tools/gostd/imports"
	. "github.com/candid82/joker/tools/gostd/types"
	. "github.com/candid82/joker/tools/gostd/utils"
	. "go/ast"
	"path"
	"regexp"
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

func genFuncCode(fn *FuncInfo, pkgBaseName, pkgDirUnix string, t *FuncType, goFname string) (fc funcCode) {
	var goPreCode, goParams, goResultAssign, goPostCode string

	fc.clojureParamList, fc.clojureParamListDoc, fc.clojureGoParams, fc.goParamList, fc.goParamListDoc, goPreCode, goParams, _, _ =
		genGoPre(fn, "\t", t.Params, goFname)
	goCall := genGoCall(pkgBaseName, fn.BaseName, goParams)
	goResultAssign, fc.clojureReturnType, fc.clojureReturnTypeForDoc, fc.goReturnTypeForDoc, goPostCode = genGoPost(fn, "\t", t)

	if goPostCode == "" && goResultAssign == "" {
		goPostCode = "\t...ABEND675: TODO...\n"
	}

	fc.goCode = goPreCode + // Optional block of pre-code
		"\t" + goResultAssign + goCall + // [results := ]fn-to-call([args...])
		goPostCode // Optional block of post-code
	return
}

func genReceiverCode(fn *FuncInfo, goFname string) string {
	const arityTemplate = `
	%sCheckGoArity("%s", args, %d, %d)
	`

	cljParamList, cljParamListDoc, cljGoParams, paramList, paramListDoc, preCode, params, min, max := genGoPre(fn, "\t", fn.Ft.Params, goFname)
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

	receiverName := fn.BaseName
	call := fmt.Sprintf("o.O.(%s).%s(%s)", fn.ReceiverId, receiverName, params)

	resultAssign, cljReturnType, cljReturnTypeForDoc, returnTypeForDoc, postCode := genGoPost(fn, "\t", fn.Ft)
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
	arity := fmt.Sprintf(arityTemplate[1:], maybeAssignArgList, fn.DocName, min, max)
	return arity + preCode + finishPreCode + resultAssign + call + "\n" + postCode
}

func GenReceiver(fn *FuncInfo) {
	genSymReset()
	pkgDirUnix := fn.SourceFile.Package.DirUnix
	pkgBaseName := fn.SourceFile.Package.BaseName

	const goTemplate = `
func %s(o GoObject, args Object) Object {  // %s
%s}
`

	goFname := funcNameAsGoPrivate(fn.Name)

	if !IsExported(fn.BaseName) {
		return
	}

	clojureFn := ""

	what := "Receiver"
	if fn.Fd == nil {
		what = "Method"
	}
	goFn := fmt.Sprintf(goTemplate, goFname, what, genReceiverCode(fn, goFname))

	if strings.Contains(clojureFn, "ABEND") || strings.Contains(goFn, "ABEND") {
		clojureFn = nonEmptyLineRegexp.ReplaceAllString(clojureFn, `;; $1`)
		goFn = nonEmptyLineRegexp.ReplaceAllString(goFn, `// $1`)
		abends.TrackAbends(clojureFn)
		abends.TrackAbends(goFn)
		if fn.Fd == nil {
			NumFunctions++
			godb.NumMethods++
		}
	} else {
		NumGeneratedFunctions++
		PackagesInfo[pkgDirUnix].NonEmpty = true
		imports.AddImport(PackagesInfo[pkgDirUnix].ImportsNative, ".", "github.com/candid82/joker/core", false)
		imports.AddImport(PackagesInfo[pkgDirUnix].ImportsNative, "_"+pkgBaseName, pkgDirUnix, false)
		imports.AddImport(PackagesInfo[pkgDirUnix].ImportsNative, "_reflect", "reflect", true)
		if fn.Fd == nil {
			NumFunctions++
			godb.NumMethods++
			godb.NumGeneratedMethods++
			tdi := fn.ToM
			if tdi == nil {
				panic(fmt.Sprintf("Cannot find type for %s", fn.Name))
			}
			if _, ok := GoCode[pkgDirUnix].InitVars[tdi]; !ok {
				GoCode[pkgDirUnix].InitVars[tdi] = map[string]*FnCodeInfo{}
			}
			GoCode[pkgDirUnix].InitVars[tdi][fn.BaseName] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFname, Params: fn.Ft.Params, FnDoc: fn.Doc}
		} else {
			NumGeneratedReceivers++
			for _, r := range fn.Fd.Recv.List {
				tdi, tdiFullName := TypeLookup(r.Type)
				if tdi == nil {
					panic(fmt.Sprintf("nil tdi for %s!!", tdiFullName))
				}
				if _, ok := GoCode[pkgDirUnix].InitVars[tdi]; !ok {
					GoCode[pkgDirUnix].InitVars[tdi] = map[string]*FnCodeInfo{}
				}
				GoCode[pkgDirUnix].InitVars[tdi][fn.BaseName] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFname, FnDecl: fn.Fd, Params: fn.Fd.Type.Params, FnDoc: fn.Doc}
			}
		}
	}

	if goFn != "" {
		var params *FieldList
		if fn.Fd != nil {
			params = fn.Fd.Type.Params
		}
		GoCode[pkgDirUnix].Functions[goFname] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFn, FnDecl: fn.Fd, Params: params, FnDoc: nil}
	}
}

func GenStandalone(fn *FuncInfo) {
	genSymReset()
	d := fn.Fd
	pkgDirUnix := fn.SourceFile.Package.DirUnix
	pkgBaseName := fn.SourceFile.Package.BaseName

	const clojureTemplate = `
(defn %s%s
%s  {:added "1.0"
   :go "%s"}
  [%s])
`
	goFname := funcNameAsGoPrivate(d.Name.Name)
	fc := genFuncCode(fn, pkgBaseName, pkgDirUnix, fn.Ft, goFname)
	clojureReturnType, goReturnType := clojureReturnTypeForGenerateCustom(fc.clojureReturnType, fc.goReturnTypeForDoc)

	var cl2gol string
	if clojureReturnType == "" {
		cl2gol = goFname
	} else {
		clojureReturnType += " "
		cl2gol = pkgBaseName + "." + fn.BaseName
		if _, found := PackagesInfo[pkgDirUnix]; !found {
			panic(fmt.Sprintf("Cannot find package %s", pkgDirUnix))
		}
	}
	cl2golCall := cl2gol + fc.clojureGoParams

	clojureFn := fmt.Sprintf(clojureTemplate, clojureReturnType, d.Name.Name,
		"  "+CommentGroupInQuotes(d.Doc, fc.clojureParamListDoc, fc.clojureReturnTypeForDoc,
			fc.goParamListDoc, fc.goReturnTypeForDoc)+"\n",
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
		abends.TrackAbends(clojureFn)
		abends.TrackAbends(goFn)
	} else {
		NumGeneratedFunctions++
		NumGeneratedStandalones++
		PackagesInfo[pkgDirUnix].NonEmpty = true
		if clojureReturnType == "" {
			imports.AddImport(PackagesInfo[pkgDirUnix].ImportsNative, ".", "github.com/candid82/joker/core", false)
			imports.AddImport(PackagesInfo[pkgDirUnix].ImportsNative, "_"+pkgBaseName, pkgDirUnix, false)
		}
		if clojureReturnType != "" || fn.RefersToSelf {
			imports.AddImport(PackagesInfo[pkgDirUnix].ImportsAutoGen, "", pkgDirUnix, false)
		}
	}

	ClojureCode[pkgDirUnix].Functions[d.Name.Name] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: clojureFn, FnDecl: nil, FnDoc: nil}

	if goFn != "" {
		GoCode[pkgDirUnix].Functions[d.Name.Name] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFn, FnDecl: nil, FnDoc: nil}
	}
}

func GenConstant(ci *ConstantInfo) {
	genSymReset()
	pkgDirUnix := ci.SourceFile.Package.DirUnix

	ClojureCode[pkgDirUnix].Constants[ci.Name.Name] = ci

	PackagesInfo[pkgDirUnix].NonEmpty = true

	imports.AddImport(PackagesInfo[pkgDirUnix].ImportsAutoGen, "", pkgDirUnix, false)
}

func GenVariable(ci *VariableInfo) {
	genSymReset()
	pkgDirUnix := ci.SourceFile.Package.DirUnix

	ClojureCode[pkgDirUnix].Variables[ci.Name.Name] = ci

	PackagesInfo[pkgDirUnix].NonEmpty = true

	imports.AddImport(PackagesInfo[pkgDirUnix].ImportsAutoGen, "", pkgDirUnix, false)
}

func maybeImplicitConvert(src *godb.GoFile, typeName string, ts *TypeSpec) string {
	t := toGoTypeInfo(src, ts)
	if t == nil || t.Custom {
		return ""
	}
	argType := t.ArgClojureArgType
	declType := t.ArgExtractFunc
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

func goTypeExtractor(t string, ti *GoTypeInfo) string {
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

	mangled := typeToGoExtractFuncName(ti.ArgClojureArgType)
	localType := "_" + ti.SourceFile.Package.BaseName + "." + ti.LocalName
	typeDoc := ti.ArgClojureArgType // "path.filepath.Mode"

	fmtLocal := imports.AddImport(PackagesInfo[ti.SourceFile.Package.DirUnix].ImportsNative, "", "fmt", true)
	reflectLocal := imports.AddImport(PackagesInfo[ti.SourceFile.Package.DirUnix].ImportsNative, "_reflect", "reflect", true)

	fnName := "ExtractGo_" + mangled
	resType := localType
	resTypeDoc := typeDoc // or similar
	resType += ""         // repeated here
	fmtLocal += ""        //
	resTypeDoc += ""      // repeated here
	reflectLocal += ""    //

	return fmt.Sprintf(template, fnName, resType, resTypeDoc, resType, fmtLocal, resTypeDoc, reflectLocal)
}

func GenType(t string, ti *GoTypeInfo) {
	td := ti.Td
	if !IsExported(td.Name.Name) {
		return // Do not generate anything for private or array types
	}

	pkgDirUnix := ti.SourceFile.Package.DirUnix
	pkgBaseName := ti.SourceFile.Package.BaseName
	pi := PackagesInfo[pkgDirUnix]

	pi.NonEmpty = true

	imports.AddImport(pi.ImportsNative, ".", "github.com/candid82/joker/core", false)
	imports.AddImport(pi.ImportsNative, "_"+pkgBaseName, pkgDirUnix, false)

	ClojureCode[pkgDirUnix].Types[t] = ti
	GoCode[pkgDirUnix].Types[t] = ti

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

	others := maybeImplicitConvert(ti.SourceFile, typeName, td)
	ti.GoCode = fmt.Sprintf(goExtractTemplate, baseTypeName, typeName, typeName, typeName, others, t)

	ti.GoCode += goTypeExtractor(t, ti)
}

var Ctors = map[*Type]string{}
var CtorNames = map[*Type]string{}

func genCtor(tdi *Type) {
	if tdi.TypeSpec == nil {
		return
	}

	const goConstructTemplate = `
%sfunc _Ctor_%s(_v Object) %s_%s {
	switch _o := _v.(type) {
	%s
	}
	panic(RT.NewArgTypeError(0, _v, "%s"))
}

func %s(_o Object) Object {
	return MakeGoObject(_Ctor_%s(_o))
}
`

	typeName := path.Base(tdi.GoPackage) + "." + tdi.GoName
	baseTypeName := tdi.GoName
	ctor := "_Wrapped_Ctor_" + baseTypeName
	nonGoObject, expectedObjectDoc, helperFunc, ptrTo := nonGoObjectCase(tdi, typeName, baseTypeName)
	goConstructor := fmt.Sprintf(goConstructTemplate, helperFunc, baseTypeName, ptrTo, typeName, nonGoObject, expectedObjectDoc,
		ctor, baseTypeName)

	ti := TypeDefsToGoTypes[tdi]

	if strings.Contains(goConstructor, "ABEND") {
		goConstructor = nonEmptyLineRegexp.ReplaceAllString(goConstructor, `// $1`)
		abends.TrackAbends(goConstructor)
	} else {
		promoteImports(ti)
		CtorNames[tdi] = ctor
		NumGeneratedCtors++
	}

	Ctors[tdi] = goConstructor
}

func appendMethods(tdi *Type, iface *InterfaceType) {
	for _, m := range iface.Methods.List {
		if m.Names != nil {
			if len(m.Names) != 1 {
				Print(godb.Fset, iface)
				panic("Names has more than one!")
			}
			if m.Type == nil {
				Print(godb.Fset, iface)
				panic("Why no Type field??")
			}
			for _, n := range m.Names {
				fullName := fmt.Sprintf(tdi.GoPattern, tdi.GoName) + "_" + n.Name
				doc := m.Doc
				if doc == nil {
					doc = m.Comment
				}
				QualifiedFunctions[fullName] = &FuncInfo{
					BaseName:     n.Name,
					ReceiverId:   "_" + tdi.GoFile.Package.BaseName + "." + tdi.GoName,
					Name:         fullName,
					DocName:      "(" + tdi.GoFile.Package.DirUnix + "." + tdi.GoName + ")" + n.Name + "()",
					Fd:           nil,
					ToM:          tdi,
					Ft:           m.Type.(*FuncType),
					Doc:          doc,
					SourceFile:   tdi.GoFile,
					RefersToSelf: false}
			}
			continue
		}
		ts := godb.Resolve(m.Type)
		if ts == nil {
			return
		}
		appendMethods(tdi, ts.(*TypeSpec).Type.(*InterfaceType))
	}
}

func GenTypeFromDb(tdi *Type) {
	if !tdi.IsExported || strings.Contains(tdi.FullName, "[") {
		return // Do not generate anything for private or array types
	}
	if tdi.Specificity == Concrete {
		genCtor(tdi)
		return // The code below currently handles only interface{} types
	}

	if tdi.TypeSpec != nil && tdi.TypeSpec.Type != nil {
		appendMethods(tdi, tdi.TypeSpec.Type.(*InterfaceType))
	}
}

func promoteImports(ti *GoTypeInfo) {
	for _, imp := range ti.RequiredImports.FullNames {
		imports.AddImport(PackagesInfo[ti.SourceFile.Package.DirUnix].ImportsNative, imp.Local, imp.Full, false)
	}
}

func nonGoObjectCase(tdi *Type, typeName, baseTypeName string) (nonGoObjectCase, nonGoObjectCaseDoc, helperFunc, ptrTo string) {
	const nonGoObjectCaseTemplate = `%s:
		return %s`

	nonGoObjectTypes, nonGoObjectTypeDocs, extractClojureObjects, helperFuncs, ptrTo := nonGoObjectTypeFor(tdi, typeName, baseTypeName)

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

func nonGoObjectTypeFor(tdi *Type, typeName, baseTypeName string) (nonGoObjectTypes, nonGoObjectTypeDocs, extractClojureObjects, helperFuncs []string, ptrTo string) {
	switch t := tdi.TypeSpec.Type.(type) {
	case *Ident:
		nonGoObjectType, nonGoObjectTypeDoc, extractClojureObject := simpleTypeFor(tdi.GoFile.Package.DirUnix, t.Name, tdi.TypeSpec.Type)
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
		return []string{"case *ArrayMap, *HashMap"},
			[]string{"Map"},
			[]string{mapHelperFName + "(_o.(Map))"},
			[]string{mapToType(tdi, mapHelperFName, uniqueTypeName, t)},
			"*"
	case *ArrayType:
	}
	return []string{"default"},
		[]string{"whatever"},
		[]string{fmt.Sprintf("_%s(_o.ABEND674(codegen.go: unknown underlying type %T for %s))",
			typeName, tdi.TypeSpec.Type, baseTypeName)},
		[]string{""},
		""
}

func simpleTypeFor(pkgDirUnix, name string, e Expr) (nonGoObjectType, nonGoObjectTypeDoc, extractClojureObject string) {
	v := toGoTypeNameInfo(pkgDirUnix, name, e)
	nonGoObjectType = "case " + v.ArgClojureType
	nonGoObjectTypeDoc = v.ArgClojureType
	extractClojureObject = v.ArgFromClojureObject
	if v.Unsupported || v.ArgClojureType == "" || extractClojureObject == "" {
		nonGoObjectType += fmt.Sprintf(" /* ABEND171(missing go object type or clojure-object extraction for %s) */", v.FullGoName)
	}
	return
}

func mapToType(tdi *Type, helperFName, typeName string, ty *StructType) string {
	const hFunc = `func %s(o Map) *%s {
	return &%s{%s}
}

`
	valToType := elementsToType(tdi, ty, mapElementToType)
	if valToType != "" {
		valToType = `
		` + valToType + `
	`
	}

	return fmt.Sprintf(hFunc, helperFName, typeName, typeName, valToType)
}

func elementsToType(tdi *Type, ty *StructType, toType func(tdi *Type, i int, name string, f *Field) string) string {
	els := []string{}
	i := 0
	for _, f := range ty.Fields.List {
		for _, p := range f.Names {
			fieldName := p.Name
			if fieldName == "" || !IsExported(fieldName) {
				continue
			}
			els = append(els, fmt.Sprintf("%s: %s,", fieldName, toType(tdi, i, p.Name, f)))
			i++
		}
	}
	return strings.Join(els, `
		`)
}

func mapElementToType(tdi *Type, i int, name string, f *Field) string {
	return valueToType(tdi, fmt.Sprintf(`"%s"`, name), f.Type)
}

func valueToType(tdi *Type, value string, e Expr) string {
	v := toGoExprInfo(tdi.GoFile, e)
	if v.Unsupported {
		return v.FullGoName
	}
	if !v.Exported {
		return fmt.Sprintf("ABEND049(codegen.go: no conversion to private type %s (%s))",
			v.FullGoName, toGoExprString(tdi.GoFile, v.UnderlyingType))
	}
	if v.ConvertFromMap != "" {
		return fmt.Sprintf(v.ConvertFromMap, "o", value)
	}
	return fmt.Sprintf("ABEND048(codegen.go: no conversion from Clojure for %s (%s))",
		v.FullGoName, toGoExprString(tdi.GoFile, v.UnderlyingType))
}

// Add the list of imports to those required if this type's constructor can be emitted (no ABENDs).
func addRequiredImports(tdi *Type, importeds []imports.Import) {
	ti := TypeDefsToGoTypes[tdi]
	for _, imp := range importeds {
		imports.AddImport(ti.RequiredImports, imp.Local, imp.Full, false)
	}
}
