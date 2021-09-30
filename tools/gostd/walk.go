package main

import (
	"bytes"
	"fmt"
	"github.com/candid82/joker/tools/gostd/astutils"
	"github.com/candid82/joker/tools/gostd/genutils"
	"github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/imports"
	"github.com/candid82/joker/tools/gostd/paths"
	. "go/ast"
	"go/build"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"sort"
	. "strings"
)

var WalkDump bool

var NumFunctions int
var NumStandalones int
var NumReceivers int
var NumTypes int
var NumConstants int
var NumVariables int
var NumCtableTypes int
var NumGeneratedFunctions int
var NumGeneratedStandalones int
var NumGeneratedReceivers int
var NumGeneratedConstants int
var NumGeneratedVariables int
var NumGeneratedCtors int

type PackageInfo struct {
	ImportsNative  *imports.Imports
	ImportsAutoGen *imports.Imports
	Pkg            *Package
	NonEmpty       bool   // Whether any non-comment code has been generated
	HasGoFiles     bool   // Whether any .go files (would) have been generated
	Namespace      string // E.g.: "go.std.net", "x.y.z.whatever"
}

/* Map (Unix-style) relative path to package info */
var PackagesInfo = map[string]*PackageInfo{}

/* Sort the packages -- currently appears to not actually be
/* necessary, probably because of how walkDirs() works. */
func SortedPackagesInfo(m map[string]*PackageInfo, f func(string, *PackageInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

type FnCodeInfo struct {
	SourceFile *godb.GoFile
	FnCode     string
	FnDecl     *FuncDecl // Empty for standalones and methods; used to get docstring for receivers
	Params     *types.Tuple
	FnDoc      *CommentGroup
}

type fnCodeMap map[string]*FnCodeInfo

type CodeInfo struct {
	Constants GoConstantsMap
	Variables GoVariablesMap
	Functions fnCodeMap
	Types     TypesMap
	InitTypes map[TypeInfo]struct{}               // types to be initialized
	InitVars  map[TypeInfo]map[string]*FnCodeInfo // func initNative()'s "info_key1 = ... { key2: value, ... }"
}

/* Map relative (Unix-style) package names to maps of function names to code info and strings. */
var ClojureCode = map[string]CodeInfo{}
var ClojureCodeForType = map[TypeInfo]string{}
var GoCode = map[string]CodeInfo{}
var GoCodeForType = map[TypeInfo]string{}

func SortedPackageMap(m map[string]CodeInfo, f func(string, CodeInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

func SortedCodeMap(m CodeInfo, f func(string, *FnCodeInfo)) {
	var keys []string
	for k, _ := range m.Functions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m.Functions[k])
	}
}

func SortedFnCodeInfo(m map[string]*FnCodeInfo, f func(string, *FnCodeInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

type FuncInfo struct {
	BaseName       string    // Just the name without receiver-type info
	ReceiverId     string    // Receiver info (only one type supported here and by Golang itself for now)
	Name           string    // Unique name for implementation (has Receiver info as a prefix, then baseName)
	DocName        string    // Everything, for documentation and diagnostics
	EmbedName      string    // "" for function definitions, else basename of embedded type
	Fd             *FuncDecl // nil for methods (these are declared within interface{} bodies, which are not fn declarations)
	ToM            TypeInfo  // Method operates on this type (nil for standalones and receivers)
	Signature      *types.Signature
	Doc            *CommentGroup
	SourceFile     *godb.GoFile
	ImportsNative  *imports.Imports // Add these to package imports if function is generated (no ABENDs)
	ImportsAutoGen *imports.Imports // Add these to package imports if function is generated (no ABENDs)
	Pos            token.Pos
	Comment        string
}

func initPackage(pkgDirUnix string, p *Package) {
	if godb.Verbose {
		genutils.AddSortedStdout(fmt.Sprintf("Processing package=%s:\n", pkgDirUnix))
	}

	me := generatedGoStdPrefix + pkgDirUnix

	if _, ok := PackagesInfo[pkgDirUnix]; !ok {
		PackagesInfo[pkgDirUnix] = &PackageInfo{
			ImportsNative:  &imports.Imports{Me: me, MySourcePkg: pkgDirUnix, For: "Native " + pkgDirUnix},
			ImportsAutoGen: &imports.Imports{Me: me, MySourcePkg: pkgDirUnix, For: "AutoGen " + pkgDirUnix},
			Pkg:            p,
			NonEmpty:       false,
			HasGoFiles:     false,
			Namespace:      godb.ClojureNamespaceForDirname(pkgDirUnix),
		}
		GoCode[pkgDirUnix] = CodeInfo{GoConstantsMap{}, GoVariablesMap{}, fnCodeMap{}, TypesMap{},
			map[TypeInfo]struct{}{}, map[TypeInfo]map[string]*FnCodeInfo{}}
		ClojureCode[pkgDirUnix] = CodeInfo{GoConstantsMap{}, GoVariablesMap{}, fnCodeMap{}, TypesMap{},
			map[TypeInfo]struct{}{}, map[TypeInfo]map[string]*FnCodeInfo{}}
	}
}

// Given an (possibly fully qualified) identifier name and a position
// pos, return a suitable (Go) reference to that identifier with the
// package qualifier shortened to just the base and that maps to the
// package (which might well be added to appropriate Imports for the
// package at the position).
func refToIdent(ident string, pos token.Pos, autoGen bool) string {
	ix := LastIndex(ident, ".")
	if ix < 0 {
		return ident
	}
	pkgName := ident[0:ix]

	goPkgForPos := godb.GoPackageForPos(pos)
	ns := ""
	if pi, found := PackagesInfo[goPkgForPos]; found {
		var imp *imports.Imports
		if autoGen {
			imp = pi.ImportsAutoGen
			ns = "(" + godb.GetPackageNamespace(pkgName) + ")" // Avoid key collisions compiling a_*.go files
		} else {
			imp = pi.ImportsNative
		}
		myImportId := imp.AddPackage(pkgName, ns, true, pos, "walk.go/refToIdent")
		return myImportId + "." + ident[ix+1:]
	}
	panic(fmt.Sprintf("cannot find package %q for reference at %s", goPkgForPos, godb.WhereAt(pos)))
}

/* Go apparently doesn't support/allow 'interface{}' as the value (or
/* key) of a map such that any arbitrary type can be substituted at
/* run time, so there are several of these nearly-identical functions
/* sprinkled through this code. Still get some reuse out of some of
/* them, and it's still easier to maintain these copies than if the
/* body of these were to be included at each call point.... */
func SortedFuncInfoMap(m map[string]*FuncInfo, f func(string, *FuncInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

// Add whatever ti needs to be code-generated for fn to fn's list of
// imports; return what is picked as the Go short package name for the
// generated file.
func (fn *FuncInfo) AddToAutoGen(ti TypeInfo) string {
	exprPkgName := ti.GoPackage()
	curPkgName := fn.SourceFile.Package.Dir
	if exprPkgName == "" {
		return ""
	}
	clojureStdPath := generatedPkgPrefix + ReplaceAll(ti.Namespace(), ".", "/")

	autoGen := fn.ImportsAutoGen.AddPackage(clojureStdPath, ti.Namespace(), true, fn.Pos, "walk.go/AddToAutoGen")
	if Contains(fn.Name, "Chmod") {
		fmt.Fprintf(os.Stderr, "walk.go/(%q)AddToAutoGen(%s): adding [%s %q]:\n  %+v\n", curPkgName.String()+"."+fn.Name, ti.GoName(), autoGen, clojureStdPath, fn.ImportsAutoGen)
	}
	return autoGen
}

func (fn *FuncInfo) AddApiToImports(clType string) string {
	ix := Index(clType, "/")
	if ix < 0 {
		return "" // builtin type (api is in core)
	}

	if Contains(clType, "ABEND") {
		return "" // An error in the making, but try to just not generate anything here
	}

	apiPkgPath := generatedPkgPrefix + ReplaceAll(clType[0:ix], ".", "/")
	fmt.Fprintf(os.Stderr, "walk.go/AddApiToImports: Comparing %s to %s\n", apiPkgPath, fn.SourceFile.Package.ImportMe)
	if apiPkgPath == fn.SourceFile.Package.ImportMe {
		return "" // api is local to function
	}

	native := fn.ImportsNative.AddPackage(apiPkgPath, "", true, fn.Pos, "walk.go/AddApiToImports")

	return native
}

func processTypeRef(t Expr) {
	defer func() {
		if x := recover(); x != nil {
			panic(fmt.Sprintf("(Panic at %s processing %s: %s)\n", godb.WhereAt(t.Pos()), astutils.ExprToString(t), x))
		}
	}()

	if t != nil {
		TypeInfoForExpr(t)
	}
}

func processFieldsForTypes(items []astutils.FieldItem) {
	for _, f := range items {
		processTypeRef(f.Field.Type)
	}
}

func declFuncForTypes(_ *godb.GoFile, _ string, _ *File, fd *FuncDecl) {
	if !IsExported(fd.Name.Name) {
		return // Skipping non-exported functions
	}

	processFieldsForTypes(astutils.FlattenFieldList(fd.Recv))
	processFieldsForTypes(astutils.FlattenFieldList(fd.Type.Params))
	processFieldsForTypes(astutils.FlattenFieldList(fd.Type.Results))
}

func processValueSpecsForTypes(_ *godb.GoFile, _ string, tss []Spec, _ *CommentGroup) {
	for _, spec := range tss {
		ts := spec.(*ValueSpec)
		if astutils.AnyNamesExported(ts.Names) {
			processTypeRef(ts.Type)
		}
	}
}

// Map qualified function names to info on each.
var QualifiedFunctions = map[string]*FuncInfo{}
var ReceivingTypes = map[string][]*FuncDecl{}

func receiverPrefix(src *godb.GoFile, rl []astutils.FieldItem) string {
	res := ""
	for i, r := range rl {
		if i != 0 {
			res += "_"
		}
		switch x := r.Field.Type.(type) {
		case *Ident:
			res += x.Name
		case *ArrayType:
			res += "ArrayOf_" + x.Elt.(*Ident).Name
		case *StarExpr:
			res += "PtrTo_" + x.X.(*Ident).Name
		default:
			panic(fmt.Sprintf("receiverList: unrecognized expr %T in %s", x, src.Name))
		}
	}
	return res + "_"
}

func receiverId(src *godb.GoFile, pkg string, rl []astutils.FieldItem) string {
	if pkg == "" {
		pkg = "{{myGoImport}}."
	} else {
		pkg += "."
	}
	res := ""
	for i, r := range rl {
		if i != 0 {
			res += "ABEND422(more than one receiver in list)"
		}
		switch x := r.Field.Type.(type) {
		case *Ident:
			res += pkg + x.Name
		case *ArrayType:
			res += "[]" + pkg + x.Elt.(*Ident).Name
		case *StarExpr:
			res += "*" + pkg + x.X.(*Ident).Name
		default:
			panic(fmt.Sprintf("receiverId: unrecognized expr %T in %s", x, src.Name))
		}
	}
	return res
}

func processFuncDecl(gf *godb.GoFile, pkgDirUnix string, _ *File, fd *FuncDecl, isExportable bool) {
	if WalkDump {
		fmt.Printf("Func in pkgDirUnix=%s filename=%s fd=%p exportable=%v fd.Doc=%p:\n", pkgDirUnix, godb.FileAt(fd.Pos()), fd, isExportable, fd.Doc)
		Print(godb.Fset, fd)
	}
	fl := astutils.FlattenFieldList(fd.Recv)
	fnName := receiverPrefix(gf, fl) + fd.Name.Name
	fullName := pkgDirUnix + "." + fnName
	switch fullName {
	case "unsafe._Offsetof", "unsafe._Add", "unsafe._Slice", "unsafe._Sizeof", "unsafe._Alignof":
		return // unsafe.Offsetof et al are syntactic operations in Go.
	}

	if len(fl) == 1 {
		// Add exported receivers to list of those supported
		// for this type, so any structs that embed this type
		// will have their own copies of adapters to those
		// receivers created for them.
		typeName := astutils.TypePathnameFromExpr(fl[0].Field.Type)
		if _, found := ReceivingTypes[typeName]; !found {
			ReceivingTypes[typeName] = []*FuncDecl{}
		}
		ReceivingTypes[typeName] = append(ReceivingTypes[typeName], fd)
		//		fmt.Fprintf(os.Stderr, "walk.go/processFuncDecl(): %s => %+v\n", typeName, fd)
	}

	if !isExportable {
		// Do not emit a direct wrapper for this function.
		return
	}

	if v, ok := QualifiedFunctions[fullName]; ok {
		genutils.AddSortedStdout(fmt.Sprintf("NOTE: Already seen function %s in %s, yet again in %s",
			fullName, v.SourceFile.Name, godb.FileAt(fd.Pos())))
	}
	rcvrId := receiverId(gf, "", fl)
	docName := "(" + receiverId(gf, pkgDirUnix, fl) + ")" + fd.Name.Name + "()"
	// if Contains(fullName, "DotNode") {
	// 	fmt.Fprintf(os.Stderr, "walk.go/processFuncDecl: %s\n", fullName)
	// }
	var sig *types.Signature
	if ty, ok := astutils.TypeCheckerInfo.Defs[fd.Name]; !ok {
		fmt.Fprintf(os.Stderr, "walk.go/processFuncDecl: no info on %s.%s\n", pkgDirUnix, fd.Name)
	} else {
		sig = ty.Type().(*types.Signature)
		if sig == nil {
			fmt.Fprintf(os.Stderr, "walk.go/processFuncDecl: no signature for %s.%s\n", pkgDirUnix, fd.Name)
		}
	}

	me := generatedGoStdPrefix + pkgDirUnix
	file := PackagesInfo[pkgDirUnix]

	QualifiedFunctions[fullName] = &FuncInfo{
		BaseName:       fd.Name.Name,
		ReceiverId:     rcvrId,
		Name:           fnName,
		DocName:        docName,
		EmbedName:      "",
		Fd:             fd,
		ToM:            nil,
		Signature:      sig,
		Doc:            fd.Doc,
		SourceFile:     gf,
		ImportsNative:  &imports.Imports{FileImports: file.ImportsNative, Me: me, MySourcePkg: pkgDirUnix, For: "Native " + fullName},
		ImportsAutoGen: &imports.Imports{FileImports: file.ImportsAutoGen, Me: me, MySourcePkg: pkgDirUnix, For: "AutoGen " + fullName},
		Pos:            fd.Pos(),
		Comment:        "defined function",
	}
}

func processTypeDecls(gf *godb.GoFile, pkg string, tss []Spec, parentDoc *CommentGroup) {
	var ts *TypeSpec

	defer func() {
		if x := recover(); x != nil {
			panic(fmt.Sprintf("(Panic at %s processing %+v: %s)", godb.WhereAt(ts.Pos()), ts, x))
		}
	}()

	for _, spec := range tss {
		ts = spec.(*TypeSpec)
		RegisterTypeDecl(ts, gf, pkg, parentDoc)
	}
}

func processTypesForTypeDecls(_ *godb.GoFile, _ string, tss []Spec, _ *CommentGroup) {
	var ts *TypeSpec

	defer func() {
		if x := recover(); x != nil {
			panic(fmt.Sprintf("(Panic at %s processing %+v: %s)", godb.WhereAt(ts.Pos()), ts, x))
		}
	}()

	for _, spec := range tss {
		ts = spec.(*TypeSpec)
		RegisterAllSubtypes(ts.Type)
	}
}

type VariableInfo struct {
	Name       *Ident
	SourceFile *godb.GoFile
	Def        string
	Pos        token.Pos
}

type GoVariablesMap map[string]*VariableInfo

var GoVariables = GoVariablesMap{}

func SortedVariableInfoMap(m map[string]*VariableInfo, f func(string, *VariableInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

type ConstantInfo struct {
	Name       *Ident
	SourceFile *godb.GoFile
	Def        string
	Pos        token.Pos
}

type GoConstantsMap map[string]*ConstantInfo

var GoConstants = GoConstantsMap{}

func SortedConstantInfoMap(m map[string]*ConstantInfo, f func(string, *ConstantInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

// Returns the Clojure type and a constant expression of that type.
// E.g. "GoObject" and "net.Flags(1)" (which becomes net.FlagsUp).
// The returned <gl> argument must be returned inside double
// quotes. If it represents an unwrapped (builtin) string, it needs to
// be quoted "%q"-style, which .ExactString() does for strings.  If
// wrapped by a type, the outer expression must still have double
// quotes.
func genCodeForConstant(constObj types.Object, origVal Expr) (cl, gl string) {
	c := constObj.(*types.Const)
	typ, val := types.Default(c.Type()), c.Val()

	typeName := typ.String()
	ti := TypeInfoForType(typ)
	var valPat string

	if typ.Underlying() != nil && typ.Underlying() != typ {
		cl = "GoObject"
		valPat = fmt.Sprintf("%s(%%s)", refToIdent(typeName, c.Pos(), true))
	} else {
		cl = ti.ArgClojureArgType()
		valPat = ti.PromoteType()
	}

	// Though documented as returning a "quoted" string,
	// .ExactString() uses strconv.Quote() only for
	// constant.String types; the other types are not surrounded
	// by double quotes.

	numPat := "%s"
	switch val.Kind() {
	case constant.String:
		gl = val.ExactString()
	case constant.Float:
		f, ok := constant.Float64Val(val)
		gl = fmt.Sprintf("%g", f)
		if !ok {
			cl = "BigFloat"
			gl = val.ExactString()
			numPat = "MakeMathBigFloatFromString(%q)"
		}
	case constant.Int:
		f, ok := constant.Int64Val(val)
		gl = fmt.Sprintf("%d", f)
		if !ok {
			if cl != "GoObject" {
				cl = "Number"
			}
			u, _ := constant.Uint64Val(val)
			gl = fmt.Sprintf("%d", u)
			numPat = "uint64(%s)"
		}
	default:
		gl = val.String()
	}

	if lit := isBasicLiteral(origVal); lit != nil {
		// After determining what type is suitable based on
		// the value, substitute the original literal if
		// available.
		gl = lit.Value
	}

	if cl == "BigFloat" && Contains(gl, "/") {
		cl = "Ratio"
		numPat = "MakeMathBigRatFromString(%q)"
	}

	return cl, fmt.Sprintf("%q", fmt.Sprintf(valPat, fmt.Sprintf(numPat, gl)))
}

func isBasicLiteral(e Expr) *BasicLit {
	if e == nil {
		return nil
	}
	switch v := e.(type) {
	case *BasicLit:
		return v
	default:
		return nil
	}
}

func processConstantSpec(gf *godb.GoFile, pkg string, name *Ident, val Expr, docString string) bool {
	defer func() {
		if x := recover(); x != nil {
			fmt.Fprintf(os.Stderr, "(Panic due to: %s: %+v)\n", godb.WhereAt(name.Pos()), x)
		}
	}()

	clName := name.Name
	fullName := pkg + "." + name.Name

	if c, ok := GoConstants[fullName]; ok {
		fmt.Fprintf(os.Stderr, "WARNING: constant %s found at %s and now again at %s\n",
			fullName, godb.WhereAt(c.Name.NamePos), godb.WhereAt(name.NamePos))
	}

	valTypeString, goCode := genCodeForConstant(astutils.TypeCheckerInfo.Defs[name], val)

	if valTypeString == "" {
		return false
	}

	// Note: :tag value is a string to avoid conflict with like-named member of namespace
	constantDefInfo := map[string]string{
		"DocString":       docString,
		"ValueTypeString": valTypeString,
		"GoCode":          goCode,
		"ClojureName":     clName,
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "clojure-constant-def.tmpl", constantDefInfo)

	gt := &ConstantInfo{name, gf, buf.String(), val.Pos()}
	GoConstants[fullName] = gt
	NumGeneratedConstants++

	return true
}

func processVariableSpec(gf *godb.GoFile, pkg string, name *Ident, docString string) bool {
	clName := name.Name
	localName := gf.Package.BaseName + "." + name.Name
	fullName := pkg + "." + name.Name

	if c, ok := GoVariables[fullName]; ok {
		fmt.Fprintf(os.Stderr, "WARNING: variable %s found at %s and now again at %s\n",
			localName, godb.WhereAt(c.Name.NamePos), godb.WhereAt(name.NamePos))
	}

	if WalkDump {
		fmt.Printf("Variable %s at %s.\n", name, godb.WhereAt(name.Pos()))
	}

	// Note: :tag value is a string to avoid conflict with like-named member of namespace
	variableDefInfo := map[string]string{
		"DocString":   docString,
		"LocalName":   localName,
		"ClojureName": clName,
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "clojure-variable-def.tmpl", variableDefInfo)

	gt := &VariableInfo{name, gf, buf.String(), name.NamePos}
	GoVariables[fullName] = gt
	NumGeneratedVariables++

	return true
}

func what(constant bool) string {
	if constant {
		return "Constant"
	}
	return "Variable"
}

func processValueSpecs(gf *godb.GoFile, pkg string, tss []Spec, parentDoc *CommentGroup, constant bool) {
	var previousVal, previousValType Expr
	for ix, spec := range tss {
		ts := spec.(*ValueSpec)
		for jx, valName := range ts.Names {
			valType := ts.Type
			var val Expr
			if ts.Values != nil {
				if jx >= len(ts.Values) {
					// This seems crazy (more names receiving values than values??) until one
					// investigates the single case that hits this, os/executable_procfs.go, which
					// does something like "var a, b = func() (bool, bool) { ... }()", i.e. the
					// names receive the values returned by the function.
					val = nil
				} else {
					val = ts.Values[jx]
				}
			}

			if val == nil {
				val = previousVal
			}
			if valType == nil {
				valType = previousValType
			}

			if constant {
				previousVal = val
				previousValType = valType
			}

			if !IsExported(valName.Name) {
				continue
			}
			if constant {
				NumConstants++
			} else {
				NumVariables++
			}

			if WalkDump {
				fmt.Printf("%s #%d of spec #%d %s at %s:\n", what(constant), jx, ix, valName, godb.WhereAt(valName.NamePos))
				if valType != nil {
					fmt.Printf("  valType:\n")
					Print(godb.Fset, valType)
				}
				if val != nil {
					fmt.Printf("  val:\n")
					Print(godb.Fset, val)
				}
			}
			doc := ts.Doc // Try block comments for this specific decl
			if doc == nil {
				doc = ts.Comment // Use line comments if no preceding block comments are available
			}
			if doc == nil {
				doc = parentDoc // Use 'var'/'const' statement block comments as last resort
			}
			docString := genutils.CommentGroupInQuotes(doc, "", "", "", "")
			if constant {
				processConstantSpec(gf, pkg, valName, val, docString)
			} else {
				processVariableSpec(gf, pkg, valName, docString)
			}
		}
	}
}

func declFunc(gf *godb.GoFile, pkgDirUnix string, f *File, v *FuncDecl) {
	if !IsExported(v.Name.Name) {
		return // Skipping non-exported functions
	}

	isExportable := true

	if v.Recv != nil {
		// Count the func as eligible if the types of all its
		// receivers (Golang supports only one for now,
		// thankfully) are exported. But process it
		// regardless, because another exported type (in the
		// same package) can embed an unexported type serving
		// as a receiver to one or more functions, which are
		// then magically accessible to users of that exported
		// type.
		for _, r := range v.Recv.List {
			if !astutils.IsExportedType(&r.Type) {
				isExportable = false
			}
		}
		if isExportable {
			NumReceivers++
		}
	} else {
		NumStandalones++
	}

	if isExportable {
		NumFunctions++
	}

	processFuncDecl(gf, pkgDirUnix, f, v, isExportable)
}

func declType(gf *godb.GoFile, pkgDirUnix string, _ *File, v *GenDecl) {
	processTypeDecls(gf, pkgDirUnix, v.Specs, v.Doc)
}

func declTypesForTypes(gf *godb.GoFile, pkgDirUnix string, _ *File, v *GenDecl) {
	processTypesForTypeDecls(gf, pkgDirUnix, v.Specs, v.Doc)
}

func declValueSpecForTypes(gf *godb.GoFile, pkgDirUnix string, specs []Spec, doc *CommentGroup) {
	processValueSpecsForTypes(gf, pkgDirUnix, specs, doc)
}

func declConstSpec(gf *godb.GoFile, pkgDirUnix string, specs []Spec, doc *CommentGroup) {
	processValueSpecs(gf, pkgDirUnix, specs, doc, true)
}

func declVarSpec(gf *godb.GoFile, pkgDirUnix string, specs []Spec, doc *CommentGroup) {
	processValueSpecs(gf, pkgDirUnix, specs, doc, false)
}

type fileDeclFuncs struct {
	FuncDecl  func(*godb.GoFile, string, *File, *FuncDecl)
	TypeDecl  func(*godb.GoFile, string, *File, *GenDecl)
	ConstDecl func(*godb.GoFile, string, []Spec, *CommentGroup)
	VarDecl   func(*godb.GoFile, string, []Spec, *CommentGroup)
}

func processDecls(gf *godb.GoFile, pkgDirUnix string, f *File, declFuncs fileDeclFuncs) {
	for _, s := range f.Decls {
		switch v := s.(type) {
		case *FuncDecl:
			if declFuncs.FuncDecl != nil {
				declFuncs.FuncDecl(gf, pkgDirUnix, f, v)
			}
		case *GenDecl:
			switch v.Tok {
			case token.TYPE:
				if declFuncs.TypeDecl != nil {
					declFuncs.TypeDecl(gf, pkgDirUnix, f, v)
				}
			case token.CONST:
				if declFuncs.ConstDecl != nil {
					declFuncs.ConstDecl(gf, pkgDirUnix, v.Specs, v.Doc)
				}
			case token.VAR:
				if declFuncs.VarDecl != nil {
					declFuncs.VarDecl(gf, pkgDirUnix, v.Specs, v.Doc)
				}
			case token.IMPORT: // Ignore these
			default:
				panic(fmt.Sprintf("unrecognized token %s at: %s", v.Tok.String(), godb.WhereAt(v.Pos())))
			}
		}
	}
}

type phaseFunc func(*godb.GoFile, string, *File)

func phaseTypeDefs(gf *godb.GoFile, pkgDirUnix string, f *File) {
	processDecls(gf, pkgDirUnix, f, fileDeclFuncs{
		FuncDecl:  nil,
		TypeDecl:  declType,
		ConstDecl: nil,
		VarDecl:   nil,
	})
}

func phaseTypeRefs(gf *godb.GoFile, pkgDirUnix string, f *File) {
	processDecls(gf, pkgDirUnix, f, fileDeclFuncs{
		FuncDecl:  declFuncForTypes,
		TypeDecl:  declTypesForTypes,
		ConstDecl: declValueSpecForTypes,
		VarDecl:   declValueSpecForTypes,
	})
}

func phaseOtherDecls(gf *godb.GoFile, pkgDirUnix string, f *File) {
	processDecls(gf, pkgDirUnix, f, fileDeclFuncs{
		FuncDecl:  declFunc,
		TypeDecl:  nil,
		ConstDecl: declConstSpec,
		VarDecl:   declVarSpec,
	})
}

func processPackage(rootUnix, pkgDirUnix, _ string, p *Package, fn phaseFunc) {
	for path, f := range p.Files {
		goFilePathUnix := TrimPrefix(filepath.ToSlash(path), rootUnix+"/")
		gf := godb.GoFilesRelative[goFilePathUnix]
		if gf == nil {
			panic(fmt.Sprintf("cannot find GoFile object for %s", goFilePathUnix))
		}

		fn(gf, pkgDirUnix, f)
	}
}

func processDir(rootNative, pathNative paths.NativePath, nsRoot, importMeRoot string) error {
	pkgDirNative, ok := pathNative.RelativeTo(rootNative)
	if !ok {
		panic(fmt.Sprintf("%s is not relative to %s", pathNative, rootNative))
	}
	pkgDirUnix := pkgDirNative.ToUnix()
	if godb.Verbose {
		genutils.AddSortedStdout(fmt.Sprintf("Processing %s:\n", pkgDirUnix))
	}
	importMe := path.Join(importMeRoot, pkgDirUnix.String())

	pkgs, err := parser.ParseDir(godb.Fset, pathNative.String(),
		// Walk only *.go files that meet default (target) build constraints, e.g. per "// build ..."
		func(info os.FileInfo) bool {
			if HasSuffix(info.Name(), "_test.go") {
				if godb.Verbose {
					genutils.AddSortedStdout(fmt.Sprintf("Ignoring test code in %s\n", info.Name()))
				}
				return false
			}
			b, e := build.Default.MatchFile(pathNative.String(), info.Name())
			if godb.Verbose {
				genutils.AddSortedStdout(fmt.Sprintf("Matchfile(%s) => %v %v\n",
					pathNative.Join(info.Name()).ToUnix(),
					b, e))
			}
			return b && e == nil
		},
		parser.ParseComments|parser.AllErrors)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	found := false
	for pkgBaseName, pkg := range pkgs {
		// fmt.Println(pkg.ID, pkg.GoFiles)
		// info := &types.Info{}
		// checkedPkg, err := typesConf.Check("hey/what", godb.Fset, pkg.GoFiles, info)

		if pkgBaseName != pathNative.Base() {
			if godb.Verbose {
				genutils.AddSortedStdout(fmt.Sprintf("NOTICE: Package %s is defined in %s -- ignored due to name mismatch\n",
					pkgBaseName, pathNative))
			}
		} else {
			if found {
				panic("whaaa??")
			}
			// Cannot currently do this, as public constants generated via "_ Something = iota" are omitted:
			// FilterPackage(v, IsExported)
			godb.RegisterPackage(rootNative.ToUnix(), pkgDirUnix, nsRoot+ReplaceAll(pkgDirUnix.String(), "/", "."), importMe, pkg)
			found = true
		}
	}

	return nil
}

var excludeDirs = map[string]bool{
	"builtin":  true,
	"cmd":      true,
	"internal": true, // look into this later?
	"testdata": true,
	"vendor":   true,
}

func LegitimateImport(p string) bool {
	if p == "C" {
		return false
	}
	elements := Split(p, "/")
	for _, e := range elements {
		if excludeDirs[e] {
			return false
		}
	}
	return true
}

func walkDir(fsRoot paths.NativePath, nsRoot, importMeRoot string) error {
	target, err := fsRoot.EvalSymlinks()
	Check(err)

	err = target.Walk(
		func(path paths.NativePath, info os.FileInfo, err error) error {
			rel := ReplaceAll(path.String(), target.String(), fsRoot.String())
			relNative := paths.NewNativePath(rel)
			relUnix := relNative.ToUnix()
			if err != nil {
				genutils.EndSortedStdout()
				fmt.Fprintf(os.Stderr, "Skipping %s due to: %v\n", relUnix, err)
				return err
			}
			if relNative == fsRoot {
				return nil // skip (implicit) "."
			}
			if excludeDirs[relUnix.Base()] {
				if godb.Verbose {
					genutils.AddSortedStdout(fmt.Sprintf("Excluding %s\n", relUnix))
				}
				return paths.SkipDir
			}
			if info.IsDir() {
				return processDir(fsRoot, relNative, nsRoot, importMeRoot)
			}
			return nil // not a directory
		})

	if err != nil {
		genutils.EndSortedStdout()
		fmt.Fprintf(os.Stderr, "Error while walking %s: %v\n", fsRoot, err)
		return err
	}

	return err
}

type dirToWalk struct {
	srcDir   paths.NativePath
	fsRoot   paths.NativePath
	nsRoot   string
	importMe string
}

var dirsToWalk []dirToWalk

func AddWalkDir(srcDir, fsRoot paths.NativePath, nsRoot, importMe string) {
	dirsToWalk = append(dirsToWalk, dirToWalk{srcDir, fsRoot, nsRoot, importMe})
}

func myImporter(cfg *types.Config, info *types.Info, path string) (*types.Package, error) {
	// if path == "unsafe" {
	// 	return types.Unsafe, nil
	// }
	pkg := godb.GetPackagePackage(path)
	if pkg == nil {
		return nil, nil // TODO: Something better when package not found?
	}
	files := []*File{}
	for _, f := range pkg.Files {
		files = append(files, f)
	}
	return cfg.Check(path, godb.Fset, files, info)
}

func WalkAllDirs() (error, paths.NativePath) {
	var phases = []phaseFunc{
		phaseTypeDefs,
		phaseTypeRefs,
		phaseOtherDecls,
	}

	genutils.StartSortedStdout()
	defer func() {
		genutils.EndSortedStdout()
	}()

	for _, d := range dirsToWalk {
		err := walkDir(d.fsRoot, d.nsRoot, d.importMe)
		if err != nil {
			return err, d.srcDir
		}
	}

	cfg := &types.Config{
		IgnoreFuncBodies: true,
		FakeImportC:      true,
		Importer:         importer.Default(),
	}
	info := &types.Info{
		Types: map[Expr]types.TypeAndValue{},
		Defs:  map[*Ident]types.Object{},
	}
	astutils.TypeCheckerInfo = info

	for _, wp := range godb.PackagesAsDiscovered {
		pkg := wp.Dir.String()
		if _, err := myImporter(cfg, info, pkg); err != nil {
			fmt.Fprintf(os.Stderr, "walk.go/WalkAllDirs(): Failed to check %q: %s\n", pkg, err)
		}
		initPackage(wp.Dir.String(), wp.Pkg)
	}

	for _, phase := range phases {
		for _, wp := range godb.PackagesAsDiscovered {
			if true || wp.Pkg.Name != "unsafe" {
				processPackage(wp.Root.String(), wp.Dir.String(), wp.Namespace, wp.Pkg, phase)
			}
		}
	}

	return nil, paths.NewNativePath("")
}

func findApis(src paths.NativePath) (apis map[string]struct{}) {
	start := getCPU()
	defer func() {
		end := getCPU()
		if godb.Verbose && !noTimeAndVersion {
			fmt.Printf("findApis() took %d ns.\n", end-start)
		}
	}()

	apis = map[string]struct{}{}

	var fset = token.NewFileSet()

	target, err := src.ToNative().EvalSymlinks()
	Check(err)

	pkgs, err := parser.ParseDir(fset, target.String(), nil, 0)
	Check(err)

	var pkg *Package
	for k, v := range pkgs {
		if k != "core" {
			panic(fmt.Sprintf("Expected only package 'core', found '%s'", k))
		}
		pkg = v
	}

	for _, f := range pkg.Files {
		for _, d := range f.Decls {
			switch o := d.(type) {
			case *FuncDecl:
				if o.Recv == nil {
					if IsExported(o.Name.Name) {
						apis[o.Name.Name] = struct{}{}
					}
				}
			}
		}
	}

	return
}

// Determine the runtime API name for a function call, given a choice
// of prefixes (core and namespace-based) and the type name.  Ensure
// the resulting API has been code-generated or already exists in
// package core, wrapping it in an ABEND if not, and return the
// resulting wrap or, if no errors, the original string (which
// generate-std.joke will use to reconstitute the same API name).
func assertRuntime(prefix, nsPrefix, s string) string {
	runtime := s
	if ix := Index(s, "("); ix >= 0 {
		runtime = runtime[0:ix]
	}
	if ix := Index(runtime, "/"); ix >= 0 {
		ns := runtime[0 : ix+1]
		runtime = ns + nsPrefix + runtime[ix+1:]
	} else {
		runtime = prefix + runtime
	}
	if Contains(runtime, "ABEND") {
		return s
	}
	if _, found := definedApis[runtime]; !found {
		return fmt.Sprintf("ABEND707(API '%s' is unimplemented: %s)", runtime, s)
	}
	return s
}

// Determines (and validates) the API to call (in context), given the
// full Clojure typename (e.g. "go.std.something/Foo" or
// "arrayOfByte"), the import base name, and choice of prefixes.
func determineRuntime(prefix, nsPrefix, imp, clType string) string {
	var runtime, call string
	if ix := Index(clType, "/"); ix >= 0 {
		runtime = clType[0:ix+1] + nsPrefix + clType[ix+1:]
		if imp != "" {
			imp += "."
		}
		call = imp + nsPrefix + clType[ix+1:]
	} else {
		runtime = prefix + clType
		call = runtime
	}
	if Contains(runtime, "ABEND") {
		return runtime
	}
	if _, found := definedApis[runtime]; !found {
		return fmt.Sprintf("ABEND707(API '%s' is unimplemented: %s)", runtime, clType)
	}
	return call
}
