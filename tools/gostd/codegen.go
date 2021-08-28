package main

import (
	"bytes"
	"fmt"
	"github.com/candid82/joker/tools/gostd/abends"
	"github.com/candid82/joker/tools/gostd/genutils"
	"github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/imports"
	. "go/ast"
	"os"
	"path"
	"regexp"
	"strconv"
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

func genGoCall(pkgBaseName, goFname, goParams string) string {
	return "{{myGoImport}}." + goFname + "(" + goParams + ")\n"
}

func genFuncCode(fn *FuncInfo, pkgBaseName string, t *FuncType, goFname string) (fc funcCode) {
	var goPreCode, goParams, goResultAssign, goPostCode string

	fc.clojureParamList, fc.clojureParamListDoc, fc.clojureGoParams, fc.goParamList, fc.goParamListDoc, goPreCode, goParams = genGoPreFunc(fn)
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
	preCode, params, min, max := genGoPreReceiver(fn)

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
		postCode = "\treturn NIL\n"
	}
	maybeAssignArgList := ""
	if max > 0 {
		maybeAssignArgList = "_argList := "
	}
	ai := map[string]interface{}{
		"ArgList": maybeAssignArgList,
		"DocName": strconv.Quote(fn.DocName),
		"Min":     min,
		"Max":     max,
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "go-receiver-arity.tmpl", ai)
	arity := buf.String()

	return arity + preCode + "\t" + resultAssign + call + "\n" + postCode
}

func GenReceiver(fn *FuncInfo) {
	genutils.GenSymReset()
	pkgDirUnix := fn.SourceFile.Package.Dir.String()
	ns := fn.Namespace

	goFname := genutils.FuncNameAsGoPrivate(fn.Name)

	if !IsExported(fn.BaseName) {
		return
	}

	receiverFuncInfo := map[string]string{
		"FuncName": goFname,
		"What": func() string {
			if fn.Fd == nil {
				return "Method"
			} else {
				return "Receiver"
			}
		}(),
		"Code": genReceiverCode(fn, goFname),
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "go-receiver-func.tmpl", receiverFuncInfo)
	goFn := buf.String()

	clojureFn := ""
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
		if _, found := NamespacesInfo[ns]; !found {
			panic(fmt.Sprintf("no ns=%s", ns))
		}
		NamespacesInfo[ns].NonEmpty = true
		im := NamespacesInfo[ns].ImportsNative
		im.Promote(fn.ImportsNative, fn.Pos)
		im.InternPackage(godb.ClojureCoreDir, "", "", fn.Pos)
		myGoImport := im.AddPackage(pkgDirUnix, "", "", "", true, fn.Pos)
		goFn = strings.ReplaceAll(goFn, "{{myGoImport}}", myGoImport)
		if fn.Fd == nil {
			NumFunctions++
			godb.NumMethods++
			godb.NumGeneratedMethods++
			ti := fn.ToM
			if ti == nil {
				panic(fmt.Sprintf("Cannot find type for %s", fn.Name))
			}
			if _, ok := GoCode[ns].InitVars[ti]; !ok {
				GoCode[ns].InitVars[ti] = map[string]*FnCodeInfo{}
			}
			GoCode[ns].InitVars[ti][fn.BaseName] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFname, Params: fn.Ft.Params, FnDoc: fn.Doc}
		} else {
			NumGeneratedReceivers++
			for _, r := range fn.Fd.Recv.List {
				ti := TypeInfoForExpr(r.Type)
				if ti == nil {
					panic(fmt.Sprintf("nil ti for %v!!", r.Type))
				}
				if _, ok := GoCode[ns].InitVars[ti]; !ok {
					GoCode[ns].InitVars[ti] = map[string]*FnCodeInfo{}
				}
				GoCode[ns].InitVars[ti][fn.BaseName] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFname, FnDecl: fn.Fd, Params: fn.Fd.Type.Params, FnDoc: fn.Doc}
			}
		}
	}

	if goFn != "" {
		var params *FieldList
		if fn.Fd != nil {
			params = fn.Fd.Type.Params
		}
		GoCode[ns].Functions[goFname] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFn, FnDecl: fn.Fd, Params: params, FnDoc: nil}
		//		fmt.Printf("codegen.go/GenReceiver: Added %s to %s\n", goFname, ns)
	}
}

func GenStandalone(fn *FuncInfo) {
	genutils.GenSymReset()
	d := fn.Fd
	pkgDirUnix := fn.SourceFile.Package.Dir.String()
	pkgBaseName := fn.SourceFile.Package.BaseName
	ns := fn.Namespace

	goFname := genutils.FuncNameAsGoPrivate(d.Name.Name)
	fc := genFuncCode(fn, pkgBaseName, fn.Ft, goFname)
	clojureReturnType, goReturnType := genutils.ClojureReturnTypeForGenerateCustom(fc.clojureReturnType, fc.goReturnTypeForDoc)

	var cl2gol string
	if clojureReturnType == "" {
		cl2gol = goFname
		fc.conversion = ""
	} else {
		// No Go code needs to be generated when a return type is explicitly specified.
		clojureReturnType += " "
		cl2gol = pkgBaseName + "." + fn.BaseName
		if _, found := NamespacesInfo[ns]; !found {
			panic(fmt.Sprintf("Cannot find namespace %s", ns))
		}
	}
	cl2golCall := cl2gol + fc.clojureGoParams
	if fc.conversion != "" {
		cl2golCall = fmt.Sprintf(fc.conversion, cl2golCall)
	}

	clojureDefnInfo := map[string]string{
		"ReturnType": clojureReturnType,
		"Name":       d.Name.Name,
		"DocString": genutils.CommentGroupInQuotes(d.Doc, fc.clojureParamListDoc, fc.clojureReturnTypeForDoc,
			fc.goParamListDoc, fc.goReturnTypeForDoc) + "\n",
		"GoCode":    cl2golCall,
		"ParamList": fc.clojureParamList,
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "clojure-defn.tmpl", clojureDefnInfo)
	clojureFn := buf.String()

	goFuncInfo := map[string]string{
		"Name":       goFname,
		"ParamList":  fc.goParamList,
		"ReturnType": goReturnType,
		"Code":       fc.goCode,
	}

	buf.Reset()
	Templates.ExecuteTemplate(buf, "go-func-info.tmpl", goFuncInfo)
	goFn := buf.String()

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
		nsi := NamespacesInfo[ns]
		nsi.NonEmpty = true
		if clojureReturnType == "" {
			im := nsi.ImportsNative
			im.InternPackage(godb.ClojureCoreDir, "", "", fn.Pos)
			myGoImport := im.AddPackage(pkgDirUnix, "", "", "", true, fn.Pos)
			goFn = strings.ReplaceAll(goFn, "{{myGoImport}}", myGoImport)
			im.Promote(fn.ImportsNative, fn.Pos)
		} else {
			// No Go code needs to be generated when a return type is explicitly specified.
			nsi.ImportsAutoGen.AddPackage(pkgDirUnix, fn.SourceFile.Package.NsRoot, "", "", false, fn.Pos)
		}
		nsi.ImportsAutoGen.Promote(fn.ImportsAutoGen, fn.Pos)
	}

	ClojureCode[ns].Functions[d.Name.Name] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: clojureFn, FnDecl: nil, FnDoc: nil}

	if goFn != "" {
		GoCode[ns].Functions[d.Name.Name] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFn, FnDecl: nil, FnDoc: nil}
	}
}

func GenConstant(ci *ConstantInfo) {
	genutils.GenSymReset()
	ns := ci.Namespace

	NamespacesInfo[ns].NonEmpty = true

	defer func() {
		if x := recover(); x != nil {
			fmt.Fprintf(os.Stderr, "Panic \"%s\" for ns=%s name=%s\n", x, ns, ci.Name.Name)
		}
	}()
	ClojureCode[ns].Constants[ci.Name.Name] = ci
}

func GenVariable(vi *VariableInfo) {
	genutils.GenSymReset()
	pkgDirUnix := vi.SourceFile.Package.Dir.String()
	ns := vi.Namespace

	NamespacesInfo[ns].NonEmpty = true

	myGoImport := NamespacesInfo[ns].ImportsAutoGen.AddPackage(pkgDirUnix, vi.SourceFile.Package.NsRoot, "", "", true, vi.Name.NamePos)
	vi.Def = strings.ReplaceAll(vi.Def, "{{myGoImport}}", myGoImport)

	ClojureCode[ns].Variables[vi.Name.Name] = vi
}

func maybeImplicitConvert(src *godb.GoFile, typeName string, ti TypeInfo) string {
	ts := ti.TypeSpec()
	if ts == nil {
		return ""
	}

	t := TypeInfoForExpr(ts.Type)
	if t.IsCustom() {
		return ""
	}

	argType := t.ArgClojureType()
	declType := t.GoApiString(false)
	if argType == "" || declType == "" {
		return ""
	}

	coerceApi := "ObjectAs_" + declType
	if _, ok := definedApis[coerceApi]; !ok {
		return "" // Not implemented
	}

	addressOf := ""
	if t.IsPassedByAddress() {
		addressOf = "&"
	}

	mic := map[string]string{
		"ArgType":   argType,
		"TypeName":  typeName,
		"CoerceApi": coerceApi,
		"AddressOf": addressOf,
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "go-implicit-convert.tmpl", mic)

	return buf.String()
}

func GenType(t string, ti TypeInfo) {
	//	fmt.Printf("codegen.go/GenType(): %s\n", ti.GoName())
	if ti.IsUnsupported() || !ti.IsExported() || ti.IsArbitraryType() {
		return
	}
	//	fmt.Printf("codegen.go/GenType(): %s GOOD SO FAR\n", ti.GoName())

	ts := ti.UnderlyingTypeSpec()
	if ts == nil {
		return
	}
	//	fmt.Printf("codegen.go/GenType(): %s BETTER\n", ti.GoName())

	pkgDirUnix := godb.GoPackageForTypeSpec(ts)
	pkgInfo := PackagesInfo[pkgDirUnix]
	ns := pkgInfo.Namespace
	nsi := NamespacesInfo[ns]

	nsi.NonEmpty = true
	where := ts.Pos()

	nsi.ImportsNative.InternPackage(godb.ClojureCoreDir, "", "", where)
	myGoImport := nsi.ImportsNative.AddPackage(pkgDirUnix, "", "", "", true, where)

	ClojureCode[ns].Types[t] = ti
	GoCode[ns].Types[t] = ti

	const goExtractTemplate = `
		case %s%s:
			return r, true
`

	const goExtractRefToTemplate = `
		case %s:
			return %sr, true
`

	typeName := fmt.Sprintf(ti.GoPattern(), myGoImport+"."+ti.GoBaseName())
	apiSuffix := "_ns_" + fmt.Sprintf(ti.ClojurePattern(), ti.ClojureBaseName())
	MaybeIsApiName := "MaybeIs" + apiSuffix
	ExtractApiName := "Extract" + apiSuffix
	FieldAsApiName := "FieldAs" + apiSuffix
	ReceiverArgAsApiName := "ReceiverArgAs" + apiSuffix

	info := map[string]string{}

	info["MaybeIsApiName"] = MaybeIsApiName
	info["ExtractApiName"] = ExtractApiName
	info["FieldAsApiName"] = FieldAsApiName
	info["ReceiverArgAsApiName"] = ReceiverArgAsApiName
	info["TypeName"] = typeName
	info["TypeAsString"] = t

	info["Others"] = maybeImplicitConvert(godb.GoFileForTypeSpec(ts), typeName, ti)

	coerce := ""
	coerceRefTo := ""
	ptrTo := ""
	refTo := ""
	nilForType := fmt.Sprintf(ti.NilPattern(), typeName)
	if ti.IsPassedByAddress() {
		ptrTo = "*"
		refTo = "&"
		nilForType = "nil"
		if ti.IsAddressable() {
			coerceRefTo = fmt.Sprintf(goExtractRefToTemplate[1:], typeName, refTo)
		}
	}

	coerce = fmt.Sprintf(goExtractTemplate[1:], ptrTo, typeName)

	//	fmt.Printf("codegen.go/GenType(): %s DONE!\n", ti.GoName())
	info["Coerce"] = coerce
	info["CoerceRefTo"] = coerceRefTo
	info["PtrTo"] = ptrTo
	info["NilForType"] = nilForType

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "go-coerce.tmpl", info)

	GoCodeForType[ti] = strings.ReplaceAll(buf.String(), "{{myGoImport}}", myGoImport)
	ClojureCodeForType[ti] = ""

	NewDefinedApi(ns+"/"+MaybeIsApiName, "codegen.go/GenType()")
	NewDefinedApi(ns+"/"+ExtractApiName, "codegen.go/GenType()")
	NewDefinedApi(ns+"/"+FieldAsApiName, "codegen.go/GenType()")
	NewDefinedApi(ns+"/"+ReceiverArgAsApiName, "codegen.go/GenType()")
}

var Ctors = map[TypeInfo]string{}
var CtorNames = map[TypeInfo]string{}

func genCtor(tyi TypeInfo) {
	if !tyi.IsCtorable() {
		//		fmt.Fprintf(os.Stderr, "codegen.go/genCtor: Not ctorable: %q\n", tyi.GoName())
		return
	}

	refTo := ""
	uti := tyi
	ts := tyi.TypeSpec()
	if ts == nil {
		uti = tyi.UnderlyingTypeInfo()
		ts = uti.TypeSpec()
		if ts == nil {
			return
		}
	}

	goTypeName := fmt.Sprintf(tyi.GoPattern(), "{{myGoImport}}."+tyi.GoBaseName())
	clojureTypeName := fmt.Sprintf(tyi.ClojurePattern(), tyi.ClojureBaseName())
	ctorApiName := "_Ctor_" + clojureTypeName
	wrappedCtorApiName := "_Wrapped" + ctorApiName

	possibleObject, expectedObjectDoc, helperFunc := nonGoObjectCase(tyi, goTypeName, clojureTypeName)

	goCtorInfo := map[string]string{
		"HelperFunc":      helperFunc,
		"CtorName":        ctorApiName,
		"WrappedCtorName": wrappedCtorApiName,
		"RefTo":           refTo,
		"GoTypeName":      goTypeName,
		"Cases":           possibleObject,
		"Expected":        expectedObjectDoc,
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "go-ctor.tmpl", goCtorInfo)
	goConstructor := buf.String()

	pkgDirUnix := godb.GoPackageForTypeSpec(ts)
	ns := tyi.ClojureNamespace()
	if strings.Contains(goConstructor, "ABEND") {
		goConstructor = nonEmptyLineRegexp.ReplaceAllString(goConstructor, `// $1`)
		goConstructor = strings.ReplaceAll(goConstructor, "{{myGoImport}}", path.Base(pkgDirUnix))
		abends.TrackAbends(goConstructor)
	} else {
		nsi := NamespacesInfo[ns]
		nsi.ImportsNative.Promote(tyi.RequiredImports(), tyi.DefPos())
		myGoImport := nsi.ImportsNative.AddPackage(pkgDirUnix, "", "", "", true, tyi.DefPos())
		goConstructor = strings.ReplaceAll(goConstructor, "{{myGoImport}}", myGoImport)
		CtorNames[tyi] = wrappedCtorApiName
		if tyi != uti {
			CtorNames[uti] = wrappedCtorApiName
		}
		NumGeneratedCtors++
	}

	Ctors[tyi] = goConstructor

	//	fmt.Printf("codegen.go/genCtor: %s\n%s\n", tyi.GoName(), goConstructor)
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
					Namespace:      ti.ClojureNamespace(),
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
		genTypeFromDb(ti)
	}

	var types []TypeInfo
	ord := (uint)(0)

	for _, ti := range allTypesSorted {
		more := false
		if ti.GoName() == "[][]*crypto/x509.Certificate XXX DISABLED XXX" {
			fmt.Printf("codegen.go/GenTypeInfo(): %s == %+v %+v\n", ti.ClojureName(), ti.GoTypeInfo(), ti.ClojureTypeInfo())
			more = true
		}
		if !ti.IsCustom() {
			if uti := ti.UnderlyingTypeInfo(); uti == nil || !uti.IsCustom() {
				if more {
					fmt.Printf("codegen.go/GenTypeInfo(): no underlying type @%p: %s == @%p %+v @%p %+v @%p %+v\n", uti, ti.ClojureName(), ti, ti, ti.GoTypeInfo(), ti.GoTypeInfo(), ti.ClojureTypeInfo(), ti.ClojureTypeInfo())
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

func genTypeFromDb(ti TypeInfo) {
	if ti.ClojureName() == "crypto/Hash" || true {
		//		fmt.Printf("codegen.go/GenTypeFromDb: %s == @%p %+v @%p %+v @%p %+v\n", ti.ClojureName(), ti, ti, ti.ClojureTypeInfo(), ti.ClojureTypeInfo(), ti.GoTypeInfo(), ti.GoTypeInfo())
	}

	if !ti.IsExported() || ti.IsArbitraryType() {
		//		fmt.Printf("codegen.go/GenTypeFromDb: not exported or a special type\n")
		return // Do not generate anything for private or special types
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
		return // The code below currently handles only interface{} types
	}

	if ts != nil {
		if ts.Type != nil {
			if it, ok := ts.Type.(*InterfaceType); ok {
				appendMethods(ti, it)
			} else {
				panic(fmt.Sprintf("ts.Type for %q is %T, not *InterfaceType", ti.GoName(), ts.Type))
			}
		}
	}
}

func GenTypeCtors() {
	allTypes := AllTypesSorted()

	for _, ti := range allTypes {
		if ti.IsCtorable() {
			genCtor(ti)
		}
	}

}

func nonGoObjectCase(ti TypeInfo, goTypeName, clojureTypeName string) (nonGoObjectCase, nonGoObjectCaseDoc, helperFunc string) {
	nonGoObjectTypes, nonGoObjectTypeDocs, extractClojureObjects, helperFuncs := nonGoObjectTypeFor(ti, goTypeName, clojureTypeName)

	nonGoObjectCasePrefix := ""
	nonGoObjectCase = ""
	buf := new(bytes.Buffer)
	for i := 0; i < len(nonGoObjectTypes); i++ {
		possibleObjectCaseInfo := map[string]string{
			"Type":   nonGoObjectTypes[i],
			"Return": extractClojureObjects[i],
		}

		Templates.ExecuteTemplate(buf, "go-possible-case.tmpl", possibleObjectCaseInfo)
		nonGoObjectCase += nonGoObjectCasePrefix + buf.String()

		nonGoObjectCasePrefix = `
	`
	}

	return nonGoObjectCase,
		fmt.Sprintf("GoObject[%s] or: %s", goTypeName, strings.Join(nonGoObjectTypeDocs, " or ")),
		strings.Join(helperFuncs, "")
}

func nonGoObjectTypeFor(ti TypeInfo, goTypeName, clojureTypeName string) (nonGoObjectTypes, nonGoObjectTypeDocs, extractClojureObjects, helperFuncs []string) {
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
		extractClojureObject = goTypeName + "(_o" + extractClojureObject + ")"
		nonGoObjectTypes = []string{nonGoObjectType}
		nonGoObjectTypeDocs = []string{nonGoObjectTypeDoc}
		extractClojureObjects = []string{extractClojureObject}
		if nonGoObjectType != "" {
			return
		}
	case *StructType:
		mapHelperFName := "_mapTo_" + clojureTypeName
		return []string{"case *ArrayMap, *HashMap"},
			[]string{"Map"},
			[]string{mapHelperFName + "(_o.(Map))"},
			[]string{mapToType(ti, mapHelperFName, goTypeName, t)}
	case *ArrayType:
	}
	return []string{"default"},
		[]string{"whatever"},
		[]string{fmt.Sprintf("%s(_o.ABEND674(codegen.go: unknown underlying type %T for %s))",
			goTypeName, ts.Type, clojureTypeName)},
		[]string{""}
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

func mapToType(ti TypeInfo, helperFName, goTypeName string, ty *StructType) string {
	const hFunc = `func %s(o Map) %s {
	return %s{%s}
}

`
	goTypeCtorName := goTypeName
	if goTypeName[0] == '*' {
		goTypeCtorName = "&" + goTypeName[1:]
	}

	valToType := elementsToType(ti, ty, mapElementToType)
	if valToType != "" {
		valToType = `
		` + valToType + `
	`
	}

	return fmt.Sprintf(hFunc, helperFName, goTypeName, goTypeCtorName, valToType)
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
	clType := v.GoApiString(false)
	apiImportName := addApiToImports(ti, clType) // apiImportName := AddApiToImports(clType)
	api := determineRuntime("FieldAs_", "FieldAs_ns_", apiImportName, clType)

	deref := ""
	if v.IsPassedByAddress() {
		deref = "*"
	}

	return fmt.Sprintf("%s%s(o, %s)", deref, api, value)
}

func addApiToImports(ti TypeInfo, clType string) string {
	ix := strings.Index(clType, "/")
	if ix < 0 {
		return "" // builtin type (api is in core)
	}

	apiPkgPath := godb.ClojureSourceDir.Join(importStdRoot.String(), strings.ReplaceAll(clType[0:ix], ".", "/")).String()
	clojureStdPath := godb.ClojureSourceDir.Join(importStdRoot.String()).String()
	//	fmt.Fprintf(os.Stderr, "codegen.go/addApiToImports: Compared %s to %s\n", apiPkgPath, fn.GoFile().Package.ImportMe)
	if apiPkgPath == ti.GoFile().Package.ImportMe {
		return "" // api is local to function
	}

	clojureStdNs := ti.GoFile().Package.NsRoot
	native := ti.RequiredImports().AddPackage(apiPkgPath, clojureStdNs, clojureStdPath, "_gostd", true, ti.DefPos())

	return native
}

// Add the list of imports to those required if this type's constructor can be emitted (no ABENDs).
func addRequiredImports(ti TypeInfo, importeds []imports.Import) {
	to := ti.RequiredImports()
	for _, imp := range importeds {
		to.AddPackage(imp.Full, imp.ClojurePrefix, imp.PathPrefix, "", false, imp.Pos)
	}
}

func GenBuiltins() {
	ns := NamespacesInfo["go.std.builtin"]
	fmt.Printf("go.std.builtin: %+v\n", ns)
}

func init() {
	nonEmptyLineRegexp = regexp.MustCompile(`(?m)^(.)`)
}
