package jtypes

import (
	"fmt"
	. "github.com/candid82/joker/tools/gostd/godb"
	. "github.com/candid82/joker/tools/gostd/utils"
	. "go/ast"
	"go/types"
)

// Info on Joker types, including map of Joker type names to said type
// info.  A Joker type name is either unqualified (built-in, not
// namespace-rooted) or fully qualified by a namespace name
// (e.g. "go.std.example/SomeType").
type Info struct {
	Expr               Expr   // [key] The canonical referencing expression (if any)
	FullName           string // [key] Full name of type as a Joker expression
	ArgExtractFunc     string
	ArgClojureArgType  string // Clojure argument type for a Go function arg with my type
	ConvertFromClojure string // Pattern to convert a (scalar) %s to this type
	ConvertToClojure   string // Pattern to convert this type to an appropriate Clojure object
	JokerNameDoc       string // Full name of type as a Joker expression (for documentation)
	AsJokerObject      string // Pattern to convert this type to a normal Joker type; empty string means wrap in a GoObject
	Namespace          string // In which this type resides (empty string means a global Joker namespace)
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

func typeNameForExpr(e Expr) (ns, name string, info *Info) {
	switch v := e.(type) {
	case *Ident:
		if types.Universe.Lookup(v.Name) == nil {
			return ClojureNamespaceForExpr(e), v.Name, nil
		}
		info, found := goTypeMap[v.Name]
		if !found {
			panic(fmt.Sprintf("no type info for universal symbol `%s'", v.Name))
		}
		return "", info.JokerNameDoc, info
	case *ArrayType:
		ns, name, _ = typeNameForExpr(v.Elt)
		return ns, "arrayOf" + name, nil
	case *StarExpr:
		ns, name, _ = typeNameForExpr(v.X)
		return ns, "refTo" + name, nil
	case *SelectorExpr:
		pkgName := v.X.(*Ident).Name
		fullPathUnix := Unix(FileAt(v.Pos()))
		rf := GoFileForExpr(v)
		if fullPkgName, found := (*rf.Spaces)[pkgName]; found {
			return fullPkgName.String(), v.Sel.Name, nil
		}
		panic(fmt.Sprintf("processing %s: could not find %s in %s",
			WhereAt(v.Pos()), pkgName, fullPathUnix))
	default:
		return "", "GoObject", nil
	}
}

func TypeInfoForExpr(e Expr) *Info {
	if info, ok := typesByExpr[e]; ok {
		return info
	}

	ns, name, info := typeNameForExpr(e)

	if info != nil {
		// Already found info on builtin Go type, so just return that.
		typesByExpr[e] = info
		return info
	}

	fullName := combine(ns, name)

	if info, found := typesByFullname[fullName]; found {
		typesByExpr[e] = info
		return info
	}

	info = &Info{
		Expr:         e,
		FullName:     fullName,
		JokerNameDoc: fullName,
		Namespace:    ns,
	}

	typesByExpr[e] = info
	typesByFullname[fullName] = info

	return info
}

func (ti *Info) Register() {
	if _, found := typesByFullname[ti.FullName]; !found {
		typesByFullname[ti.FullName] = ti
	}
}

var Nil = &Info{}

var Error = &Info{
	FullName:          "Error",
	ArgExtractFunc:    "Error",
	ArgClojureArgType: "Error",
	ConvertToClojure:  "Error(%s%s)",
	JokerNameDoc:      "Error",
	AsJokerObject:     "Error(%s%s)",
}

var Boolean = &Info{
	FullName:          "Boolean",
	ArgExtractFunc:    "Boolean",
	ArgClojureArgType: "Boolean",
	ConvertToClojure:  "Boolean(%s%s)",
	JokerNameDoc:      "Boolean",
	AsJokerObject:     "Boolean(%s%s)",
}

var Byte = &Info{
	FullName:          "Byte",
	ArgExtractFunc:    "Byte",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerNameDoc:      "Byte",
	AsJokerObject:     "Int(int(%s)%s)",
}

var Rune = &Info{
	FullName:          "Char",
	ArgExtractFunc:    "Char",
	ArgClojureArgType: "Char",
	ConvertToClojure:  "Char(%s%s)",
	JokerNameDoc:      "Char",
	AsJokerObject:     "Char(%s%s)",
}

var String = &Info{
	FullName:          "String",
	ArgExtractFunc:    "String",
	ArgClojureArgType: "String",
	ConvertToClojure:  "String(%s%s)",
	JokerNameDoc:      "String",
	AsJokerObject:     "String(%s%s)",
}

var Int = &Info{
	FullName:          "Int",
	ArgExtractFunc:    "Int",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(%s%s)",
	JokerNameDoc:      "Int",
	AsJokerObject:     "Int(%s%s)",
}

var Int8 = &Info{
	FullName:          "Int",
	ArgExtractFunc:    "Int8",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerNameDoc:      "Int",
	AsJokerObject:     "Int(int(%s)%s)",
}

var Int16 = &Info{
	FullName:          "Int",
	ArgExtractFunc:    "Int16",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerNameDoc:      "Int",
	AsJokerObject:     "Int(int(%s)%s)",
}

var Int32 = &Info{
	FullName:          "Int",
	ArgExtractFunc:    "Int32",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerNameDoc:      "Int",
	AsJokerObject:     "Int(int(%s)%s)",
}

var Int64 = &Info{
	FullName:          "BigInt",
	ArgExtractFunc:    "Int64",
	ArgClojureArgType: "BigInt",
	ConvertToClojure:  "BigInt(%s%s)",
	JokerNameDoc:      "BigInt",
	AsJokerObject:     "BigInt(%s%s)",
}

var UInt = &Info{
	FullName:          "Number",
	ArgExtractFunc:    "Uint",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigIntU(uint64(%s)%s)",
	JokerNameDoc:      "Number",
	AsJokerObject:     "BigIntU(uint64(%s)%s)",
}

var UInt8 = &Info{
	FullName:          "Int",
	ArgExtractFunc:    "Uint8",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerNameDoc:      "Int",
	AsJokerObject:     "Int(int(%s)%s)",
}

var UInt16 = &Info{
	FullName:          "Int",
	ArgExtractFunc:    "Uint16",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerNameDoc:      "Int",
	AsJokerObject:     "Int(int(%s)%s)",
}

var UInt32 = &Info{
	FullName:          "Number",
	ArgExtractFunc:    "Uint32",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigIntU(uint64(%s)%s)",
	JokerNameDoc:      "Number",
	AsJokerObject:     "BigIntU(uint64(%s)%s)",
}

var UInt64 = &Info{
	FullName:          "Number",
	ArgExtractFunc:    "Uint64",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigIntU(%s%s)",
	JokerNameDoc:      "Number",
	AsJokerObject:     "BigIntU(%s%s)",
}

var UIntPtr = &Info{
	FullName:          "Number",
	ArgExtractFunc:    "UintPtr",
	ArgClojureArgType: "Number",
	JokerNameDoc:      "Number",
	AsJokerObject:     "BigIntU(%s%s)",
}

var Float32 = &Info{
	FullName:          "Double",
	ArgExtractFunc:    "ABEND007(find these)",
	ArgClojureArgType: "Double",
	JokerNameDoc:      "Double",
	AsJokerObject:     "Double(float64(%s)%s)",
}

var Float64 = &Info{
	FullName:          "Double",
	ArgExtractFunc:    "ABEND007(find these)",
	ArgClojureArgType: "Double",
	JokerNameDoc:      "Double",
	AsJokerObject:     "Double(%s%s)",
}

var Complex128 = &Info{
	FullName:          "ABEND007(find these)",
	ArgExtractFunc:    "ABEND007(find these)",
	ArgClojureArgType: "ABEND007(find these)",
	JokerNameDoc:      "ABEND007(find these)",
	AsJokerObject:     "Complex(%s%s)",
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
