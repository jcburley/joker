package jtypes

import (
	"fmt"
	. "github.com/candid82/joker/tools/gostd/godb"
	. "go/ast"
)

// Info on Joker types, including map of Joker type names to said type
// info.  A Joker type name is either unqualified (built-in, not
// namespace-rooted) or fully qualified by a namespace name
// (e.g. "go.std.example/SomeType").
type Info struct {
	Expr                 Expr   // [key] The canonical referencing expression (if any)
	FullName             string // [key] Full name of type as a Joker expression
	Who                  string // who made me
	Pattern              string // E.g. "%s", "refTo%s" (for reference types), "arrayOf%s" (for array types)
	Namespace            string // E.g. "go.std.net.url", in which this type resides ("" denotes global namespace)
	BaseName             string // E.g. "Listener"
	ArgClojureType       string // Can convert this type to a Go function arg with my type
	ArgFromClojureObject string // Append this to Clojure object to extract value of my type
	ArgExtractFunc       string
	ArgClojureArgType    string // Clojure argument type for a Go function arg with my type
	ConvertFromClojure   string // Pattern to convert a (scalar) %s to this type
	ConvertFromMap       string // Pattern to convert a map %s key %s to this type
	FullNameDoc          string // Full name of type as a Joker expression (for documentation); just e.g. "Int" for builtin global types
	AsJokerObject        string // Pattern to convert this type to a normal Joker type; empty string means wrap in a GoObject
	PromoteType          string // Pattern to promote to a canonical type (used by constant evaluation)
	IsUnsupported        bool   // Is this unsupported?
}

// Maps type-defining Expr or Joker type names (with or without
// "<ns>/" prefixes, depending on globality) to exactly one struct
// describing that type.
var typesByExpr = map[Expr]*Info{}
var typesByFullname = map[string]*Info{}

func combine(ns, name string) string {
	if ns == "" {
		return name
	}
	return ns + "/" + name
}

func patternForExpr(e Expr) (pattern string, ue Expr) {
	switch v := e.(type) {
	case *ArrayType:
		pattern, e = patternForExpr(v.Elt)
		return "arrayOf" + pattern, e
	case *StarExpr:
		pattern, e = patternForExpr(v.X)
		return "refTo" + pattern, e
	default:
		return "%s", e
	}
}

func namingForExpr(e Expr) (pattern, ns, baseName, name string, info *Info) {

	var ue Expr
	pattern, ue = patternForExpr(e)

	switch v := ue.(type) {
	case *Ident:
		if !IsBuiltin(v.Name) {
			ns = ClojureNamespaceForExpr(ue)
			baseName = v.Name
		} else {
			uInfo, found := goTypeMap[v.Name]
			if !found {
				panic(fmt.Sprintf("no type info for builtin `%s'", v.Name))
			}
			baseName = uInfo.FullNameDoc
			if e == ue {
				info = uInfo
			}
		}
	case *SelectorExpr:
		pkgName := v.X.(*Ident).Name
		ns = ClojureNamespaceForGoFile(pkgName, GoFileForExpr(v))
		baseName = v.Sel.Name
	default:
		baseName = "GoObject"
	}

	name = combine(ns, fmt.Sprintf(pattern, baseName))

	//	fmt.Printf("jtypes.go/typeNameForExpr: %s (`%s' %s %s) %+v => `%s' %T at:%s\n", name, pattern, ns, baseName, e, pattern, ue, WhereAt(e.Pos()))

	return
}

func Define(ts *TypeSpec, varExpr Expr) *Info {

	ns := ClojureNamespaceForPos(Fset.Position(ts.Name.NamePos))

	pattern, _ := patternForExpr(varExpr)

	name := combine(ns, fmt.Sprintf(pattern, ts.Name.Name))

	jti := &Info{
		FullName:          name,
		Who:               "TypeDefine",
		Pattern:           pattern,
		Namespace:         ns,
		BaseName:          ts.Name.Name,
		ArgExtractFunc:    "Object",
		ArgClojureArgType: name,
		FullNameDoc:       name,
		AsJokerObject:     "GoObject(%s%s)",
	}

	jti.register()

	return jti

}

func InfoForGoName(fullName string) *Info {
	return goTypeMap[fullName]
}

func InfoForExpr(e Expr) *Info {
	if info, ok := typesByExpr[e]; ok {
		return info
	}

	pattern, ns, baseName, fullName, info := namingForExpr(e)

	if info != nil {
		// Already found info on builtin Go type, so just return that.
		typesByExpr[e] = info
		return info
	}

	if info, found := typesByFullname[fullName]; found {
		typesByExpr[e] = info
		return info
	}

	convertFromClojure, convertFromMap := ConversionsFn(e)

	info = &Info{
		Expr:               e,
		FullName:           fullName,
		Who:                fmt.Sprintf("TypeForExpr %T", e),
		Pattern:            pattern,
		Namespace:          ns,
		BaseName:           baseName,
		FullNameDoc:        fullName,
		ArgClojureArgType:  fullName,
		ConvertFromClojure: convertFromClojure,
		ConvertFromMap:     convertFromMap,
	}

	typesByExpr[e] = info
	typesByFullname[fullName] = info

	return info
}

func (ti *Info) NameDoc(e Expr) string {
	if ti.Pattern == "" || ti.Namespace == "" {
		return ti.FullName
	}
	if e != nil && ClojureNamespaceForExpr(e) != ti.Namespace {
		return ti.FullName
	}
	return fmt.Sprintf(ti.Pattern, ti.BaseName)
}

func (ti *Info) register() {
	if _, found := typesByFullname[ti.FullName]; !found {
		typesByFullname[ti.FullName] = ti
	}
}

var Nil = &Info{}

var Error = &Info{
	FullName:             "Error",
	ArgClojureType:       "Error",
	ArgFromClojureObject: "",
	ArgExtractFunc:       "Error",
	ArgClojureArgType:    "Error",
	ConvertFromMap:       `FieldAsError(%s, %s)`,
	FullNameDoc:          "Error",
	AsJokerObject:        "Error(%s%s)",
	PromoteType:          "%s",
}

var Boolean = &Info{
	FullName:             "Boolean",
	ArgClojureType:       "Boolean",
	ArgFromClojureObject: ".B",
	ArgExtractFunc:       "Boolean",
	ArgClojureArgType:    "Boolean",
	ConvertFromMap:       "FieldAsBoolean(%s, %s)",
	FullNameDoc:          "Boolean",
	AsJokerObject:        "Boolean(%s%s)",
	PromoteType:          "%s",
}

var Byte = &Info{
	FullName:             "Byte",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Byte",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsByte(%s, %s)`,
	FullNameDoc:          "Byte",
	AsJokerObject:        "Int(int(%s)%s)",
	PromoteType:          "int(%s)",
}

var Rune = &Info{
	FullName:             "Char",
	ArgClojureType:       "Char",
	ArgFromClojureObject: ".Ch",
	ArgExtractFunc:       "Char",
	ArgClojureArgType:    "Char",
	ConvertFromMap:       `FieldAsChar(%s, %s)`,
	FullNameDoc:          "Char",
	AsJokerObject:        "Char(%s%s)",
	PromoteType:          "%s",
}

var String = &Info{
	FullName:             "String",
	ArgClojureType:       "String",
	ArgFromClojureObject: ".S",
	ArgExtractFunc:       "String",
	ArgClojureArgType:    "String",
	ConvertFromMap:       `FieldAsString(%s, %s)`,
	FullNameDoc:          "String",
	AsJokerObject:        "String(%s%s)",
	PromoteType:          "%s",
}

var Int = &Info{
	FullName:             "Int",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Int",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsInt(%s, %s)`,
	FullNameDoc:          "Int",
	AsJokerObject:        "Int(%s%s)",
	PromoteType:          "%s",
}

var Int8 = &Info{
	FullName:             "Int",
	ArgClojureType:       "Int",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Int8",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsInt8(%s, %s)`,
	FullNameDoc:          "Int",
	AsJokerObject:        "Int(int(%s)%s)",
	PromoteType:          "int(%s)",
}

var Int16 = &Info{
	FullName:             "Int",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Int16",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsInt16(%s, %s)`,
	FullNameDoc:          "Int",
	AsJokerObject:        "Int(int(%s)%s)",
	PromoteType:          "int(%s)",
}

var Int32 = &Info{
	FullName:             "Int",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Int32",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsInt32(%s, %s)`,
	FullNameDoc:          "Int",
	AsJokerObject:        "Int(int(%s)%s)",
	PromoteType:          "int(%s)",
}

var Int64 = &Info{
	FullName:             "BigInt",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".BigInt().Int64()",
	ArgExtractFunc:       "Int64",
	ArgClojureArgType:    "BigInt",
	ConvertFromMap:       `FieldAsInt64(%s, %s)`,
	FullNameDoc:          "BigInt",
	AsJokerObject:        "BigInt(%s%s)",
	PromoteType:          "int64(%s)",
}

var UInt = &Info{
	FullName:             "Number",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Uint",
	ArgClojureArgType:    "Number",
	ConvertFromMap:       `FieldAsUint(%s, %s)`,
	FullNameDoc:          "Number",
	AsJokerObject:        "BigIntU(uint64(%s)%s)",
	PromoteType:          "uint64(%s)",
}

var UInt8 = &Info{
	FullName:             "Int",
	ArgClojureType:       "Int",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Uint8",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsUint8(%s, %s)`,
	FullNameDoc:          "Int",
	AsJokerObject:        "Int(int(%s)%s)",
	PromoteType:          "int(%s)",
}

var UInt16 = &Info{
	FullName:             "Int",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Uint16",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsUint16(%s, %s)`,
	FullNameDoc:          "Int",
	AsJokerObject:        "Int(int(%s)%s)",
	PromoteType:          "int(%s)",
}

var UInt32 = &Info{
	FullName:             "Number",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Uint32",
	ArgClojureArgType:    "Number",
	ConvertFromMap:       `FieldAsUint32(%s, %s)`,
	FullNameDoc:          "Number",
	AsJokerObject:        "BigIntU(uint64(%s)%s)",
	PromoteType:          "int64(%s)",
}

var UInt64 = &Info{
	FullName:             "Number",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".BigInt().Uint64()",
	ArgExtractFunc:       "Uint64",
	ArgClojureArgType:    "Number",
	ConvertFromMap:       `FieldAsUint64(%s, %s)`,
	FullNameDoc:          "Number",
	AsJokerObject:        "BigIntU(%s%s)",
	PromoteType:          "uint64(%s)",
}

var UIntPtr = &Info{
	FullName:             "Number",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".BigInt().Uint64()",
	ArgExtractFunc:       "UintPtr",
	ArgClojureArgType:    "Number",
	ConvertFromMap:       `FieldAsUintPtr(%s, %s)`,
	FullNameDoc:          "Number",
	AsJokerObject:        "BigIntU(uint64(%s)%s)",
	PromoteType:          "uint64(%s)",
}

var Float32 = &Info{
	FullName:             "Double",
	ArgClojureType:       "",
	ArgFromClojureObject: "",
	ArgExtractFunc:       "ABEND007(find these)",
	ArgClojureArgType:    "Double",
	ConvertFromMap:       `FieldAsDouble(%s, %s)`,
	FullNameDoc:          "Double",
	AsJokerObject:        "Double(float64(%s)%s)",
	PromoteType:          "float64(%s)",
}

var Float64 = &Info{
	FullName:             "Double",
	ArgClojureType:       "Double",
	ArgFromClojureObject: "",
	ArgExtractFunc:       "ABEND007(find these)",
	ArgClojureArgType:    "Double",
	ConvertFromMap:       `FieldAsDouble(%s, %s)`,
	FullNameDoc:          "Double",
	AsJokerObject:        "Double(%s%s)",
	PromoteType:          "%s",
}

var Complex128 = &Info{
	FullName:             "ABEND007(find these)",
	ArgClojureType:       "",
	ArgFromClojureObject: "",
	ArgExtractFunc:       "ABEND007(find these)",
	ArgClojureArgType:    "ABEND007(find these)",
	ConvertFromMap:       "", // TODO: support this in Joker, even if via just [real imag]
	FullNameDoc:          "ABEND007(find these)",
	AsJokerObject:        "Complex(%s%s)",
}

var goTypeMap = map[string]*Info{
	"nil":        Nil,
	"error":      Error,
	"bool":       Boolean,
	"byte":       Byte,
	"rune":       Rune,
	"string":     String,
	"int":        Int,
	"int8":       Int8,
	"int16":      Int16,
	"int32":      Int32,
	"int64":      Int64,
	"uint":       UInt,
	"uint8":      UInt8,
	"uint16":     UInt16,
	"uint32":     UInt32,
	"uint64":     UInt64,
	"uintptr":    UIntPtr,
	"float32":    Float32,
	"float64":    Float64,
	"complex128": Complex128,
}

var ConversionsFn func(e Expr) (fromClojure, fromMap string)
