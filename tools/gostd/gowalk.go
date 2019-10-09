package main

import (
	"fmt"
	. "github.com/candid82/joker/tools/gostd/types"
	. "github.com/candid82/joker/tools/gostd/utils"
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
	pkg            *Package
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
	nsRoot      string             // "go.std." or whatever is desired as the root namespace
}

var goFiles = map[string]*goFile{}

type fnCodeInfo struct {
	sourceFile *goFile
	fnCode     string
	fnDecl     *FuncDecl     // Empty for standalone functions; used to get docstring for receivers
	fnDoc      *CommentGroup // for some reason, fnDecl.Doc disappears by the time we try to use it!!??
}

type fnCodeMap map[string]*fnCodeInfo

type codeInfo struct {
	constants goConstantsMap
	variables goVariablesMap
	functions fnCodeMap
	types     goTypeMap
	initTypes map[*TypeDefInfo]struct{}            // types to be initialized
	initVars  map[*TypeInfo]map[string]*fnCodeInfo // func initNative()'s "info_key1 = ... { key2: value, ... }"
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

func sortedCodeMap(m codeInfo, f func(k string, v *fnCodeInfo)) {
	var keys []string
	for k, _ := range m.functions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m.functions[k])
	}
}

func sortedFnCodeInfo(m map[string]*fnCodeInfo, f func(k string, v *fnCodeInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
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
		fmt.Printf("Func in pkgDirUnix=%s filename=%s fd=%p fd.Doc=%p:\n", pkgDirUnix, filename, fd, fd.Doc)
		Print(Fset, fd)
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
func processTypeSpec(gf *goFile, pkg string, ts *TypeSpec, parentDoc *CommentGroup) bool {
	typename := pkg + "." + ts.Name.Name
	if dump {
		fmt.Printf("Type %s at %s:\n", typename, whereAt(ts.Pos()))
		Print(Fset, ts)
	}

	ti := TypeDefine(ts, parentDoc)
	if c, ok := goTypes[typename]; ok {
		fmt.Fprintf(os.Stderr, "WARNING: type %s found at %s and now again at %s\n",
			typename, whereAt(c.where), whereAt(ts.Pos()))
	}
	clojureCode[pkg].initTypes[ti] = struct{}{}
	goCode[pkg].initTypes[ti] = struct{}{}

	gt := registerType(gf, typename, ts)
	gt.td = ts
	gt.where = ts.Pos()
	gt.requiredImports = &packageImports{}
	if !isPrivate(ts.Name.Name) {
		numTypes++
	}
	return true
}

func processTypeSpecs(gf *goFile, pkg string, tss []Spec, parentDoc *CommentGroup) (found bool) {
	for _, spec := range tss {
		ts := spec.(*TypeSpec)
		if processTypeSpec(gf, pkg, ts, parentDoc) {
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
		case "Errno": // TODO: another heuristic, for go.std.syscall only though
			typeName, result = "uintptr", "0"
		case "Signal": // TODO: another heuristic, for go.std.syscall only though
			typeName, result = "int16", "0" // int16 forces "int()" conversion, which Go requires of "type Signal int"!
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

func determineType(name string, valType, val Expr) (cl, gl string) {
	switch name {
	case "InvalidHandle": // TODO: uintptr on Windows; not found elsewhere
		return "Number", "uint64(%s)"
	}
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

// Constants are currently emitted while walking the packages. Unlike with variables, where the types are not needed,
// this code seemingly must determine the type of a constant so as to give the Clojure wrapper the appropriate type (and
// that is the straightforward way to handle this).
//
// In Go, constants can be explicitly typed, implicitly typed via the constant expressions to which they're assigned, or
// untyped via untyped constant expressions.
//
// Further, Go allows those expressions to refer to constants in other packages, to invoke constructors (say, for simple
// named types like "Type Foo Int") in other packages (as in "const x = Foo(22)", which gives x the type Foo and the
// value 22), and other such things.
//
// Since this code currently makes a complete determination of a constant's type during package walking, it can't count
// on being able to determine the type of anything in another package in order to infer the type that will be given to
// the constant.
//
// Even when all the info is available, this code currently does not attempt to properly evaluate a constant expression
// in order to assure that (for examples) "1 << 30" is "int", "1 << 31" could be "uint" (need to check that), "1 << 32"
// is "int64", "1 << 63" might be "uint64", and so on.
//
// Instead, this code uses some heuristics, including known names of things in Go 1.12's std library, to guess mostly
// correctly, erring on the side of being conservative (which usually means constants that could fit in an "Int" are
// instead a "BigInt").
//
// An attempt was made to change one constant to "variable style" in order to try to eliminate the need for determining
// the type info, via e.g.:
//
//   var EXFULL = syscall.EXFULL
//   var EXFULL_ *GoVar = &GoVar{Value: EXFULL}
//
// That yielded a GoVar[syscall.Errno] type that (int) couldn't convert (probably because it hasn't been special-cased
// to handle GoObject types).
//
// Changing that first line to
//
//   var EXFULL = int(syscall.EXFULL)
//
// solved the problem (EXFULL printed out as an integer, though couldn't be simply, say, added to another integer due to
// being a GoVar[int]), but obviously brings things back to needing to know the type.
//
// Going back to that first approach, and adding this (quick-kludge) code to the procInt function in procs.go:
//
//   	case GoObject:
//		return Int{I: int(obj.O.(syscall.Errno))}
//	case *GoVar:
//		return Int{I: int(obj.Value.(syscall.Errno))}
//
// This allowed "(int EXFULL)" (also "(int (deref EXFULL))", i.e. (int <GoObject[syscall.Errno]>)) to work in that it
// evaluates to a Clojure object of type "Int".
//
// Though the kludge (special-casing syscall.Errno) above can be automated away, it doesn't seem like having to always
// wrap such constants in a converter is a helpful requirement.
//
// This isn't just an issue with a named type wrapping a builtin type; even this didn't allow direct referencing of E as
// "Double":
//
//    var E = math.E
//    var E_ *GoVar = &GoVar{Value: E}
//
// It's possible (and perhaps desirable anyway?) that Joker could automatically cast (convert) all GoVar and GoObject
// values to their builtin equivalents, which might allow this all to make more sense.
//
// But it might actually be less work to move the determination of a constant's type to the code-generation phase (so it
// has access to all the packages on which constant expressions might depend) and fully evaluate constant expressions to
// faithfully determine not only their types, but their values as well, and just use those (so, no need to import
// dependent packages).
//
// There might even be an existing Go package to do some of the heavy lifting in that direction. In any case, the result
// would be a lot cleaner and clearer than having "constants" wrapped in GoObject's or GoVar's.
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

	valTypeString, promoteType := determineType(name.Name, valType, val)
	if dump || (verbose && valTypeString == "**FOO**") { // or "**FOO**" to quickly disable this
		fmt.Printf("Constant %s at %s:\n", name, whereAt(name.Pos()))
		if valType != nil {
			fmt.Printf("  valType at %s:\n", whereAt(valType.Pos()))
			Print(Fset, valType)
		}
		if val != nil {
			fmt.Printf("  val at %s:\n", whereAt(val.Pos()))
			Print(Fset, val)
		}
	}
	if valTypeString == "" {
		return false
	}

	goCode := fmt.Sprintf(promoteType, localName)

	// Note: :tag value is a string to avoid conflict with like-named member of namespace
	def := fmt.Sprintf(`
(def
  ^{:doc %s
    :added "1.0"
    :tag "%s"
    :go "%s"}
  %s)
`,
		docString, valTypeString, goCode, clName)

	gt := &constantInfo{name, gf, def}
	goConstants[fullName] = gt
	numGeneratedConstants++

	return true
}

// Note that the 'val' argument isn't used (except when dumping info)
// as it isn't needed to determine the type of a variable, since the
// type isn't needed for code generation for variables -- just for
// constants.
func processVariableSpec(gf *goFile, pkg string, name *Ident, valType Expr, val Expr, docString string) bool {
	clName := name.Name
	localName := gf.pkgBaseName + "." + name.Name
	fullName := pkg + "." + name.Name

	if c, ok := goVariables[fullName]; ok {
		fmt.Fprintf(os.Stderr, "WARNING: variable %s found at %s and now again at %s\n",
			localName, whereAt(c.name.NamePos), whereAt(name.NamePos))
	}

	switch name.Name {
	case "Int", "String", "Boolean":
		clName += "-renamed" // TODO: is there a better solution possible?
	}

	if dump {
		fmt.Printf("Variable %s at %s:\n", name, whereAt(name.Pos()))
		if valType != nil {
			fmt.Printf("  valType at %s:\n", whereAt(valType.Pos()))
			Print(Fset, valType)
		}
		if val != nil {
			fmt.Printf("  val at %s:\n", whereAt(val.Pos()))
			Print(Fset, val)
		}
	}

	// Note: :tag value is a string to avoid conflict with like-named member of namespace
	def := fmt.Sprintf(`
(def
  ^{:doc %s
    :added "1.0"
    :tag "Var"
    :go "%s"}
  %s)
`,
		docString, localName, clName)

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
					Print(Fset, valType)
				}
				if val != nil {
					fmt.Printf("  val:\n")
					Print(Fset, val)
				}
			}
			doc := ts.Doc // Try block comments for this specific decl
			if doc == nil {
				doc = ts.Comment // Use line comments if no preceding block comments are available
			}
			if doc == nil {
				doc = parentDoc // Use 'var'/'const' statement block comments as last resort
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
				if processTypeSpecs(gf, pkgDirUnix, v.Specs, v.Doc) {
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

func processPackageMeta(rootUnix, pkgDirUnix, goFilePathUnix, nsRoot string, f *File) (gf *goFile) {
	if egf, found := goFiles[goFilePathUnix]; found {
		panic(fmt.Sprintf("Found %s twice -- now in %s, previously in %s!", goFilePathUnix, pkgDirUnix, egf.pkgDirUnix))
	}
	importsMap := map[string]string{}
	gf = &goFile{goFilePathUnix, rootUnix, pkgDirUnix, f.Name.Name, &importsMap, nsRoot}
	goFiles[goFilePathUnix] = gf

	for _, imp := range f.Imports {
		if dump {
			fmt.Printf("Import for file %s:\n", goFilePathUnix)
			Print(Fset, imp)
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

func processPackage(rootUnix, pkgDirUnix, nsRoot string, p *Package) {
	if verbose {
		fmt.Printf("Processing package=%s:\n", pkgDirUnix)
	}

	if _, ok := packagesInfo[pkgDirUnix]; !ok {
		packagesInfo[pkgDirUnix] = &packageInfo{&packageImports{}, &packageImports{}, p, false, false}
		goCode[pkgDirUnix] = codeInfo{goConstantsMap{}, goVariablesMap{}, fnCodeMap{}, goTypeMap{},
			map[*TypeDefInfo]struct{}{}, map[*TypeInfo]map[string]*fnCodeInfo{}}
		clojureCode[pkgDirUnix] = codeInfo{goConstantsMap{}, goVariablesMap{}, fnCodeMap{}, goTypeMap{},
			map[*TypeDefInfo]struct{}{}, map[*TypeInfo]map[string]*fnCodeInfo{}}
	}

	found := false

	// Must process all types before processing functions, since receivers are defined on types.
	for path, f := range p.Files {
		goFilePathUnix := strings.TrimPrefix(filepath.ToSlash(path), rootUnix+"/")
		gf := processPackageMeta(rootUnix, pkgDirUnix, goFilePathUnix, nsRoot, f)
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

	if !found {
		return
	}
}

func processDir(root, rootUnix, path, nsRoot string, mode parser.Mode) error {
	pkgDir := strings.TrimPrefix(path, root+string(filepath.Separator))
	pkgDirUnix := filepath.ToSlash(pkgDir)
	if verbose {
		fmt.Printf("Processing %s:\n", pkgDirUnix)
	}

	pkgs, err := parser.ParseDir(Fset, path,
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
			processPackage(rootUnix, pkgDirUnix, nsRoot, v)
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

func walkDirs(fsRoot, nsRoot string, mode parser.Mode) error {
	rootUnix := filepath.ToSlash(fsRoot)
	target, err := filepath.EvalSymlinks(fsRoot)
	check(err)
	err = filepath.Walk(target,
		func(path string, info os.FileInfo, err error) error {
			rel := strings.Replace(path, target, fsRoot, 1)
			relUnix := filepath.ToSlash(rel)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Skipping %s due to: %v\n", relUnix, err)
				return err
			}
			if rel == fsRoot {
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
				return processDir(fsRoot, rootUnix, rel, nsRoot, mode)
			}
			return nil // not a directory
		})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while walking %s: %v\n", fsRoot, err)
		return err
	}

	return err
}
