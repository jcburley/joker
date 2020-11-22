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
	}
	return "", fmt.Sprintf("ABEND883(jtypes.go: unrecognized Expr type %T at: %s)", e, Unix(WhereAt(e.Pos()))), nil
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
		JokerName:    fullName,
		JokerNameDoc: fullName,
		Namespace:    ns,
	}

	typesByExpr[e] = info
	typesByFullname[fullName] = info

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

var Boolean = &Info{
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
	JokerName:         "Int",
	JokerNameDoc:      "Int",
	AsJokerObject:     "Int(int(%s)%s)",
}

var Int16 = &Info{
	ArgExtractFunc:    "Int16",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerName:         "Int",
	JokerNameDoc:      "Int",
	AsJokerObject:     "Int(int(%s)%s)",
}

var Int32 = &Info{
	ArgExtractFunc:    "Int32",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerName:         "Int",
	JokerNameDoc:      "Int",
	AsJokerObject:     "Int(int(%s)%s)",
}

var Int64 = &Info{
	ArgExtractFunc:    "Int64",
	ArgClojureArgType: "BigInt",
	ConvertToClojure:  "BigInt(%s%s)",
	JokerName:         "BigInt",
	JokerNameDoc:      "BigInt",
	AsJokerObject:     "BigInt(%s%s)",
}

var UInt = &Info{
	ArgExtractFunc:    "Uint",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigIntU(uint64(%s)%s)",
	JokerName:         "Number",
	JokerNameDoc:      "Number",
	AsJokerObject:     "BigIntU(uint64(%s)%s)",
}

var UInt8 = &Info{
	ArgExtractFunc:    "Uint8",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerName:         "Int",
	JokerNameDoc:      "Int",
	AsJokerObject:     "Int(int(%s)%s)",
}

var UInt16 = &Info{
	ArgExtractFunc:    "Uint16",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	JokerName:         "Int",
	JokerNameDoc:      "Int",
	AsJokerObject:     "Int(int(%s)%s)",
}

var UInt32 = &Info{
	ArgExtractFunc:    "Uint32",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigIntU(uint64(%s)%s)",
	JokerName:         "Number",
	JokerNameDoc:      "Number",
	AsJokerObject:     "BigIntU(uint64(%s)%s)",
}

var UInt64 = &Info{
	ArgExtractFunc:    "Uint64",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigIntU(%s%s)",
	JokerName:         "Number",
	JokerNameDoc:      "Number",
	AsJokerObject:     "BigIntU(%s%s)",
}

var UIntPtr = &Info{
	ArgExtractFunc:    "UintPtr",
	ArgClojureArgType: "Number",
	JokerName:         "Number",
	JokerNameDoc:      "Number",
	AsJokerObject:     "BigIntU(%s%s)",
}

var Float32 = &Info{
	ArgExtractFunc:    "ABEND007(find these)",
	ArgClojureArgType: "Double",
	JokerName:         "Double",
	JokerNameDoc:      "Double",
	AsJokerObject:     "Double(float64(%s)%s)",
}

var Float64 = &Info{
	ArgExtractFunc:    "ABEND007(find these)",
	ArgClojureArgType: "Double",
	JokerName:         "Double",
	JokerNameDoc:      "Double",
	AsJokerObject:     "Double(%s%s)",
}

var Complex128 = &Info{
	ArgExtractFunc:    "ABEND007(find these)",
	ArgClojureArgType: "ABEND007(find these)",
	JokerName:         "ABEND007(find these)",
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
