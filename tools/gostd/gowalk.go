package main

import (
	"fmt"
	. "go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var numFunctions int
var numStandalones int
var numReceivers int
var numTypes int
var numConstants int
var numVariables int
var numGeneratedFunctions int
var numGeneratedStandalones int
var numGeneratedReceivers int
var numGeneratedTypes int
var numGeneratedConstants int
var numGeneratedVariables int

type packageInfo struct {
	importsNative  *packageImports
	importsAutoGen *packageImports
	nonEmpty       bool // Whether any non-comment code has been generated
	hasGoFiles     bool // Whether any .go files (would) have been generated
}

/* Map (Unix-style) relative path to package info */
var packagesInfo = map[string]*packageInfo{}

/* Sort the packages -- currently appears to not actually be
/* necessary, probably because of how walkDirs() works. */
func sortedPackagesInfo(m map[string]*packageInfo, f func(k string, i *packageInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

func sortedPackageImports(pi packageImports, f func(k, local, full string)) {
	var keys []string
	for k, _ := range pi.fullNames {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := pi.fullNames[k]
		f(k, v.local, v.full)
	}
}

type goFile struct {
	name        string
	rootUnix    string
	pkgDirUnix  string
	pkgBaseName string
	spaces      *map[string]string // maps "foo" (in a reference such as "foo.Bar") to the pkgDirUnix in which it is defined
}

var goFiles = map[string]*goFile{}

type fnCodeInfo struct {
	sourceFile *goFile
	fnCode     string
}

type fnCodeMap map[string]fnCodeInfo

type codeInfo struct {
	constants goConstantsMap
	variables goVariablesMap
	functions fnCodeMap
	types     goTypeMap
	initTypes map[string]string            // func init() "GoTypes[key] = value"
	initVars  map[string]map[string]string // "var members_key1 = ... { key2: value, ... }"
}

/* Map relative (Unix-style) package names to maps of function names to code info and strings. */
var clojureCode = map[string]codeInfo{}
var goCode = map[string]codeInfo{}

func sortedPackageMap(m map[string]codeInfo, f func(k string, v codeInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

func sortedCodeMap(m codeInfo, f func(k string, v fnCodeInfo)) {
	var keys []string
	for k, _ := range m.functions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m.functions[k])
	}
}

type funcInfo struct {
	baseName     string // Just the name without receiver-type info
	receiverId   string // Receiver info (only one type supported here and by Golang itself for now)
	name         string // Unique name for implementation (has Receiver info as a prefix, then baseName)
	docName      string // Everything, for documentation and diagnostics
	fd           *FuncDecl
	sourceFile   *goFile
	refersToSelf bool // whether :go-imports should list itself
}

/* Go apparently doesn't support/allow 'interface{}' as the value (or
/* key) of a map such that any arbitrary type can be substituted at
/* run time, so there are several of these nearly-identical functions
/* sprinkled through this code. Still get some reuse out of some of
/* them, and it's still easier to maintain these copies than if the
/* body of these were to be included at each call point.... */
func sortedFuncInfoMap(m map[string]*funcInfo, f func(k string, v *funcInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

// Map qualified function names to info on each.
var qualifiedFunctions = map[string]*funcInfo{}

var alreadySeen = []string{}

func receiverPrefix(src *goFile, rl []fieldItem) string {
	res := ""
	for i, r := range rl {
		if i != 0 {
			res += "_"
		}
		switch x := r.field.Type.(type) {
		case *Ident:
			res += x.Name
		case *ArrayType:
			res += "ArrayOf_" + x.Elt.(*Ident).Name
		case *StarExpr:
			res += "PtrTo_" + x.X.(*Ident).Name
		default:
			panic(fmt.Sprintf("receiverList: unrecognized expr %T in %s", x, src.name))
		}
	}
	return res + "_"
}

func receiverId(src *goFile, pkgName string, rl []fieldItem) string {
	pkg := "_" + pkgName + "."
	res := ""
	for i, r := range rl {
		if i != 0 {
			res += "ABEND422(more than one receiver in list)"
		}
		switch x := r.field.Type.(type) {
		case *Ident:
			res += pkg + x.Name
		case *ArrayType:
			res += "[]" + pkg + x.Elt.(*Ident).Name
		case *StarExpr:
			res += "*" + pkg + x.X.(*Ident).Name
		default:
			panic(fmt.Sprintf("receiverId: unrecognized expr %T in %s", x, src.name))
		}
	}
	return res
}

// Returns whether any public functions were actually processed.
func processFuncDecl(gf *goFile, pkgDirUnix, filename string, f *File, fd *FuncDecl) bool {
	if dump {
		fmt.Printf("Func in pkgDirUnix=%s filename=%s:\n", pkgDirUnix, filename)
		Print(fset, fd)
	}
	fl := flattenFieldList(fd.Recv)
	fnName := receiverPrefix(gf, fl) + fd.Name.Name
	fullName := pkgDirUnix + "." + fnName
	if v, ok := qualifiedFunctions[fullName]; ok {
		alreadySeen = append(alreadySeen,
			fmt.Sprintf("NOTE: Already seen function %s in %s, yet again in %s",
				fullName, v.sourceFile.name, filename))
	}
	rcvrId := receiverId(gf, gf.pkgBaseName, fl)
	docName := "(" + receiverId(gf, pkgDirUnix, fl) + ")" + fd.Name.Name + "()"
	qualifiedFunctions[fullName] = &funcInfo{fd.Name.Name, rcvrId, fnName, docName, fd, gf, false}
	return true
}

func sortedTypeInfoMap(m map[string]*goTypeInfo, f func(k string, v *goTypeInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

// Maps qualified typename ("path/to/pkg.TypeName") to type info.
func processTypeSpec(gf *goFile, pkg string, ts *TypeSpec) bool {
	typename := pkg + "." + ts.Name.Name
	if dump {
		fmt.Printf("Type %s at %s:\n", typename, whereAt(ts.Pos()))
		Print(fset, ts)
	}
	if c, ok := goTypes[typename]; ok {
		fmt.Fprintf(os.Stderr, "WARNING: type %s found at %s and now again at %s\n",
			typename, whereAt(c.where), whereAt(ts.Pos()))
	}
	gt := registerType(gf, typename, ts)
	gt.td = ts
	gt.where = ts.Pos()
	gt.requiredImports = &packageImports{}
	if !isPrivate(ts.Name.Name) {
		numTypes++
	}
	return true
}

func processTypeSpecs(gf *goFile, pkg string, tss []Spec) (found bool) {
	for _, spec := range tss {
		ts := spec.(*TypeSpec)
		if processTypeSpec(gf, pkg, ts) {
			found = true
		}
	}
	return
}

type variableInfo struct {
	name       *Ident
	sourceFile *goFile
	def        string
}

type goVariablesMap map[string]*variableInfo

var goVariables = goVariablesMap{}

func sortedVariableInfoMap(m map[string]*variableInfo, f func(k string, v *variableInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

type constantInfo struct {
	name       *Ident
	sourceFile *goFile
	def        string
}

type goConstantsMap map[string]*constantInfo

var goConstants = goConstantsMap{}

func sortedConstantInfoMap(m map[string]*constantInfo, f func(k string, v *constantInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

func fitInt(value string) string {
	_, e := strconv.ParseInt(value, 0, 32)
	if e == nil {
		return "int"
	}
	_, e = strconv.ParseInt(value, 0, 64)
	if e == nil {
		return "int64"
	}
	_, e = strconv.ParseUint(value, 0, 64)
	if e == nil {
		return "uint64"
	}
	return ""
}

func evalConstType(ty *TypeSpec) (typeName string) {
	typeName = ty.Name.Name
	_, ok := goTypes[typeName]
	if !ok {
		typeName = ty.Type.(*Ident).Name
	}

	return
}

func evalConstExpr(val Expr) (typeName, result string) {
	switch v := val.(type) {
	case *BasicLit:
		result = v.Value
		switch v.Kind {
		case token.STRING:
			typeName = "string"
		case token.INT:
			typeName = fitInt(result)
		case token.FLOAT:
			typeName = "float64"
		case token.CHAR:
			typeName = "rune"
		}
	case *UnaryExpr:
		typeName, result = evalConstExpr(v.X)
		if result == "" && typeName == "" {
			typeName = "int" // TODO: maybe not, but try this for now
		}
		switch v.Op {
		case token.SUB:
			typeName, result = fitInt(result), "-"+result
		default:
		}
	case *BinaryExpr:
		leftType, _ := evalConstExpr(v.X)
		rightType, rightValue := evalConstExpr(v.Y)
		if leftType == rightType {
			typeName = leftType
		} else if leftType == "float64" || rightType == "float64" {
			typeName = "float64" // TODO: probably a good guess for now
		} else if leftType == "int64" || rightType == "int64" {
			typeName = "int64"
		} else if leftType == "rune" || rightType == "rune" {
			typeName = "int"
		}
		if typeName == "int" && v.Op == token.SHL {
			if rightValue == "64" { // TODO: this supports MaxUint64 but is overly specific
				typeName, result = "uint64", strconv.FormatUint(math.MaxUint64, 10)
			} else {
				typeName = "int64"
			}
		} else if typeName == "" {
			typeName = "uint64" // TODO: the outer MaxUint64 definition
		}
	case *ParenExpr:
		typeName, result = evalConstExpr(v.X)
	case *Ident:
		switch v.Name {
		case "iota":
			typeName, result = "int", "0"
		case "false", "true":
			typeName, result = "bool", v.Name
		}
		if v.Obj != nil {
			switch spec := v.Obj.Decl.(type) {
			case *ValueSpec:
				if len(spec.Values) == 0 {
					typeName, result = "int", "1" // TODO: probably a good guess for now
				} else {
					typeName, result = evalConstExpr(spec.Values[0])
				}
			case *TypeSpec:
				typeName = evalConstType(spec)
			}
		}
	case *CallExpr:
		typeName, result = evalConstExpr(v.Fun)
	}
	return
}

func determineConstExprType(val Expr) (typeName string) {
	switch v := val.(type) {
	case *BasicLit:
		switch v.Kind {
		case token.STRING:
			typeName = "string"
		case token.INT:
			typeName = fitInt(v.Value)
		case token.FLOAT:
			typeName = "float64"
		case token.CHAR:
			typeName = "rune"
		}
	default:
		typeName, _ = evalConstExpr(val)
	}
	return
}

func determineType(valType Expr, val Expr) (cl, gl string) {
	typeName := ""
	innerPromotion := "%s"
	if valType == nil {
		typeName = determineConstExprType(val)
	} else {
		ident, ok := valType.(*Ident)
		if !ok {
			return
		}
		valObj := ident.Obj
		if valObj != nil {
			if valObj.Decl != nil {
				ts, ok := valObj.Decl.(*TypeSpec)
				if !ok {
					return
				}
				if ts.Name == nil {
					return
				}
				if id, ok := ts.Type.(*Ident); ok {
					typeName = id.Name
				}
				innerPromotion = typeName + "(%s)"
			}
		} else {
			typeName = ident.Name
		}
	}
	if typeName == "" {
		return
	}
	gt, ok := goTypes[typeName]
	if !ok || gt.argClojureArgType == "" || gt.promoteType == "" {
		return "", ""
	}
	return gt.argClojureArgType, strings.ReplaceAll(gt.promoteType, "%s", innerPromotion)
}

func processConstantSpec(gf *goFile, pkg string, name *Ident, valType Expr, val Expr, docString string) bool {
	clName := name.Name
	localName := gf.pkgBaseName + "." + name.Name
	fullName := pkg + "." + name.Name

	if c, ok := goConstants[fullName]; ok {
		fmt.Fprintf(os.Stderr, "WARNING: constant %s found at %s and now again at %s\n",
			localName, whereAt(c.name.NamePos), whereAt(name.NamePos))
	}

	switch name.Name {
	case "Int", "String", "Boolean":
		clName += "-renamed" // TODO: is there a better solution possible?
	}

	valTypeString, promoteType := determineType(valType, val)
	if dump || (verbose && valTypeString == "") {
		fmt.Printf("Constant %s at %s:\n", name, whereAt(name.Pos()))
		if valType != nil {
			fmt.Printf("  valType at %s:\n", whereAt(valType.Pos()))
			Print(fset, valType)
		}
		if val != nil {
			fmt.Printf("  val at %s:\n", whereAt(val.Pos()))
			Print(fset, val)
		}
	}
	if valTypeString == "" {
		return false
	}

	goCode := fmt.Sprintf(promoteType, localName)

	def := fmt.Sprintf(`
(def
  ^{:doc %s
    :added "1.0"
    :tag %s
    :go "%s"}
  %s)
`,
		docString, valTypeString, goCode, clName)

	gt := &constantInfo{name, gf, def}
	goConstants[fullName] = gt
	numGeneratedConstants++

	return true
}

func processVariableSpec(gf *goFile, pkg string, name *Ident, valType Expr, val Expr, docString string) bool {
	clName := name.Name
	localName := gf.pkgBaseName + "." + name.Name
	fullName := pkg + "." + name.Name

	if c, ok := goVariables[fullName]; ok {
		fmt.Fprintf(os.Stderr, "WARNING: variable %s found at %s and now again at %s\n",
			localName, whereAt(c.name.NamePos), whereAt(name.NamePos))
	}

	if dump {
		fmt.Printf("Variable %s at %s:\n", name, whereAt(name.Pos()))
		if valType != nil {
			fmt.Printf("  valType at %s:\n", whereAt(valType.Pos()))
			Print(fset, valType)
		}
		if val != nil {
			fmt.Printf("  val at %s:\n", whereAt(val.Pos()))
			Print(fset, val)
		}
	}

	goCode := gf.pkgBaseName + "." + clName

	def := fmt.Sprintf(`
(def
  ^{:doc %s
    :added "1.0"
    :tag Var
    :go "%s"}
  %s)
`,
		docString, goCode, clName)

	gt := &variableInfo{name, gf, def}
	goVariables[fullName] = gt
	numGeneratedVariables++

	return true
}

func what(constant bool) string {
	if constant {
		return "Constant"
	}
	return "Variable"
}

func processValueSpecs(gf *goFile, pkg string, tss []Spec, parentDoc *CommentGroup, constant bool) (processed bool) {
	var previousVal, previousValType Expr
	for ix, spec := range tss {
		ts := spec.(*ValueSpec)
		for jx, valName := range ts.Names {
			valType := ts.Type
			var val Expr
			if ts.Values != nil {
				if jx >= len(ts.Values) {
					fmt.Printf("ts.Values index %d for %s out of range 0..%d:", jx, valName, len(ts.Values)-1)
					Print(fset, ts)
					continue
				}
				val = ts.Values[jx]
			}

			if val == nil {
				val = previousVal
			}
			if valType == nil {
				valType = previousValType
			}

			previousVal = val
			previousValType = valType

			if isPrivate(valName.Name) {
				continue
			}
			if constant {
				numConstants++
			} else {
				numVariables++
			}

			if dump {
				fmt.Printf("%s #%d of spec #%d %s at %s:\n", what(constant), jx, ix, valName, whereAt(valName.NamePos))
				if valType != nil {
					fmt.Printf("  valType:\n")
					Print(fset, valType)
				}
				if val != nil {
					fmt.Printf("  val:\n")
					Print(fset, val)
				}
			}
			doc := parentDoc
			if doc == nil {
				doc = ts.Doc
			}
			docString := commentGroupInQuotes(doc, "", "", "", "")
			if constant {
				if processConstantSpec(gf, pkg, valName, valType, val, docString) {
					processed = true
				}
			} else {
				if processVariableSpec(gf, pkg, valName, valType, val, docString) {
					processed = true
				}
			}
		}
	}
	return
}

func isPrivateType(f *Expr) bool {
	switch td := (*f).(type) {
	case *Ident:
		return isPrivate(td.Name)
	case *ArrayType:
		return isPrivateType(&td.Elt)
	case *StarExpr:
		return isPrivateType(&td.X)
	default:
		panic(fmt.Sprintf("unsupported expr type %T", f))
	}
}

// Returns whether any public declarations were actually processed.
func processDecls(gf *goFile, pkgDirUnix string, f *File) (processed bool) {
	for _, s := range f.Decls {
		switch v := s.(type) {
		case *FuncDecl:
		case *GenDecl:
			switch v.Tok {
			case token.TYPE:
				if processTypeSpecs(gf, pkgDirUnix, v.Specs) {
					processed = true
				}
			case token.CONST:
				if processValueSpecs(gf, pkgDirUnix, v.Specs, v.Doc, true) {
					processed = true
				}
			case token.VAR:
				if processValueSpecs(gf, pkgDirUnix, v.Specs, v.Doc, false) {
					processed = true
				}
			case token.IMPORT: // Ignore these
			default:
				panic(fmt.Sprintf("unrecognized token %s at: %s", v.Tok.String(), whereAt(v.Pos())))
			}
		default:
			panic(fmt.Sprintf("unrecognized Decl type %T at: %s", v, whereAt(v.Pos())))
		}
	}
	return
}

func processFuncs(gf *goFile, pkgDirUnix, pathUnix string, f *File) (processed bool) {
Funcs:
	for _, s := range f.Decls {
		switch v := s.(type) {
		case *FuncDecl:
			if isPrivate(v.Name.Name) {
				continue // Skipping non-exported functions
			}
			if v.Recv != nil {
				for _, r := range v.Recv.List {
					if isPrivateType(&r.Type) {
						continue Funcs // Publishable receivers must operate on public types
					}
				}
				numReceivers += 1
			} else {
				numStandalones += 1
			}
			numFunctions += 1
			if processFuncDecl(gf, pkgDirUnix, pathUnix, f, v) {
				processed = true
			}
		case *GenDecl:
		}
	}
	return
}

func processPackageMeta(rootUnix, pkgDirUnix, goFilePathUnix string, f *File) (gf *goFile) {
	if egf, found := goFiles[goFilePathUnix]; found {
		panic(fmt.Sprintf("Found %s twice -- now in %s, previously in %s!", goFilePathUnix, pkgDirUnix, egf.pkgDirUnix))
	}
	importsMap := map[string]string{}
	gf = &goFile{goFilePathUnix, rootUnix, pkgDirUnix, f.Name.Name, &importsMap}
	goFiles[goFilePathUnix] = gf

	for _, imp := range f.Imports {
		if dump {
			fmt.Printf("Import for file %s:\n", goFilePathUnix)
			Print(fset, imp)
		}
		importPath, err := strconv.Unquote(imp.Path.Value)
		check(err)
		var as string
		if n := imp.Name; n != nil {
			switch n.Name {
			case "_":
				continue // Ignore these
			case ".":
				fmt.Fprintf(os.Stderr, "ERROR: `.' not supported in import directive at %v\n", whereAt(n.NamePos))
				continue
			default:
				as = n.Name
			}
		} else {
			as = filepath.Base(importPath)
		}
		importsMap[as] = importPath
	}

	return
}

/* Represents an 'import ( foo "bar/bletch/foo" )' line to be produced. */
type packageImport struct {
	local    string // "foo", "_", ".", or empty
	localRef string // local unless empty, in which case final component of full (e.g. "foo")
	full     string // "bar/bletch/foo"
}

/* Maps relative package (unix-style) names to their imports, non-emptiness, etc. */
type packageImports struct {
	localNames map[string]string         // "foo" -> "bar/bletch/foo"; no "_" nor "." entries here
	fullNames  map[string]*packageImport // "bar/bletch/foo" -> ["foo", "bar/bletch/foo"]
}

/* Given desired local and the full (though relative) name of the
/* package, make sure the local name agrees with any existing entry
/* and isn't already used (someday picking an alternate local name if
/* necessary), add the mapping if necessary, and return the (possibly
/* alternate) local name. */
func addImport(packageImports *packageImports, local, full string, okToSubstitute bool) string {
	if e, found := packageImports.fullNames[full]; found {
		if e.local == local {
			return e.localRef
		}
		if okToSubstitute {
			return e.localRef
		}
		panic(fmt.Sprintf("addImport(%s,%s) trying to replace (%s,%s)", local, full, e.local, e.full))
	}
	localRef := local
	if local == "" {
		components := strings.Split(full, "/")
		localRef = components[len(components)-1]
	}
	if localRef != "." {
		if curFull, found := packageImports.localNames[localRef]; found {
			panic(fmt.Sprintf("addImport(%s,%s) trying to replace (%s,%s)", local, full, localRef, curFull))
		}
	}
	if packageImports.localNames == nil {
		packageImports.localNames = map[string]string{}
	}
	packageImports.localNames[localRef] = full
	if packageImports.fullNames == nil {
		packageImports.fullNames = map[string]*packageImport{}
	}
	packageImports.fullNames[full] = &packageImport{local, localRef, full}
	return localRef
}

func processPackage(rootUnix, pkgDirUnix string, p *Package) {
	if verbose {
		fmt.Printf("Processing package=%s:\n", pkgDirUnix)
	}

	found := false

	// Must process all types before processing functions, since receivers are defined on types.
	for path, f := range p.Files {
		goFilePathUnix := strings.TrimPrefix(filepath.ToSlash(path), rootUnix+"/")
		gf := processPackageMeta(rootUnix, pkgDirUnix, goFilePathUnix, f)
		if processDecls(gf, pkgDirUnix, f) {
			found = true
		}
	}

	// Now process functions.
	for path, f := range p.Files {
		goFilePathUnix := strings.TrimPrefix(filepath.ToSlash(path), rootUnix+"/")
		gf := goFiles[goFilePathUnix]
		if processFuncs(gf, pkgDirUnix, goFilePathUnix, f) {
			found = true
		}
	}

	if found {
		if _, ok := packagesInfo[pkgDirUnix]; !ok {
			packagesInfo[pkgDirUnix] = &packageInfo{&packageImports{}, &packageImports{}, false, false}
			goCode[pkgDirUnix] = codeInfo{goConstantsMap{}, goVariablesMap{}, fnCodeMap{}, goTypeMap{},
				map[string]string{}, map[string]map[string]string{}}
			clojureCode[pkgDirUnix] = codeInfo{goConstantsMap{}, goVariablesMap{}, fnCodeMap{}, goTypeMap{},
				map[string]string{}, map[string]map[string]string{}}
		}
	}
}

func processDir(root, rootUnix, path string, mode parser.Mode) error {
	pkgDir := strings.TrimPrefix(path, root+string(filepath.Separator))
	pkgDirUnix := filepath.ToSlash(pkgDir)
	if verbose {
		fmt.Printf("Processing %s:\n", pkgDirUnix)
	}

	pkgs, err := parser.ParseDir(fset, path,
		// Walk only *.go files that meet default (target) build constraints, e.g. per "// build ..."
		func(info os.FileInfo) bool {
			if strings.HasSuffix(info.Name(), "_test.go") {
				if verbose {
					fmt.Printf("Ignoring test code in %s\n", info.Name())
				}
				return false
			}
			b, e := build.Default.MatchFile(path, info.Name())
			if verbose {
				fmt.Printf("Matchfile(%s) => %v %v\n",
					filepath.ToSlash(filepath.Join(path, info.Name())),
					b, e)
			}
			return b && e == nil
		},
		mode)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	for pkgBaseName, v := range pkgs {
		if pkgBaseName != filepath.Base(path) {
			if verbose {
				fmt.Printf("NOTICE: Package %s is defined in %s -- ignored due to name mismatch\n",
					pkgBaseName, path)
			}
		} else if pkgBaseName == "unsafe" {
			if verbose {
				fmt.Printf("NOTICE: Ignoring package %s in %s\n", pkgBaseName, pkgDirUnix)
			}
		} else {
			processPackage(rootUnix, pkgDirUnix, v)
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

func walkDirs(root string, mode parser.Mode) error {
	rootUnix := filepath.ToSlash(root)
	target, err := filepath.EvalSymlinks(root)
	check(err)
	err = filepath.Walk(target,
		func(path string, info os.FileInfo, err error) error {
			rel := strings.Replace(path, target, root, 1)
			relUnix := filepath.ToSlash(rel)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Skipping %s due to: %v\n", relUnix, err)
				return err
			}
			if rel == root {
				return nil // skip (implicit) "."
			}
			if excludeDirs[filepath.Base(rel)] {
				if verbose {
					fmt.Printf("Excluding %s\n", relUnix)
				}
				return filepath.SkipDir
			}
			if info.IsDir() {
				if verbose {
					fmt.Printf("Walking from %s to %s\n", rootUnix, relUnix)
				}
				return processDir(root, rootUnix, rel, mode)
			}
			return nil // not a directory
		})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while walking %s: %v\n", root, err)
		return err
	}

	return err
}
