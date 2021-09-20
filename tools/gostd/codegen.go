package main

import (
	"bytes"
	"fmt"
	"github.com/candid82/joker/tools/gostd/abends"
	"github.com/candid82/joker/tools/gostd/astutils"
	"github.com/candid82/joker/tools/gostd/genutils"
	"github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/imports"
	. "go/ast"
	"go/token"
	"go/types"
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

func genGoCall(goFname, goParams string) string {
	return "{{myGoImport}}." + goFname + "(" + goParams + ")\n"
}

func genFuncCode(fn *FuncInfo, t *types.Signature) (fc funcCode) {
	var goParams, goResultAssign, goPostCode string

	fc.clojureParamList, fc.clojureParamListDoc, fc.clojureGoParams, fc.goParamList, fc.goParamListDoc, goParams = genGoPreFunc(fn)
	goCall := genGoCall(fn.BaseName, goParams)
	goResultAssign, fc.clojureReturnType, fc.clojureReturnTypeForDoc, fc.goReturnTypeForDoc, goPostCode, fc.conversion = genGoPost("\t", t)

	if goPostCode == "" && goResultAssign == "" {
		goPostCode = "\treturn NIL\n"
	}

	fc.goCode = "\t" + goResultAssign + goCall + // [results := ]fn-to-call([args...])
		goPostCode // Optional block of post-code
	return
}

func genReceiverCode(fn *FuncInfo, goFname string) string {
	defer func() {
		if x := recover(); x != nil {
			panic(fmt.Sprintf("panic generating code for %s at %s: %s\n", goFname, godb.WhereAt(fn.Pos), x))
		}
	}()

	preCode, params, min, max := genGoPreReceiver(fn)

	receiverName := fn.BaseName
	call := fmt.Sprintf("o.O.(%s).%s(%s)", fn.ReceiverId, receiverName, params)

	resultAssign, cljReturnType, cljReturnTypeForDoc, returnTypeForDoc, postCode, _ := genGoPost("\t", fn.Signature)
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
		}() + ": " + fn.Comment,
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
		PackagesInfo[pkgDirUnix].NonEmpty = true
		im := PackagesInfo[pkgDirUnix].ImportsNative
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
			if _, ok := GoCode[pkgDirUnix].InitVars[ti]; !ok {
				GoCode[pkgDirUnix].InitVars[ti] = map[string]*FnCodeInfo{}
			}
			GoCode[pkgDirUnix].InitVars[ti][fn.BaseName] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFname, Params: fn.Signature.Params(), FnDoc: fn.Doc}
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
				GoCode[pkgDirUnix].InitVars[ti][fn.BaseName] = &FnCodeInfo{SourceFile: fn.SourceFile, FnCode: goFname, FnDecl: fn.Fd, Params: fn.Signature.Params(), FnDoc: fn.Doc}
			}
		}
	}

	if goFn != "" {
		var params *types.Tuple
		if fn.Fd != nil {
			params = fn.Signature.Params()
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

	goFname := genutils.FuncNameAsGoPrivate(d.Name.Name)
	fc := genFuncCode(fn, fn.Signature)
	clojureReturnType, goReturnType := genutils.ClojureReturnTypeForGenerateCustom(fc.clojureReturnType)

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
		pi := PackagesInfo[pkgDirUnix]
		pi.NonEmpty = true
		if clojureReturnType == "" {
			im := pi.ImportsNative
			im.InternPackage(godb.ClojureCoreDir, "", "", fn.Pos)
			myGoImport := im.AddPackage(pkgDirUnix, "", "", "", true, fn.Pos)
			goFn = strings.ReplaceAll(goFn, "{{myGoImport}}", myGoImport)
			im.Promote(fn.ImportsNative, fn.Pos)
		} else {
			// No Go code needs to be generated when a return type is explicitly specified.
			pi.ImportsAutoGen.AddPackage(pkgDirUnix, fn.SourceFile.Package.NsRoot, "", "", false, fn.Pos)
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

	ClojureCode[pkgDirUnix].Constants[ci.Name.Name] = ci
}

func GenVariable(vi *VariableInfo) {
	genutils.GenSymReset()
	pkgDirUnix := vi.SourceFile.Package.Dir.String()

	PackagesInfo[pkgDirUnix].NonEmpty = true

	myGoImport := PackagesInfo[pkgDirUnix].ImportsAutoGen.AddPackage(pkgDirUnix, vi.SourceFile.Package.NsRoot, "", "", true, vi.Name.NamePos)
	vi.Def = strings.ReplaceAll(vi.Def, "{{myGoImport}}", myGoImport)

	ClojureCode[pkgDirUnix].Variables[vi.Name.Name] = vi
}

func maybeImplicitConvert(typeName string, ti TypeInfo) string {
	ts := ti.TypeSpec()
	if ts == nil {
		return ""
	}

	t := TypeInfoForExpr(ts.Type)
	if t.IsCustom() {
		return ""
	}

	argType := t.ArgClojureType()
	declType := t.ArgExtractFunc()
	if argType == "" || declType == "" {
		return ""
	}

	coerceApi := "ObjectAs" + declType
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
	pi := PackagesInfo[pkgDirUnix]

	pi.NonEmpty = true
	where := ts.Pos()

	pi.ImportsNative.InternPackage(godb.ClojureCoreDir, "", "", where)
	myGoImport := pi.ImportsNative.AddPackage(pkgDirUnix, "", "", "", true, where)

	ClojureCode[pkgDirUnix].Types[t] = ti
	GoCode[pkgDirUnix].Types[t] = ti

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

	info["Others"] = maybeImplicitConvert(typeName, ti)

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

	NewDefinedApi(pi.ClojureNameSpace+"/"+MaybeIsApiName, "codegen.go/GenType()")
	NewDefinedApi(pi.ClojureNameSpace+"/"+ExtractApiName, "codegen.go/GenType()")
	NewDefinedApi(pi.ClojureNameSpace+"/"+FieldAsApiName, "codegen.go/GenType()")
	NewDefinedApi(pi.ClojureNameSpace+"/"+ReceiverArgAsApiName, "codegen.go/GenType()")
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
	if strings.Contains(goConstructor, "ABEND") {
		goConstructor = nonEmptyLineRegexp.ReplaceAllString(goConstructor, `// $1`)
		goConstructor = strings.ReplaceAll(goConstructor, "{{myGoImport}}", path.Base(pkgDirUnix))
		abends.TrackAbends(goConstructor)
	} else {
		pi := PackagesInfo[pkgDirUnix]
		pi.ImportsNative.Promote(tyi.RequiredImports(), tyi.DefPos())
		myGoImport := pi.ImportsNative.AddPackage(pkgDirUnix, "", "", "", true, tyi.DefPos())
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

func SetSwitchableTypes(allTypesSorted []TypeInfo) {
	var types []TypeInfo
	ord := 0

	for _, ti := range allTypesSorted {
		more := false
		if false && strings.Contains(ti.GoName(), "FileMode") {
			fmt.Printf("codegen.go/GenTypeInfo(): %s == %+v %+v\n", ti.ClojureName(), ti.GoTypeInfo(), ti.ClojureTypeInfo())
			more = true
		}
		if !ti.IsCustom() {
			if uti := ti.UnderlyingTypeInfo(); uti == nil || !uti.IsCustom() {
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
		Ordinal[ti] = ord + 1
		if more {
			fmt.Printf("codegen.go/GenTypeInfo(): assigned ordinal %3d to %s @%p (specificity=%d)\n", ord, ti, ti, ti.Specificity())
		}
		ord++
	}

	SwitchableTypes = types
}

func addQualifiedFunction(ti TypeInfo, typeBaseName, receiverId, name, embedName, fullName, baseName, comment string, doc *CommentGroup, xft interface{}, pos token.Pos) {
	sig := (*types.Signature)(nil)
	switch x := xft.(type) {
	case *types.Signature:
		sig = x
	default:
		panic(fmt.Sprintf("unexpected type %T", xft))
	}
	if f, found := QualifiedFunctions[fullName]; found {
		if f.EmbedName != "" && f.EmbedName != name {
			//			fmt.Fprintf(os.Stderr, "codegen.go/addQualifiedFunction: not replacing %s with %s\n", f.EmbedName, name)
			QualifiedFunctions[fullName] = nil
		}
		return
	}
	if ti.GoFile() == nil {
		fmt.Fprintf(os.Stderr, "codegen.go/addQualifiedFunction(): No GoFile() for %s\n", ti.GoName())
		return
	}
	QualifiedFunctions[fullName] = &FuncInfo{
		BaseName:       name,
		ReceiverId:     receiverId,
		Name:           baseName,
		DocName:        "(" + ti.GoFile().Package.Dir.String() + "." + typeBaseName + ")" + name + "()",
		EmbedName:      embedName,
		Fd:             nil,
		ToM:            ti,
		Signature:      sig,
		Doc:            doc,
		SourceFile:     ti.GoFile(),
		ImportsNative:  &imports.Imports{},
		ImportsAutoGen: &imports.Imports{},
		Pos:            pos,
		Comment:        comment,
	}
}

func appendMethods(ti TypeInfo, ity *InterfaceType, comment string) {
	d, ok := astutils.TypeCheckerInfo.Types[ity]
	if !ok {
		fmt.Fprintf(os.Stderr, "codegen.go/appendMethods(): Cannot find def for %T %+v\n", ity, ity)
		return
	}
	iface, yes := d.Type.(*types.Interface)
	if !yes {
		return
	}

	typeFullName := ti.GoName()
	typeBaseName := ti.GoBaseName()
	receiverId := "{{myGoImport}}." + typeBaseName

	num := iface.NumMethods()
	for i := 0; i < num; i++ {
		m := iface.Method(i)
		name := m.Name()
		doc := &CommentGroup{}
		addQualifiedFunction(
			ti,
			typeBaseName,
			receiverId,
			name,
			"", /* embedName*/
			typeFullName+"_"+name,
			typeBaseName+"_"+name,
			comment,
			doc,
			m.Type(),
			m.Pos())
	}
	// num := iface.NumEmbeddeds()
	// for i := 0; i < num; i++ {
	// 	m := iface.EmbeddedType(i)
	// 	appendMethods(ti, ts.(*TypeSpec).Type.(*InterfaceType), "embedded interface")
	// }
}

// ptr is true when processing type *T (thus adding to *T's list of functions), false otherwise.
func appendReceivers(ti TypeInfo, ty *StructType, ptr bool, comment string) {
	d, ok := astutils.TypeCheckerInfo.Types[ty]
	if !ok {
		fmt.Fprintf(os.Stderr, "codegen.go/appendReceivers(): Cannot find def for %T %+v\n", ty, ty)
		return
	}
	s, yes := d.Type.(*types.Struct)
	if !yes {
		return
	}

	typePkgName := ti.GoFile().Package.Dir.String()
	typeFullName := ti.GoName()
	typeBaseName := ti.GoBaseName()
	receiverId := "{{myGoImport}}." + typeBaseName

	n := s.NumFields()
	for i := 0; i < n; i++ {
		v := s.Field(i)
		if !v.Embedded() {
			continue
		}

		embedName := v.Name()

		f := func(p types.Type) {
			receivingTypeName := astutils.TypePathname(p)
			m, found := ReceivingTypes[receivingTypeName]

			//			fmt.Fprintf(os.Stderr, "codegen.go/appendReceivers(): %s (found=%v):\n", receivingTypeName, found)

			if !found {
				return
			}

			for _, fd := range m {
				name := fd.Name.Name
				//				fmt.Fprintf(os.Stderr, "codegen.go/appendReceivers(): %s\n", name)

				if overriddenByMethod(typePkgName, typeBaseName, name) {
					// For type T embedding type
					// U, which implements (U)F(),
					// do not emit that
					// (embedded/lifted) function
					// if (*T)F() is
					// defined. Otherwise, the
					// generated (T)F() wrapper
					// will actually call (*T)F()
					// via &T, which (currently)
					// Joker-gostd doesn't
					// support, due to embedding T
					// as a GoObject[interface{}]
					// of T, not *T.
					fmt.Fprintf(os.Stderr, "codegen.go/appendReceivers: inhibiting overridden method (%s)%s() while processing %s (embed=%s)\n", receivingTypeName, name, typeFullName+"_"+name, embedName)
					continue
				}

				var sig *types.Signature
				if ty, ok := astutils.TypeCheckerInfo.Defs[fd.Name]; !ok {
					fmt.Fprintf(os.Stderr, "codegen.go/appendReceivers: no info on %s.%s\n", typeFullName, fd.Name)
				} else {
					sig = ty.Type().(*types.Signature)
					if sig == nil {
						fmt.Fprintf(os.Stderr, "codegen.go/appendReceivers: no signature for %s.%s\n", typeFullName, fd.Name)
					}
				}
				doc := fd.Doc
				addQualifiedFunction(
					ti,
					typeBaseName,
					receiverId,
					name,
					embedName,
					typeFullName+"_"+name,
					typeBaseName+"_"+name,
					comment,
					doc,
					sig,
					fd.Name.NamePos)
			}
		}

		p := v.Type()
		if ptr { // Adding to *T's list of methods
			f(p)
		} else {
			if _, yes := p.(*types.Pointer); !yes {
				f(p)
				//			f(types.NewPointer(p))
			}
		}
	}
}

func overriddenByMethod(typeName, baseName, name string) bool {
	n := typeName + ".PtrTo_" + baseName + "_" + name
	f, found := QualifiedFunctions[n]
	if found && f.Fd != nil {
		return true
	}
	n = typeName + "." + baseName + "_" + name
	f, found = QualifiedFunctions[n]
	return found && f.Fd != nil
}

func GenQualifiedFunctionsFromEmbeds(allTypesSorted []TypeInfo) {
	for _, ti := range allTypesSorted {
		if ti.ClojureName() == "crypto/Hash" || true {
			//		fmt.Printf("codegen.go/GenTypeFromDb: %s == @%p %+v @%p %+v @%p %+v\n", ti.ClojureName(), ti, ti, ti.ClojureTypeInfo(), ti.ClojureTypeInfo(), ti.GoTypeInfo(), ti.GoTypeInfo())
		}

		if !ti.IsExported() || ti.IsArbitraryType() {
			//		fmt.Printf("codegen.go/GenTypeFromDb: not exported or a special type\n")
			continue // Do not generate anything for private or special types
		}

		ty := ti.GoTypeExpr()

		if ty != nil {
			switch ty := ty.(type) {
			case *InterfaceType:
				appendMethods(ti, ty, "declared interface")
			case *StructType:
				appendReceivers(ti, ty, false, "embedded type having defined function")
			case *StarExpr:
				switch ty := ty.X.(type) {
				case *StructType:
					appendReceivers(ti, ty, true, "embedded pointer type having defined function")
				}
			}
		}
	}
}

func GenTypeCtors(allTypesSorted []TypeInfo) {
	for _, ti := range allTypesSorted {
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

	nonGoObjectCaseDoc = fmt.Sprintf("GoObject[%s] or: %s", goTypeName, strings.Join(nonGoObjectTypeDocs, " or "))
	helperFunc = strings.Join(helperFuncs, "")
	return
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
		nonGoObjectType, nonGoObjectTypeDoc, extractClojureObject := simpleTypeFor(ti.GoFile().Package.Dir.String(), t.Name)
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

	nonGoObjectTypes = []string{"default"}
	nonGoObjectTypeDocs = []string{"whatever"}
	extractClojureObjects = []string{fmt.Sprintf("%s(_o.ABEND674(codegen.go: unknown underlying type %T for %s))",
		goTypeName, ts.Type, clojureTypeName)}
	helperFuncs = []string{""}

	return
}

func simpleTypeFor(pkgDirUnix, name string) (nonGoObjectType, nonGoObjectTypeDoc, extractClojureObject string) {
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

func elementsToType(ti TypeInfo, ty *StructType, toType func(TypeInfo, int, string, *Field) string) string {
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

func mapElementToType(ti TypeInfo, _ int, name string, f *Field) string {
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
	clType := v.ClojureEffectiveName()
	apiImportName := addApiToImports(ti, clType) // apiImportName := AddApiToImports(clType)
	api := determineRuntime("FieldAs", "FieldAs_ns_", apiImportName, clType)

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

func init() {
	nonEmptyLineRegexp = regexp.MustCompile(`(?m)^(.)`)
}
