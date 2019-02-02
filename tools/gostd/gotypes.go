package main

import (
	"fmt"
	. "go/ast"
	gotypes "go/types"
)

type goTypeInfo struct {
	typeName                  string          // empty (not a declared type) or the basic type name ("foo" for "x/y.foo")
	fullName                  string          // empty ("struct {...}" etc.), typeName (built-in), or path/to/pkg.typeName
	subType                   *Expr           // nil if not a declared type
	argClojureType            string          // Can convert this type to a Go function arg with my type
	argFromClojureObject      string          // Append this to Clojure object to extract value of my type
	argClojureArgType         string          // Clojure argument type for a Go function arg with my type
	argExtractFunc            string          // Call Extract<this>() for arg with my type
	convertFromClojure        string          // Pattern to convert a (scalar) %s to this type
	convertFromClojureImports []packageImport // Imports needed to support the above
	uncompleted               bool            // Has this type's info been filled in beyond the registration step?
	custom                    bool            // Is this not a builtin Go type?
	private                   bool            // Is this a private type?
	unsupported               bool            // Is this unsupported?
	constructs                bool            // Does the convertion from Clojure actually construct (via &sometype{}), returning ptr?
}

/* These map fullNames to type info. */
var goTypes = map[string]*goTypeInfo{}

func registerType(gf *goFile, fullTypeName string, ts *TypeSpec) {
	if _, found := goTypes[fullTypeName]; found {
		return
	}
	ti := &goTypeInfo{
		typeName:    ts.Name.Name,
		fullName:    fullTypeName,
		subType:     &ts.Type,
		private:     isPrivate(ts.Name.Name),
		custom:      true,
		uncompleted: true,
	}
	goTypes[fullTypeName] = ti
}

func toGoTypeNameInfo(pkgDirUnix, baseName string, e Expr) *goTypeInfo {
	if ti, found := goTypes[baseName]; found {
		return ti
	}
	fullName := pkgDirUnix + "." + baseName
	if ti, found := goTypes[fullName]; found {
		return ti
	}
	// Check whether type is builtin but not supported here. If so, register it accordingly and return it.
	if gotypes.Universe.Lookup(baseName) != nil {
		ti := &goTypeInfo{
			typeName:    baseName,
			fullName:    baseName,
			unsupported: true,
		}
		goTypes[fullName] = ti
		return ti
	}
	panic(fmt.Sprintf("type %s not found at %s", fullName, whereAt(e.Pos())))
}

func toGoTypeInfo(src *goFile, ts *TypeSpec) *goTypeInfo {
	return toGoExprInfo(src, ts.Type)
}

func toGoExprInfo(src *goFile, e Expr) *goTypeInfo {
	typeName := ""
	fullName := ""
	convertFromClojure := ""
	private := false
	var subType *Expr
	unsupported := false
	switch td := e.(type) {
	case *Ident:
		ti := toGoTypeNameInfo(src.pkgDirUnix, td.Name, e)
		if !ti.custom {
			return ti
		}
		if ti.uncompleted {
			// Fill in other info now that all types are registered.
			// Is type actually builtin by Go? See special code to check that and define it as unsupported if so.
			ti.uncompleted = false
		}
		return ti
	case *ArrayType:
		return goArrayType(src, td.Len, td.Elt)
	case *StarExpr:
		return goStarExpr(src, td.X)
	}
	if typeName == "" {
		typeName = fmt.Sprintf("ABEND047(codegen.go: unsupported type %T)", e)
		unsupported = true
	}
	v := &goTypeInfo{
		typeName:           typeName,
		fullName:           fullName,
		subType:            subType,
		private:            private,
		unsupported:        unsupported,
		convertFromClojure: convertFromClojure,
	}
	goTypes[v.fullName] = v
	return v
}

func toGoExprString(src *goFile, e Expr) string {
	t := toGoExprInfo(src, e)
	if t != nil {
		return t.fullName
	}
	return fmt.Sprintf("%T", e)
}

func goArrayType(src *goFile, len Expr, elt Expr) *goTypeInfo {
	var fullName string
	en := toGoExprString(src, elt)
	if len == nil {
		fullName = "[]" + en
	} else {
		fullName = "..." + en
	}
	if v, ok := goTypes[fullName]; ok {
		return v
	}
	v := &goTypeInfo{
		fullName: fullName,
	}
	goTypes[fullName] = v
	return v
}

func ptrTo(expr string) string {
	if expr[0] == '*' {
		return expr[1:]
	}
	return expr
}

func goStarExpr(src *goFile, x Expr) *goTypeInfo {
	ex := toGoExprInfo(src, x)
	fullName := "*" + ex.fullName
	if v, ok := goTypes[fullName]; ok {
		return v
	}
	typeName := "*" + ex.typeName
	convertFromClojure := ""
	if !ex.unsupported && ex.convertFromClojure != "" {
		convertFromClojure = ptrTo(ex.convertFromClojure)
	}
	v := &goTypeInfo{
		typeName:           typeName,
		fullName:           fullName,
		convertFromClojure: convertFromClojure,
	}
	goTypes[fullName] = v
	return v
}

func init() {
	goTypes["bool"] = &goTypeInfo{
		typeName:             "bool",
		fullName:             "bool",
		argClojureType:       "Boolean",
		argFromClojureObject: ".Boolean().B",
		argClojureArgType:    "Boolean",
		argExtractFunc:       "Boolean",
		convertFromClojure:   "ToBool(%s)",
	}
	goTypes["string"] = &goTypeInfo{
		typeName:             "string",
		fullName:             "string",
		argClojureType:       "String",
		argFromClojureObject: ".S",
		argClojureArgType:    "String",
		argExtractFunc:       "String",
		convertFromClojure:   `AssertString(%s, "").S`,
	}
	goTypes["rune"] = &goTypeInfo{
		typeName:             "rune",
		fullName:             "rune",
		argClojureType:       "Char",
		argFromClojureObject: ".Ch",
		argClojureArgType:    "Char",
		argExtractFunc:       "Char",
		convertFromClojure:   `AssertChar(%s, "").Ch`,
	}
	goTypes["byte"] = &goTypeInfo{
		typeName:             "byte",
		fullName:             "byte",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "Byte",
		convertFromClojure:   `byte(AssertInt(%s, "").I)`,
	}
	goTypes["int"] = &goTypeInfo{
		typeName:             "int",
		fullName:             "int",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "Int",
		convertFromClojure:   `AssertInt(%s, "").I`,
	}
	goTypes["uint"] = &goTypeInfo{
		typeName:             "uint",
		fullName:             "uint",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Number",
		argExtractFunc:       "UInt",
		convertFromClojure:   `uint(AssertInt(%s, "").I)`,
	}
	goTypes["int8"] = &goTypeInfo{
		typeName:             "int8",
		fullName:             "int8",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "Byte",
		convertFromClojure:   `int8(AssertInt(%s, "").I)`,
	}
	goTypes["uint8"] = &goTypeInfo{
		typeName:             "uint8",
		fullName:             "uint8",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "UInt8",
		convertFromClojure:   `uint8(AssertInt(%s, "").I)`,
	}
	goTypes["int16"] = &goTypeInfo{
		typeName:             "int16",
		fullName:             "int16",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "Int16",
		convertFromClojure:   `int16(AssertInt(%s, "").I)`,
	}
	goTypes["uint16x"] = &goTypeInfo{
		typeName:             "uint16",
		fullName:             "uint16",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "UInt16",
		convertFromClojure:   `uint16(AssertInt(%s, "").I)`,
	}
	goTypes["int32"] = &goTypeInfo{
		typeName:             "int32",
		fullName:             "int32",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Int",
		argExtractFunc:       "Int32",
		convertFromClojure:   `int32(AssertInt(%s, "").I)`,
	}
	goTypes["uint32"] = &goTypeInfo{
		typeName:             "uint32",
		fullName:             "uint32",
		argClojureType:       "Number",
		argFromClojureObject: ".Int().I",
		argClojureArgType:    "Number",
		argExtractFunc:       "UInt32",
		convertFromClojure:   `uint32(AssertNumber(%s, "").BigInt().Uint64())`,
	}
	goTypes["int64"] = &goTypeInfo{
		typeName:             "int64",
		fullName:             "int64",
		argClojureType:       "Number",
		argFromClojureObject: ".BigInt().Int64()",
		argClojureArgType:    "Number",
		argExtractFunc:       "Int64",
		convertFromClojure:   `AssertNumber(%s, "").BigInt().Int64()`,
	}
	goTypes["uint64"] = &goTypeInfo{
		typeName:             "uint64",
		fullName:             "uint64",
		argClojureType:       "Number",
		argFromClojureObject: ".BigInt().Uint64()",
		argClojureArgType:    "Number",
		argExtractFunc:       "UInt64",
		convertFromClojure:   `AssertNumber(%s, "").BigInt().Uint64()`,
	}
	goTypes["uintptr"] = &goTypeInfo{
		typeName:             "uintptr",
		fullName:             "uintptr",
		argClojureType:       "Number",
		argFromClojureObject: ".BigInt().Uint64()",
		argClojureArgType:    "Number",
		argExtractFunc:       "UIntPtr",
		convertFromClojure:   `uintptr(AssertNumber(%s, "").BigInt().Uint64())`,
	}
	goTypes["float32"] = &goTypeInfo{
		typeName:             "float32",
		fullName:             "float32",
		argClojureType:       "Double",
		argFromClojureObject: "",
		argClojureArgType:    "Double",
		argExtractFunc:       "ABEND007(find these)",
		convertFromClojure:   `float32(AssertDouble(%s, "").D)`,
	}
	goTypes["float64"] = &goTypeInfo{
		typeName:             "float64",
		fullName:             "float64",
		argClojureType:       "Double",
		argFromClojureObject: "",
		argClojureArgType:    "Double",
		argExtractFunc:       "ABEND007(find these)",
		convertFromClojure:   `float64(AssertDouble(%s, "").D)`,
	}
	goTypes["complex64"] = &goTypeInfo{
		typeName:             "complex64",
		fullName:             "complex64",
		argClojureType:       "",
		argFromClojureObject: "",
		argClojureArgType:    "",
		argExtractFunc:       "ABEND007(find these)",
		convertFromClojure:   "", // TODO: support this in Joker, even if via just [real imag]
	}
	goTypes["complex128"] = &goTypeInfo{
		typeName:             "complex128",
		fullName:             "complex128",
		argClojureType:       "",
		argFromClojureObject: "",
		argClojureArgType:    "",
		argExtractFunc:       "ABEND007(find these)",
		convertFromClojure:   "", // TODO: support this in Joker, even if via just [real imag]
	}
	goTypes["error"] = &goTypeInfo{
		typeName:                  "error",
		fullName:                  "error",
		argClojureType:            "Error",
		argFromClojureObject:      "",
		argClojureArgType:         "String",
		argExtractFunc:            "",
		convertFromClojure:        `_errors.New(AssertString(%s, "").S)`,
		convertFromClojureImports: []packageImport{{"_errors", "errors"}},
	}
}
