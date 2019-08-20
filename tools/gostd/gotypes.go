package main

import (
	"fmt"
	. "go/ast"
	"go/token"
	gotypes "go/types"
	"strings"
)

type goTypeInfo struct {
	goName                    string  // empty ("struct {...}" etc.), built-in name, path/to/pkg.name, or ABEND if unsupported
	fullClojureName           string  // Clojure version of goName
	sourceFile                *goFile // location of the type defintion
	td                        *TypeSpec
	where                     token.Pos
	underlyingType            *Expr           // nil if not a declared type
	argClojureType            string          // Can convert this type to a Go function arg with my type
	argFromClojureObject      string          // Append this to Clojure object to extract value of my type
	argClojureArgType         string          // Clojure argument type for a Go function arg with my type
	argExtractFunc            string          // Call Extract<this>() for arg with my type
	convertFromClojure        string          // Pattern to convert a (scalar) %s to this type
	convertFromClojureImports []packageImport // Imports needed to support the above
	convertToClojure          string          // Pattern to convert this type to an appropriate Clojure object
	clojureCode               string
	goCode                    string
	requiredImports           *packageImports
	uncompleted               bool // Has this type's info been filled in beyond the registration step?
	custom                    bool // Is this not a builtin Go type?
	private                   bool // Is this a private type?
	unsupported               bool // Is this unsupported?
	constructs                bool // Does the convertion from Clojure actually construct (via &sometype{}), returning ptr?
	nullable                  bool // Can an instance of the type == nil (e.g. 'error' type)?
}

type goTypeMap map[string]*goTypeInfo

/* These map goNames to type info. */
var goTypes = goTypeMap{}

func goTypeName(src *goFile, e *Expr) string {
	goName := ""
	switch compound := (*e).(type) {
	case *ArrayType:
		goName = "[]"
		e = &(*compound).Elt
	case *StarExpr:
		goName = "*"
		e = &(*compound).X
	}
	switch x := (*e).(type) {
	case *SelectorExpr:
		goName += x.X.(*Ident).Name + "." + x.Sel.Name
	case *Ident:
		baseName := x.Name
		if gotypes.Universe.Lookup(baseName) == nil { // builtin
			goName += src.pkgDirUnix + "." + baseName
		} else {
			goName += x.Name
		}
	case *InterfaceType:
		goName += "interface{}"
	case *MapType:
		goName += "map[" + goTypeName(src, &x.Key) + "]" + goTypeName(src, &x.Value)
	default:
		goName += fmt.Sprintf("%T", x)
	}
	return goName
}

func lookupGoType(src *goFile, e *Expr) *goTypeInfo {
	goName := goTypeName(src, e)
	if ti, found := goTypes[goName]; found {
		return ti
	}
	if gotypes.Universe.Lookup(goName) != nil {
		ti := &goTypeInfo{
			goName:             fmt.Sprintf("ABEND046(gotypes.go: unsupported builtin type %s)", goName),
			fullClojureName:    fullTypeNameAsClojure(goName),
			argClojureType:     goName,
			argClojureArgType:  goName,
			convertFromClojure: goName + "(%s)",
			convertToClojure:   "GoObject(%s%s)",
			unsupported:        true,
		}
		goTypes[goName] = ti
		return ti
	}
	panic(fmt.Sprintf("type %s not found at %s", goName, whereAt((*e).Pos())))
}

func registerType(gf *goFile, fullGoTypeName string, ts *TypeSpec) *goTypeInfo {
	if ti, found := goTypes[fullGoTypeName]; found {
		return ti
	}
	ti := &goTypeInfo{
		goName:            fullGoTypeName,
		fullClojureName:   fullTypeNameAsClojure(fullGoTypeName),
		sourceFile:        gf,
		underlyingType:    &ts.Type,
		argClojureArgType: fullTypeNameAsClojure(fullGoTypeName),
		private:           isPrivate(ts.Name.Name),
		custom:            true,
		uncompleted:       true,
		convertToClojure:  "GoObject(%s%s)",
		argExtractFunc:    "Object",
	}
	goTypes[fullGoTypeName] = ti
	return ti
}

func toGoTypeNameInfo(pkgDirUnix, baseName string, e *Expr) *goTypeInfo {
	if ti, found := goTypes[baseName]; found {
		return ti
	}
	goName := pkgDirUnix + "." + baseName
	if ti, found := goTypes[goName]; found {
		return ti
	}
	if gotypes.Universe.Lookup(baseName) != nil {
		ti := &goTypeInfo{
			goName:             fmt.Sprintf("ABEND046(gotypes.go: unsupported builtin type %s for %s)", baseName, pkgDirUnix),
			fullClojureName:    fullTypeNameAsClojure(goName),
			argClojureType:     baseName,
			argClojureArgType:  baseName,
			convertFromClojure: baseName + "(%s)",
			convertToClojure:   "GoObject(%s%s)",
			unsupported:        true,
		}
		goTypes[baseName] = ti
		return ti
	}
	panic(fmt.Sprintf("type %s not found at %s", goName, whereAt((*e).Pos())))
}

func toGoTypeInfo(src *goFile, ts *TypeSpec) *goTypeInfo {
	return toGoExprInfo(src, &ts.Type)
}

func toGoExprInfo(src *goFile, e *Expr) *goTypeInfo {
	convertFromClojure := ""
	private := false
	var underlyingType *Expr
	unsupported := false
	switch td := (*e).(type) {
	case *Ident:
		ti := toGoTypeNameInfo(src.pkgDirUnix, td.Name, e)
		if ti.uncompleted {
			// Fill in other info now that all types are registered.
			ut := toGoExprInfo(src, ti.underlyingType)
			if ut.convertFromClojure != "" {
				if ut.constructs {
					ti.convertFromClojure = "*" + ut.convertFromClojure
				} else {
					ti.convertFromClojure = fmt.Sprintf("_%s.%s(%s)", ti.sourceFile.pkgBaseName, ti.goBaseName(), ut.convertFromClojure)
				}
			}
			ti.convertFromClojureImports = ut.convertFromClojureImports
			ti.uncompleted = false
		}
		return ti
	case *ArrayType:
		return goArrayType(src, &td.Len, &td.Elt)
	case *StarExpr:
		return goStarExpr(src, &td.X)
	case *SelectorExpr:
		return goSelectorExpr(src, td)
	}
	goName := fmt.Sprintf("ABEND047(gotypes.go: unsupported type %T at: %s)", *e, unix(whereAt((*e).Pos())))
	unsupported = true
	v := &goTypeInfo{
		goName:             goName,
		fullClojureName:    fullTypeNameAsClojure(goName),
		underlyingType:     underlyingType,
		private:            private,
		unsupported:        unsupported,
		convertFromClojure: convertFromClojure,
		convertToClojure:   "GoObject(%s%s)",
	}
	goTypes[goName] = v
	return v
}

// Either the builtin name or the full package name and type name
// (e.g. "go.std.net/IPAddr"), possibly preceded by (say) "*" for a
// pointer to it.
func (t *goTypeInfo) goFullName() string {
	return t.goName
}

func (t *goTypeInfo) goBaseName() string {
	strs := strings.SplitAfter(t.goFullName(), ".")
	return strs[len(strs)-1]
}

func toGoExprString(src *goFile, e *Expr) string {
	if e == nil {
		return "-"
	}
	t := toGoExprInfo(src, e)
	if t != nil {
		return t.goName
	}
	return fmt.Sprintf("%T", e)
}

func lenString(len *Expr) string {
	if len == nil || *len == nil {
		return ""
	}
	l := *len
	switch n := l.(type) {
	case *Ident:
		return n.Name
	case *BasicLit:
		return n.Value
	}
	return fmt.Sprintf("%T", l)
}

func goArrayType(src *goFile, len *Expr, elt *Expr) *goTypeInfo {
	var goName string
	e := toGoExprInfo(src, elt)
	goName = "[" + lenString(len) + "]" + e.goName
	if v, ok := goTypes[goName]; ok {
		return v
	}
	v := &goTypeInfo{
		goName:           goName,
		fullClojureName:  "GoObject",
		underlyingType:   elt,
		custom:           true,
		unsupported:      e.unsupported,
		constructs:       e.constructs,
		convertToClojure: "GoObject(%s%s)",
	}
	goTypes[goName] = v
	return v
}

func ptrTo(expr string) string {
	if expr[0] == '*' {
		return expr[1:]
	}
	return expr
}

func goStarExpr(src *goFile, x *Expr) *goTypeInfo {
	e := toGoExprInfo(src, x)
	goName := "*" + e.goName
	if v, ok := goTypes[goName]; ok {
		return v
	}
	convertFromClojure := ""
	if e.convertFromClojure != "" {
		if e.constructs {
			convertFromClojure = e.convertFromClojure
		} else if e.argClojureArgType == e.argExtractFunc {
			/* Not a conversion, so can take address of the Clojure object's internals. */
			convertFromClojure = "&" + e.convertFromClojure
		}
	}
	v := &goTypeInfo{
		goName:             goName,
		fullClojureName:    "GoObject",
		underlyingType:     x,
		convertFromClojure: convertFromClojure,
		custom:             true,
		private:            e.private,
		unsupported:        e.unsupported,
		convertToClojure:   "GoObject(%s%s)",
	}
	goTypes[goName] = v
	return v
}

func goSelectorExpr(src *goFile, e *SelectorExpr) *goTypeInfo {
	pkgName := e.X.(*Ident).Name
	fullPathUnix := unix(fileAt(e.Pos()))
	referringFile := strings.TrimPrefix(fullPathUnix, src.rootUnix+"/")
	rf, ok := goFiles[referringFile]
	if !ok {
		panic(fmt.Sprintf("goSelectorExpr: could not find referring file %s for file %s at %s",
			referringFile, fullPathUnix, whereAt(e.Pos())))
	}
	fullPkgName, found := (*rf.spaces)[pkgName]
	if !found {
		panic(fmt.Sprintf("processing %s: could not find %s in %s",
			whereAt(e.Pos()), pkgName, src.name))
	}

	clType, _, convertFromClojure, _ := fullPkgNameAsGoType(src, fullPkgName, (*e).Sel.Name)
	v := &goTypeInfo{
		goName:             clType,
		fullClojureName:    "GoObject",
		underlyingType:     &e.X,
		convertFromClojure: convertFromClojure,
		custom:             true,
		private:            false, // TODO: look into doing this better
		unsupported:        false, // TODO: look into doing this better
		convertToClojure:   "GoObject(%s%s)",
	}
	// TODO: memoize?
	return v
}

func init() {
	goTypes["bool"] = &goTypeInfo{
		goName:               "bool",
		fullClojureName:      "Boolean",
		argClojureType:       "Boolean",
		argFromClojureObject: ".B",
		argClojureArgType:    "Boolean",
		argExtractFunc:       "Boolean",
		convertFromClojure:   "ToBool(%s)",
		convertToClojure:     "Boolean(%s%s)",
	}
	goTypes["string"] = &goTypeInfo{
		goName:               "string",
		fullClojureName:      "String",
		argClojureType:       "String",
		argFromClojureObject: ".S",
		argClojureArgType:    "String",
		argExtractFunc:       "String",
		convertFromClojure:   `AssertString(%s, "").S`,
		convertToClojure:     "String(%s%s)",
	}
	goTypes["rune"] = &goTypeInfo{
		goName:               "rune",
		fullClojureName:      "Char",
		argClojureType:       "Char",
		argFromClojureObject: ".Ch",
		argClojureArgType:    "Char",
		argExtractFunc:       "Char",
		convertFromClojure:   `AssertChar(%s, "").Ch`,
		convertToClojure:     "Char(%s%s)",
	}
	goTypes["byte"] = &goTypeInfo{
		goName:               "byte",
		fullClojureName:      "Int",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "Byte",
		convertFromClojure:   `byte(AssertInt(%s, "").I)`,
		convertToClojure:     "Int(int(%s)%s)",
	}
	goTypes["int"] = &goTypeInfo{
		goName:               "int",
		fullClojureName:      "Int",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "Int",
		convertFromClojure:   `AssertInt(%s, "").I`,
		convertToClojure:     "Int(%s%s)",
	}
	goTypes["uint"] = &goTypeInfo{
		goName:               "uint",
		fullClojureName:      "Number",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Number",
		argExtractFunc:       "UInt",
		convertFromClojure:   `uint(AssertInt(%s, "").I)`,
		convertToClojure:     "BigIntU(uint64(%s)%s)",
	}
	goTypes["int8"] = &goTypeInfo{
		goName:               "int8",
		fullClojureName:      "Int",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "Byte",
		convertFromClojure:   `int8(AssertInt(%s, "").I)`,
		convertToClojure:     "Int(int(%s)%s)",
	}
	goTypes["uint8"] = &goTypeInfo{
		goName:               "uint8",
		fullClojureName:      "Int",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "UInt8",
		convertFromClojure:   `uint8(AssertInt(%s, "").I)`,
		convertToClojure:     "Int(int(%s)%s)",
	}
	goTypes["int16"] = &goTypeInfo{
		goName:               "int16",
		fullClojureName:      "Int",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "Int16",
		convertFromClojure:   `int16(AssertInt(%s, "").I)`,
		convertToClojure:     "Int(int(%s)%s)",
	}
	goTypes["uint16"] = &goTypeInfo{
		goName:               "uint16",
		fullClojureName:      "Int",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "UInt16",
		convertFromClojure:   `uint16(AssertInt(%s, "").I)`,
		convertToClojure:     "Int(int(%s)%s)",
	}
	goTypes["int32"] = &goTypeInfo{
		goName:               "int32",
		fullClojureName:      "Int",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "Int32",
		convertFromClojure:   `int32(AssertInt(%s, "").I)`,
		convertToClojure:     "Int(int(%s)%s)",
	}
	goTypes["uint32"] = &goTypeInfo{
		goName:               "uint32",
		fullClojureName:      "Number",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Number",
		argExtractFunc:       "UInt32",
		convertFromClojure:   `uint32(AssertNumber(%s, "").BigInt().Uint64())`,
		convertToClojure:     "BigIntU(uint64(%s)%s)",
	}
	goTypes["int64"] = &goTypeInfo{
		goName:               "int64",
		fullClojureName:      "Number",
		argClojureType:       "Number",
		argFromClojureObject: ".BigInt().Int64()",
		argClojureArgType:    "Number",
		argExtractFunc:       "Int64",
		convertFromClojure:   `AssertNumber(%s, "").BigInt().Int64()`,
		convertToClojure:     "BigInt(%s%s)",
	}
	goTypes["uint64"] = &goTypeInfo{
		goName:               "uint64",
		fullClojureName:      "Number",
		argClojureType:       "Number",
		argFromClojureObject: ".BigInt().Uint64()",
		argClojureArgType:    "Number",
		argExtractFunc:       "UInt64",
		convertFromClojure:   `AssertNumber(%s, "").BigInt().Uint64()`,
		convertToClojure:     "BigIntU(%s%s)",
	}
	goTypes["uintptr"] = &goTypeInfo{
		goName:               "uintptr",
		fullClojureName:      "Number",
		argClojureType:       "Number",
		argFromClojureObject: ".BigInt().Uint64()",
		argClojureArgType:    "Number",
		argExtractFunc:       "UIntPtr",
		convertFromClojure:   `uintptr(AssertNumber(%s, "").BigInt().Uint64())`,
	}
	goTypes["float32"] = &goTypeInfo{
		goName:               "float32",
		fullClojureName:      "Double",
		argClojureType:       "Double",
		argFromClojureObject: "",
		argClojureArgType:    "Double",
		argExtractFunc:       "ABEND007(find these)",
		convertFromClojure:   `float32(AssertDouble(%s, "").D)`,
	}
	goTypes["float64"] = &goTypeInfo{
		goName:               "float64",
		fullClojureName:      "Double",
		argClojureType:       "Double",
		argFromClojureObject: "",
		argClojureArgType:    "Double",
		argExtractFunc:       "ABEND007(find these)",
		convertFromClojure:   `float64(AssertDouble(%s, "").D)`,
	}
	goTypes["complex64"] = &goTypeInfo{
		goName:               "complex64",
		fullClojureName:      "",
		argClojureType:       "",
		argFromClojureObject: "",
		argClojureArgType:    "",
		argExtractFunc:       "ABEND007(find these)",
		convertFromClojure:   "", // TODO: support this in Joker, even if via just [real imag]
	}
	goTypes["complex128"] = &goTypeInfo{
		goName:               "complex128",
		fullClojureName:      "",
		argClojureType:       "",
		argFromClojureObject: "",
		argClojureArgType:    "",
		argExtractFunc:       "ABEND007(find these)",
		convertFromClojure:   "", // TODO: support this in Joker, even if via just [real imag]
	}
	goTypes["error"] = &goTypeInfo{
		goName:                    "error",
		fullClojureName:           "Error",
		argClojureType:            "Error",
		argFromClojureObject:      "",
		argClojureArgType:         "Error",
		argExtractFunc:            "Error",
		convertFromClojure:        `_errors.New(AssertString(%s, "").S)`,
		convertFromClojureImports: []packageImport{{"_errors", "errors"}},
		convertToClojure:          "Error(%s%s)",
		nullable:                  true,
	}
	goTypes["interface{}"] = &goTypeInfo{
		goName:               "interface{}",
		fullClojureName:      "",
		argClojureType:       "",
		argFromClojureObject: "",
		argClojureArgType:    "",
		argExtractFunc:       "",
		convertFromClojure:   "",
		convertToClojure:     "",
	}
}
