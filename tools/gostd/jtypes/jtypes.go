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
	ArgExtractFunc     string
	ArgClojureArgType  string // Clojure argument type for a Go function arg with my type
	ConvertFromClojure string // Pattern to convert a (scalar) %s to this type
	ConvertToClojure   string // Pattern to convert this type to an appropriate Clojure object
	JokerName          string // Full name of type as a Joker expression
	JokerNameDoc       string // Full name of type as a Joker expression (for documentation)
	AsJokerObject      string // Pattern to convert this type to a normal Joker type; empty string means wrap in a GoObject
	Namespace          string // In which this type resides (empty string means a global Joker namespace)
}

func combine(ns, name string) string {
	if ns == "" {
		return name
	}
	return ns + "/" + name
}

func typeNameForExpr(e Expr) (ns, name string) {
	switch v := e.(type) {
	case *Ident:
		if types.Universe.Lookup(v.Name) == nil {
			return ClojureNamespaceForExpr(e), v.Name
		}
		info, found := goTypeMap[v.Name]
		if !found {
			panic(fmt.Sprintf("no type info for universal symbol `%s'", v.Name))
		}
		return "", info.JokerNameDoc
	case *ArrayType:
		ns, name = typeNameForExpr(v.Elt)
		return ns, "arrayOf" + name
	case *StarExpr:
		ns, name = typeNameForExpr(v.X)
		return ns, "refTo" + name
	}
	return "", fmt.Sprintf("ABEND883(jtypes.go: unrecognized Expr type %T at: %s)", e, Unix(WhereAt(e.Pos())))
}

func TypeInfoForExpr(e Expr) *Info {
	ns, name := typeNameForExpr(e)

	fullName := combine(ns, name)

	if info, found := typeMap[fullName]; found {
		return info
	}

	info := &Info{
		JokerName:    fullName,
		JokerNameDoc: fullName,
		Namespace:    ns,
	}

	typeMap[fullName] = info

	return info
}

var Nil = &Info{}

var Error = &Info{
	ArgExtractFunc:    "Error",
	ArgClojureArgType: "Error",
	ConvertToClojure:  "Error(%s%s)",
	JokerName:         "Error",
	JokerNameDoc:      "Error",
	AsJokerObject:     "Error(%s%s)",
}

var Bool = &Info{
	ArgExtractFunc:    "Boolean",
	ArgClojureArgType: "Boolean",
	ConvertToClojure:  "Boolean(%s%s)",
	JokerName:         "Boolean",
	JokerNameDoc:      "Boolean",
	AsJokerObject:     "Boolean(%s%s)",
}

var Byte = &Info{
	ArgExtractFunc:    "Byte",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerName:         "Byte",
	JokerNameDoc:      "Byte",
	AsJokerObject:     "Int(int(%s)%s)",
}

var Rune = &Info{
	ArgExtractFunc:    "Char",
	ArgClojureArgType: "Char",
	ConvertToClojure:  "Char(%s%s)",
	JokerName:         "Char",
	JokerNameDoc:      "Char",
	AsJokerObject:     "Char(%s%s)",
}

var String = &Info{
	ArgExtractFunc:    "String",
	ArgClojureArgType: "String",
	ConvertToClojure:  "String(%s%s)",
	JokerName:         "String",
	JokerNameDoc:      "String",
	AsJokerObject:     "String(%s%s)",
}

var Int = &Info{
	ArgExtractFunc:    "Int",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(%s%s)",
	JokerName:         "Int",
	JokerNameDoc:      "Int",
	AsJokerObject:     "Int(%s%s)",
}

var Int8 = &Info{
	ArgExtractFunc:    "Int8",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerName:         "Int8",
	JokerNameDoc:      "Int8",
	AsJokerObject:     "Int(int(%s)%s)",
}

var Int16 = &Info{
	ArgExtractFunc:    "Int16",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerName:         "Int16",
	JokerNameDoc:      "Int16",
	AsJokerObject:     "Int(int(%s)%s)",
}

var Int32 = &Info{
	ArgExtractFunc:    "Int32",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerName:         "Int32",
	JokerNameDoc:      "Int32",
	AsJokerObject:     "Int(int(%s)%s)",
}

var Int64 = &Info{
	ArgExtractFunc:    "Int64",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigInt(%s%s)",
	JokerName:         "Int64",
	JokerNameDoc:      "Int64",
	AsJokerObject:     "BigInt(%s%s)",
}

var UInt = &Info{
	ArgExtractFunc:    "Uint",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigIntU(uint64(%s)%s)",
	JokerName:         "Uint",
	JokerNameDoc:      "Uint",
	AsJokerObject:     "BigIntU(uint64(%s)%s)",
}

var UInt8 = &Info{
	ArgExtractFunc:    "Uint8",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerName:         "Uint8",
	JokerNameDoc:      "Uint8",
	AsJokerObject:     "Int(int(%s)%s)",
}

var UInt16 = &Info{
	ArgExtractFunc:    "Uint16",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerName:         "Uint16",
	JokerNameDoc:      "Uint16",
	AsJokerObject:     "Int(int(%s)%s)",
}

var UInt32 = &Info{
	ArgExtractFunc:    "Uint32",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigIntU(uint64(%s)%s)",
	JokerName:         "Uint32",
	JokerNameDoc:      "Uint32",
	AsJokerObject:     "BigIntU(uint64(%s)%s)",
}

var UInt64 = &Info{
	ArgExtractFunc:    "Uint64",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigIntU(%s%s)",
	JokerName:         "Uint64",
	JokerNameDoc:      "Uint64",
	AsJokerObject:     "BigIntU(%s%s)",
}

var UIntPtr = &Info{
	ArgExtractFunc:    "UintPtr",
	ArgClojureArgType: "Number",
	JokerName:         "UintPtr",
	JokerNameDoc:      "UintPtr",
	AsJokerObject:     "Number(%s%s)",
}

var Float32 = &Info{
	ArgExtractFunc:    "ABEND007(find these)",
	ArgClojureArgType: "Double",
	JokerName:         "ABEND007(find these)",
	JokerNameDoc:      "ABEND007(find these)",
	AsJokerObject:     "Double(float64(%s)%s)",
}

var Float64 = &Info{
	ArgExtractFunc:    "ABEND007(find these)",
	ArgClojureArgType: "Double",
	JokerName:         "ABEND007(find these)",
	JokerNameDoc:      "ABEND007(find these)",
	AsJokerObject:     "Double(%s%s)",
}

var Complex128 = &Info{
	ArgExtractFunc:    "ABEND007(find these)",
	ArgClojureArgType: "ABEND007(find these)",
	JokerName:         "ABEND007(find these)",
	JokerNameDoc:      "ABEND007(find these)",
	AsJokerObject:     "Complex(%s%s)",
}

var typeMap = map[string]*Info{
	"Nil":        Nil,
	"Error":      Error,
	"Bool":       Bool,
	"Byte":       Byte,
	"Rune":       Rune,
	"String":     String,
	"Int":        Int,
	"Int32":      Int32,
	"Int64":      Int64,
	"UInt":       UInt,
	"UInt8":      UInt8,
	"UInt16":     UInt16,
	"UInt32":     UInt32,
	"UInt64":     UInt64,
	"UIntPtr":    UIntPtr,
	"Float32":    Float32,
	"Float64":    Float64,
	"Complex128": Complex128,
}

var goTypeMap = map[string]*Info{
	"nil":        Nil,
	"error":      Error,
	"bool":       Bool,
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
