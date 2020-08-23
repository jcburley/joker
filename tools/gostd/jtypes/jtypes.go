package jtypes

type Info struct {
	ArgExtractFunc     string
	ArgClojureArgType  string // Clojure argument type for a Go function arg with my type
	ConvertFromClojure string // Pattern to convert a (scalar) %s to this type
	ConvertToClojure   string // Pattern to convert this type to an appropriate Clojure object
	AsJokerObject      string // Pattern to convert this type to a normal Joker type, or empty string to simply wrap in a GoObject
}

var Nil = Info{}

var Error = Info{
	ArgExtractFunc:    "Error",
	ArgClojureArgType: "Error",
	ConvertToClojure:  "Error(%s%s)",
	AsJokerObject:     "Error(%s%s)",
}

var Bool = Info{
	ArgExtractFunc:    "Boolean",
	ArgClojureArgType: "Boolean",
	ConvertToClojure:  "Boolean(%s%s)",
	AsJokerObject:     "Boolean(%s%s)",
}

var Byte = Info{
	ArgExtractFunc:    "Byte",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	AsJokerObject:     "Int(int(%s)%s)",
}

var Rune = Info{
	ArgExtractFunc:    "Char",
	ArgClojureArgType: "Char",
	ConvertToClojure:  "Char(%s%s)",
	AsJokerObject:     "Char(%s%s)",
}

var String = Info{
	ArgExtractFunc:    "String",
	ArgClojureArgType: "String",
	ConvertToClojure:  "String(%s%s)",
	AsJokerObject:     "String(%s%s)",
}

var Int = Info{
	ArgExtractFunc:    "Int",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(%s%s)",
	AsJokerObject:     "Int(%s%s)",
}

var Int32 = Info{
	ArgExtractFunc:    "Int32",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	AsJokerObject:     "Int(int(%s)%s)",
}

var Int64 = Info{
	ArgExtractFunc:    "Int64",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigInt(%s%s)",
	AsJokerObject:     "BigInt(%s%s)",
}

var UInt = Info{
	ArgExtractFunc:    "Uint",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigIntU(uint64(%s)%s)",
	AsJokerObject:     "BigIntU(uint64(%s)%s)",
}

var UInt8 = Info{
	ArgExtractFunc:    "Uint8",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	AsJokerObject:     "Int(int(%s)%s)",
}

var UInt16 = Info{
	ArgExtractFunc:    "Uint16",
	ArgClojureArgType: "Int",
	ConvertToClojure:  "Int(int(%s)%s)",
	AsJokerObject:     "Int(int(%s)%s)",
}

var UInt32 = Info{
	ArgExtractFunc:    "Uint32",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigIntU(uint64(%s)%s)",
	AsJokerObject:     "BigIntU(uint64(%s)%s)",
}

var UInt64 = Info{
	ArgExtractFunc:    "Uint64",
	ArgClojureArgType: "Number",
	ConvertToClojure:  "BigIntU(%s%s)",
	AsJokerObject:     "BigIntU(%s%s)",
}

var UIntPtr = Info{
	ArgExtractFunc:    "UintPtr",
	ArgClojureArgType: "Number",
	AsJokerObject:     "Number(%s%s)",
}

var Float32 = Info{
	ArgExtractFunc:    "ABEND007(find these)",
	ArgClojureArgType: "Double",
	AsJokerObject:     "Double(float64(%s)%s)",
}

var Float64 = Info{
	ArgExtractFunc:    "ABEND007(find these)",
	ArgClojureArgType: "Double",
	AsJokerObject:     "Double(%s%s)",
}

var Complex128 = Info{
	ArgExtractFunc:    "ABEND007(find these)",
	ArgClojureArgType: "ABEND007(find these)",
}
