package main

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/abends"
	"github.com/candid82/joker/tools/gostd/genutils"
	"github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/imports"
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
	conversion              string // empty if no conversion, else conversion expression with %s as expr to be converted
}

/* IMPORTANT: The public functions listed herein should be only those
   defined in joker/core/custom-runtime.go.

   That's how gostd knows to not actually generate calls to
   as-yet-unimplemented (or stubbed-out) functions, saving the
   developer the hassle of getting most of the way through a build
   before hitting undefined-func errors.
*/
var customRuntimeImplemented = map[string]struct{}{
	"ConvertToArrayOfbyte":   {},
	"ConvertToArrayOfint":    {},
	"ConvertToArrayOfstring": {},
}

func genGoCall(pkgBaseName, goFname, goParams string) string {
	return "{{myGoImport}}." + goFname + "(" + goParams + ")\n"
}

func genFuncCode(fn *FuncInfo, pkgBaseName, pkgDirUnix string, t *FuncType, goFname string) (fc funcCode) {
	var goPreCode, goParams, goResultAssign, goPostCode string

	fc.clojureParamList, fc.clojureParamListDoc, fc.clojureGoParams, fc.goParamList, fc.goParamListDoc, goPreCode, goParams, _, _ =
		genGoPre(fn, "\t", t.Params, goFname)
	goCall := genGoCall(pkgBaseName, fn.BaseName, goParams)
	goResultAssign, fc.clojureReturnType, fc.clojureReturnTypeForDoc, fc.goReturnTypeForDoc, goPostCode, fc.conversion = genGoPost(fn, "\t", t)

	if goPostCode == "" && goResultAssign == "" {
		goPostCode = "\treturn NIL\n"
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

	resultAssign, cljReturnType, cljReturnTypeForDoc, returnTypeForDoc, postCode, _ := genGoPost(fn, "\t", fn.Ft)
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
	genutils.GenSymReset()
	pkgDirUnix := fn.SourceFile.Package.Dir.String()

	const goTemplate = `
func %s(o GoObject, args Object) Object {  // %s
%s}
`

	goFname := genutils.FuncNameAsGoPrivate(fn.Name)

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
		goFn = strings.ReplaceAll(goFn, "{{myGoImport}}", path.Base(pkgDirUnix))
		abends.TrackAbends(clojureFn)
		abends.TrackAbends(goFn)
		if fn.Fd == nil {
			NumFunctions++
			godb.NumMethods++
		}
	} else {
		NumGeneratedFunctions++
		PackagesInfo[pkgDirUnix].NonEmpty = true
		im := PackagesInfo[pkgDirUnix].ImportsNative
		im.Promote(fn.ImportsNative, fn.Pos)
		im.InternPackage(godb.ClojureCoreDir, "", "", fn.Pos)
		myGoImport := im.AddPackage(pkgDirUnix, "", "", true, fn.Pos)
		goFn = strings.ReplaceAll(goFn, "{{myGoImport}}", myGoImport)
		if fn.Fd == nil {
			NumFunctions++
			godb.NumMethods++
			godb.NumGeneratedMethods++
			ti := fn.ToM
			if ti == nil {
				panic(fmt.Sprintf("Cannot find type for %s", fn.Name))
			}
			if _, ok := GoCode[pkgDirUnix].InitVars[ti]; !ok {
				GoCode[pkgDirUnix].InitVars[ti] = map[string]*FnCodeInfo{}
			}
			GoCode[pkgDirUnix].InitVars[ti][fn.BaseName] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFname, Params: fn.Ft.Params, FnDoc: fn.Doc}
		} else {
			NumGeneratedReceivers++
			for _, r := range fn.Fd.Recv.List {
				ti := TypeInfoForExpr(r.Type)
				if ti == nil {
					panic(fmt.Sprintf("nil ti for %v!!", r.Type))
				}
				if _, ok := GoCode[pkgDirUnix].InitVars[ti]; !ok {
					GoCode[pkgDirUnix].InitVars[ti] = map[string]*FnCodeInfo{}
				}
				GoCode[pkgDirUnix].InitVars[ti][fn.BaseName] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFname, FnDecl: fn.Fd, Params: fn.Fd.Type.Params, FnDoc: fn.Doc}
			}
		}
	}

	if goFn != "" {
		var params *FieldList
		if fn.Fd != nil {
			params = fn.Fd.Type.Params
		}
		GoCode[pkgDirUnix].Functions[goFname] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFn, FnDecl: fn.Fd, Params: params, FnDoc: nil}
		//		fmt.Printf("codegen.go/GenReceiver: Added %s to %s\n", goFname, pkgDirUnix)
	}
}

func GenStandalone(fn *FuncInfo) {
	genutils.GenSymReset()
	d := fn.Fd
	pkgDirUnix := fn.SourceFile.Package.Dir.String()
	pkgBaseName := fn.SourceFile.Package.BaseName

	const clojureTemplate = `
(defn %s%s
%s  {:added "1.0"
   :go "%s"}
  [%s])
`
	goFname := genutils.FuncNameAsGoPrivate(d.Name.Name)
	fc := genFuncCode(fn, pkgBaseName, pkgDirUnix, fn.Ft, goFname)
	clojureReturnType, goReturnType := genutils.ClojureReturnTypeForGenerateCustom(fc.clojureReturnType, fc.goReturnTypeForDoc)

	var cl2gol string
	if clojureReturnType == "" {
		cl2gol = goFname
		fc.conversion = ""
	} else {
		// No Go code needs to be generated when a return type is explicitly specified.
		clojureReturnType += " "
		cl2gol = pkgBaseName + "." + fn.BaseName
		if _, found := PackagesInfo[pkgDirUnix]; !found {
			panic(fmt.Sprintf("Cannot find package %s", pkgDirUnix))
		}
	}
	cl2golCall := cl2gol + fc.clojureGoParams
	if fc.conversion != "" {
		cl2golCall = fmt.Sprintf(fc.conversion, cl2golCall)
	}

	clojureFn := fmt.Sprintf(clojureTemplate, clojureReturnType, d.Name.Name,
		"  "+genutils.CommentGroupInQuotes(d.Doc, fc.clojureParamListDoc, fc.clojureReturnTypeForDoc,
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
		clojureFn = strings.ReplaceAll(clojureFn, "{{myGoImport}}", path.Base(pkgDirUnix))
		goFn = nonEmptyLineRegexp.ReplaceAllString(goFn, `// $1`)
		goFn = strings.ReplaceAll(goFn, "{{myGoImport}}", path.Base(pkgDirUnix))
		abends.TrackAbends(clojureFn)
		abends.TrackAbends(goFn)
	} else {
		NumGeneratedFunctions++
		NumGeneratedStandalones++
		pi := PackagesInfo[pkgDirUnix]
		pi.NonEmpty = true
		if clojureReturnType == "" {
			im := pi.ImportsNative
			im.InternPackage(godb.ClojureCoreDir, "", "", fn.Pos)
			myGoImport := im.AddPackage(pkgDirUnix, "", "", true, fn.Pos)
			goFn = strings.ReplaceAll(goFn, "{{myGoImport}}", myGoImport)
			im.Promote(fn.ImportsNative, fn.Pos)
		} else {
			// No Go code needs to be generated when a return type is explicitly specified.
			pi.ImportsAutoGen.AddPackage(pkgDirUnix, fn.SourceFile.Package.NsRoot, "", false, fn.Pos)
		}
		pi.ImportsAutoGen.Promote(fn.ImportsAutoGen, fn.Pos)
	}

	ClojureCode[pkgDirUnix].Functions[d.Name.Name] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: clojureFn, FnDecl: nil, FnDoc: nil}

	if goFn != "" {
		GoCode[pkgDirUnix].Functions[d.Name.Name] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFn, FnDecl: nil, FnDoc: nil}
	}
}

func GenConstant(ci *ConstantInfo) {
	genutils.GenSymReset()
	pkgDirUnix := ci.SourceFile.Package.Dir.String()

	PackagesInfo[pkgDirUnix].NonEmpty = true

	myGoImport := PackagesInfo[pkgDirUnix].ImportsAutoGen.AddPackage(pkgDirUnix, ci.SourceFile.Package.NsRoot, "", true, ci.Name.NamePos)
	ci.Def = strings.ReplaceAll(ci.Def, "{{myGoImport}}", myGoImport)

	ClojureCode[pkgDirUnix].Constants[ci.Name.Name] = ci
}

func GenVariable(vi *VariableInfo) {
	genutils.GenSymReset()
	pkgDirUnix := vi.SourceFile.Package.Dir.String()

	PackagesInfo[pkgDirUnix].NonEmpty = true

	myGoImport := PackagesInfo[pkgDirUnix].ImportsAutoGen.AddPackage(pkgDirUnix, vi.SourceFile.Package.NsRoot, "", true, vi.Name.NamePos)
	vi.Def = strings.ReplaceAll(vi.Def, "{{myGoImport}}", myGoImport)

	ClojureCode[pkgDirUnix].Variables[vi.Name.Name] = vi
}

func maybeImplicitConvert(src *godb.GoFile, typeName string, ti TypeInfo) string {
	ts := ti.TypeSpec()
	if ts == nil {
		return ""
	}

	t := TypeInfoForExpr(ts.Type)
	if t.Custom() {
		return ""
	}

	argType := t.ArgClojureType()
	declType := t.ArgExtractFunc()
	if argType == "" || declType == "" {
		return ""
	}
	const exTemplate = `case %s:
		v := %s(Extract%s(args, index))
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

func goTypeExtractor(t string, ti TypeInfo) string {
	ts := ti.UnderlyingTypeSpec()

	const template = `
func %s(rcvr, arg string, args *ArraySeq, n int) (res %s) {
	a := CheckGoNth(rcvr, "%s", arg, args, n).O
	res, ok := a.(%s)
	if !ok {
		panic(RT.NewError(%s.Sprintf("Argument %%d passed to %%s should be type GoObject[%s], but is GoObject[%%s]",
			n, rcvr, GoObjectTypeToString(a))))
	}
	return
}
`

	mangled := genutils.TypeToGoExtractFuncName(ti.ArgClojureArgType())
	localType := fmt.Sprintf(ti.GoPattern(), "{{myGoImport}}."+ti.GoBaseName())
	typeDoc := ti.ArgClojureArgType() // "path.filepath.Mode"

	fmtLocal := PackagesInfo[godb.GoPackageForTypeSpec(ts)].ImportsNative.AddPackage("fmt", "", "", true, ts.Pos())

	fnName := "ExtractGo_" + mangled
	resType := localType
	resTypeDoc := typeDoc // or similar
	resType += ""         // repeated here
	fmtLocal += ""        //
	resTypeDoc += ""      // repeated here

	return fmt.Sprintf(template, fnName, resType, resTypeDoc, resType, fmtLocal, resTypeDoc)
}

func GenType(t string, ti TypeInfo) {
	if ti.IsUnsupported() || !ti.IsExported() || ti.IsArbitraryType() {
		return
	}

	ts := ti.UnderlyingTypeSpec()
	if ts == nil {
		//		fmt.Printf("codegen.go/GenType(): skipping %s due to no underlying TypeSpec\n", ti.GoName())
		return
	}

	pkgDirUnix := godb.GoPackageForTypeSpec(ts)
	pi := PackagesInfo[pkgDirUnix]

	pi.NonEmpty = true
	where := ts.Pos()

	pi.ImportsNative.InternPackage(godb.ClojureCoreDir, "", "", where)
	myGoImport := pi.ImportsNative.AddPackage(pkgDirUnix, "", "", true, where)

	ClojureCode[pkgDirUnix].Types[t] = ti
	GoCode[pkgDirUnix].Types[t] = ti

	const goTemplate = `
func %s(args []Object, index int) %s%s {
	a := args[index]
	switch o := a.(type) {
	case GoObject:
		switch r := o.O.(type) {
%s%s		}
	%s}
	panic(RT.NewArgTypeError(index, a, "GoObject[%s]"))
}
`

	const goExtractTemplate = `
		case %s%s:
			return r
`

	const goExtractRefToTemplate = `
		case %s:
			return %sr  // refTo
`

	apiName := "ExtractGoObject" + fmt.Sprintf(ti.ClojurePattern(), ti.ClojureBaseName())
	typeName := fmt.Sprintf(ti.GoPattern(), myGoImport+"."+ti.GoBaseName())

	others := maybeImplicitConvert(godb.GoFileForTypeSpec(ts), typeName, ti)

	goExtract := ""
	goExtractRefTo := ""
	ptrTo := ""
	refTo := ""

	if ti.IsPassedByAddress() {
		if ti.IsAddressable() {
			ptrTo = "*"
			refTo = "&"
		}
		goExtract = fmt.Sprintf(goExtractTemplate[1:], ptrTo, typeName)
	}
	if ti.IsAddressable() {
		goExtractRefTo = fmt.Sprintf(goExtractRefToTemplate[1:], typeName, refTo)
	}
	if goExtract == "" && goExtractRefTo == "" {
		return // E.g. reflect_native.go's refToStringHeader
	}

	goc := fmt.Sprintf(goTemplate, apiName, ptrTo, typeName, goExtract, goExtractRefTo, others, t)

	goc += goTypeExtractor(t, ti)

	goc = strings.ReplaceAll(goc, "{{myGoImport}}", myGoImport)

	GoCodeForType[ti] = goc
	ClojureCodeForType[ti] = ""

	apiFullName := pi.ClojureNameSpace + "/" + apiName
	coreApis[apiFullName] = struct{}{}
	//	fmt.Printf("codegen.go/GenType(): added API '%s'\n", apiFullName)
}

var Ctors = map[TypeInfo]string{}
var CtorNames = map[TypeInfo]string{}

func genCtor(tyi TypeInfo) {
	if !tyi.Custom() || !tyi.IsAddressable() {
		return
	}

	ts := tyi.TypeSpec()
	if ts == nil {
		return // TODO: Support *type, []type, etc
	}

	const goConstructTemplate = `
%sfunc %s(_v Object) %s%s {
	switch _o := _v.(type) {
	%s
	}
	panic(RT.NewArgTypeError(0, _v, "%s"))
}

func %s(_o Object) Object {
	return MakeGoObject(%s(_o))
}
`

	typeName := fmt.Sprintf(tyi.GoPattern(), "{{myGoImport}}."+tyi.GoBaseName())
	localTypeName := fmt.Sprintf(tyi.GoPattern(), tyi.GoBaseName())
	ctorApiName := "_Ctor_" + fmt.Sprintf(tyi.ClojurePattern(), tyi.ClojureBaseName())
	wrappedCtorApiName := "_Wrapped" + ctorApiName

	nonGoObject, expectedObjectDoc, helperFunc, ptrTo := nonGoObjectCase(tyi, typeName, localTypeName)

	goConstructor := fmt.Sprintf(goConstructTemplate, helperFunc, ctorApiName, ptrTo, typeName, nonGoObject, expectedObjectDoc,
		wrappedCtorApiName, ctorApiName)

	pkgDirUnix := godb.GoPackageForTypeSpec(ts)
	if strings.Contains(goConstructor, "ABEND") {
		goConstructor = nonEmptyLineRegexp.ReplaceAllString(goConstructor, `// $1`)
		goConstructor = strings.ReplaceAll(goConstructor, "{{myGoImport}}", path.Base(pkgDirUnix))
		abends.TrackAbends(goConstructor)
	} else {
		pi := PackagesInfo[pkgDirUnix]
		pi.ImportsNative.Promote(tyi.RequiredImports(), tyi.DefPos())
		myGoImport := pi.ImportsNative.AddPackage(pkgDirUnix, "", "", true, tyi.DefPos())
		goConstructor = strings.ReplaceAll(goConstructor, "{{myGoImport}}", myGoImport)
		CtorNames[tyi] = wrappedCtorApiName
		NumGeneratedCtors++
	}

	Ctors[tyi] = goConstructor

	//	fmt.Printf("codegen.go/genCtor: %s %+v\n", tyi, tyi.ClojureTypeInfo())
}

func appendMethods(ti TypeInfo, iface *InterfaceType) {
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
				typeFullName := ti.GoName()
				fullName := typeFullName + "_" + n.Name
				typeBaseName := ti.GoBaseName()
				baseName := typeBaseName + "_" + n.Name
				doc := m.Doc
				if doc == nil {
					doc = m.Comment
				}
				QualifiedFunctions[fullName] = &FuncInfo{
					BaseName:       n.Name,
					ReceiverId:     "{{myGoImport}}." + typeBaseName,
					Name:           baseName,
					DocName:        "(" + ti.GoFile().Package.Dir.String() + "." + typeBaseName + ")" + n.Name + "()",
					Fd:             nil,
					ToM:            ti,
					Ft:             m.Type.(*FuncType),
					Doc:            doc,
					SourceFile:     ti.GoFile(),
					ImportsNative:  &imports.Imports{},
					ImportsAutoGen: &imports.Imports{},
					Pos:            n.NamePos,
				}
			}
			continue
		}
		ts := godb.Resolve(m.Type)
		if ts == nil {
			return
		}
		appendMethods(ti, ts.(*TypeSpec).Type.(*InterfaceType))
	}
}

func GenTypeInfo() {
	allTypes := AllTypesSorted()

	for _, ti := range allTypes {
		GenTypeFromDb(ti)
	}

	var types []TypeInfo
	ord := (uint)(0)

	for _, ti := range allTypesSorted {
		more := false
		if ti.GoName() == "[][]*crypto/x509.Certificate XXX DISABLED XXX" {
			fmt.Printf("codegen.go/GenTypeInfo(): %s == %+v %+v\n", ti.ClojureName(), ti.GoTypeInfo(), ti.ClojureTypeInfo())
			more = true
		}
		if !ti.Custom() {
			if uti := ti.UnderlyingTypeInfo(); uti == nil || !uti.Custom() {
				if more {
					fmt.Printf("codegen.go/GenTypeInfo(): no underlying type @%p or a builtin type: %s == @%p %+v @%p %+v @%p %+v\n", uti, ti.ClojureName(), ti, ti, ti.GoTypeInfo(), ti.GoTypeInfo(), ti.ClojureTypeInfo(), ti.ClojureTypeInfo())
				}
				continue
			}
		}
		if !ti.IsSwitchable() {
			if more {
				fmt.Printf("codegen.go/GenTypeInfo(): %s not switchable\n", ti.GoName())
			}
			continue
		}
		types = append(types, ti)
		Ordinal[ti] = ord
		if more {
			fmt.Printf("codegen.go/GenTypeInfo(): assigned ordinal %3d to %s (specificity=%d)\n", ord, ti.GoName(), ti.Specificity())
		}
		ord++
	}

	SwitchableTypes = types
}

func GenTypeFromDb(ti TypeInfo) {
	if ti.ClojureName() == "crypto/Hash" || true {
		//		fmt.Printf("codegen.go/GenTypeFromDb: %s == @%p %+v @%p %+v @%p %+v\n", ti.ClojureName(), ti, ti, ti.ClojureTypeInfo(), ti.ClojureTypeInfo(), ti.GoTypeInfo(), ti.GoTypeInfo())
	}

	if !ti.IsExported() || ti.IsArbitraryType() {
		//		fmt.Printf("codegen.go/GenTypeFromDb: not exported or an array type\n")
		return // Do not generate anything for private or array types
	}
	//	fmt.Printf("codegen.go/GenTypeFromDb: not a concrete type\n")

	ts := ti.TypeSpec()
	if ts == nil {
		if uti := ti.UnderlyingTypeInfo(); uti != nil {
			ts = uti.TypeSpec()
		} else {
			//			fmt.Printf("codegen.go/GenTypeFromDb: %s has no underlying type!\n", ti.ClojureName())
			return
		}
	}

	if ti.Specificity() == ConcreteType {
		genCtor(ti)
		return // The code below currently handles only interface{} types
	}

	if ts != nil {
		if ts.Type != nil {
			appendMethods(ti, ts.Type.(*InterfaceType))
		}
	}
}

func nonGoObjectCase(ti TypeInfo, typeName, baseTypeName string) (nonGoObjectCase, nonGoObjectCaseDoc, helperFunc, ptrTo string) {
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

func nonGoObjectTypeFor(ti TypeInfo, typeName, baseTypeName string) (nonGoObjectTypes, nonGoObjectTypeDocs, extractClojureObjects, helperFuncs []string, ptrTo string) {
	ts := ti.UnderlyingTypeSpec()
	if ts == nil {
		panic(fmt.Sprintf("nil ts for ti=%+v gti=%+v jti=%+v", ti, ti.GoTypeInfo(), ti.ClojureTypeInfo()))
	}
	if ts.Type == nil {
		panic(fmt.Sprintf("nil ts.Type for ts=%T %+v", ts, *ts))
	}
	switch t := ts.Type.(type) {
	case *Ident:
		nonGoObjectType, nonGoObjectTypeDoc, extractClojureObject := simpleTypeFor(ti.GoFile().Package.Dir.String(), t.Name, ts.Type)
		extractClojureObject = typeName + "(_o" + extractClojureObject + ")"
		nonGoObjectTypes = []string{nonGoObjectType}
		nonGoObjectTypeDocs = []string{nonGoObjectTypeDoc}
		extractClojureObjects = []string{extractClojureObject}
		if nonGoObjectType != "" {
			return
		}
	case *StructType:
		uniqueTypeName := typeName
		mapHelperFName := "_mapTo" + baseTypeName
		return []string{"case *ArrayMap, *HashMap"},
			[]string{"Map"},
			[]string{mapHelperFName + "(_o.(Map))"},
			[]string{mapToType(ti, mapHelperFName, uniqueTypeName, t)},
			"*"
	case *ArrayType:
	}
	return []string{"default"},
		[]string{"whatever"},
		[]string{fmt.Sprintf("_%s(_o.ABEND674(codegen.go: unknown underlying type %T for %s))",
			typeName, ts.Type, baseTypeName)},
		[]string{""},
		""
}

func simpleTypeFor(pkgDirUnix, name string, e Expr) (nonGoObjectType, nonGoObjectTypeDoc, extractClojureObject string) {
	v := TypeInfoForGoName(genutils.CombineGoName(pkgDirUnix, name))
	nonGoObjectType = "case " + v.ArgClojureType()
	nonGoObjectTypeDoc = v.ArgClojureType()
	extractClojureObject = v.ArgFromClojureObject()
	if v.IsUnsupported() || v.ArgClojureType() == "" || extractClojureObject == "" {
		nonGoObjectType += fmt.Sprintf(" /* ABEND171(`%s': IsUnsupported=%v ArgClojureType=%v ArgFromClojureObject=%v) */", v.GoName(), v.IsUnsupported(), v.ArgClojureType(), extractClojureObject)
	}
	return
}

func mapToType(ti TypeInfo, helperFName, typeName string, ty *StructType) string {
	const hFunc = `func %s(o Map) *%s {
	return &%s{%s}
}

`
	valToType := elementsToType(ti, ty, mapElementToType)
	if valToType != "" {
		valToType = `
		` + valToType + `
	`
	}

	return fmt.Sprintf(hFunc, helperFName, typeName, typeName, valToType)
}

func elementsToType(ti TypeInfo, ty *StructType, toType func(ti TypeInfo, i int, name string, f *Field) string) string {
	els := []string{}
	i := 0
	for _, f := range ty.Fields.List {
		for _, p := range f.Names {
			fieldName := p.Name
			if fieldName == "" || !IsExported(fieldName) {
				continue
			}
			els = append(els, fmt.Sprintf("%s: %s,", fieldName, toType(ti, i, p.Name, f)))
			i++
		}
	}
	return strings.Join(els, `
		`)
}

func mapElementToType(ti TypeInfo, i int, name string, f *Field) string {
	return valueToType(ti, fmt.Sprintf(`"%s"`, name), f.Type)
}

func valueToType(ti TypeInfo, value string, e Expr) string {
	v := TypeInfoForExpr(e)
	if v.IsUnsupported() {
		return v.GoName()
	}
	var uty Expr
	if v.TypeSpec() != nil {
		uty = v.TypeSpec().Type
	} else if v.UnderlyingType() != nil {
		uty = v.UnderlyingType()
	}
	if !v.IsExported() {
		return fmt.Sprintf("ABEND049(codegen.go: no conversion to private type %s (%s))",
			v.GoName(), StringForExpr(uty))
	}
	if v.ConvertFromMap() != "" {
		return fmt.Sprintf(v.ConvertFromMap(), "o", value)
	}
	return fmt.Sprintf("ABEND048(codegen.go: no conversion from Clojure for %s (%s))",
		v.GoName(), StringForExpr(uty))
}

// Add the list of imports to those required if this type's constructor can be emitted (no ABENDs).
func addRequiredImports(ti TypeInfo, importeds []imports.Import) {
	to := ti.RequiredImports()
	for _, imp := range importeds {
		to.AddPackage(imp.Full, imp.ClojurePrefix, imp.PathPrefix, false, imp.Pos)
	}
}
