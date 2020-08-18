package jtypes

type Info interface {
	ArgExtractFunc() string     // Call Extract<this>() for arg with my type
	ArgClojureArgType() string  // Clojure argument type for a Go function arg with my type
	ConvertFromClojure() string // Pattern to convert a (scalar) %s to this type
	ConvertToClojure() string   // Pattern to convert this type to an appropriate Clojure object
	AsJokerObject() string      // Pattern to convert this type to a normal Joker type, or empty string to simply wrap in a GoObject
	Nullable() bool             // Can an instance of the type == nil (e.g. 'error' type)?
}

type info struct {
	argExtractFunc     string // Call Extract<this>() for arg with my type
	argClojureArgType  string // Clojure argument type for a Go function arg with my type
	convertFromClojure string // Pattern to convert a (scalar) %s to this type
	convertToClojure   string // Pattern to convert this type to an appropriate Clojure object
	asJokerObject      string // Pattern to convert this type to a normal Joker type, or empty string to simply wrap in a GoObject
	nullable           bool   // Can an instance of the type == nil (e.g. 'error' type)?
}

func NewInfo(argExtractFunc, argClojureArgType, convertFromClojure, convertToClojure, asJokerObject string, nullable bool) Info {
	return &info{
		argExtractFunc:     argExtractFunc,
		argClojureArgType:  argClojureArgType,
		convertFromClojure: convertFromClojure,
		convertToClojure:   convertToClojure,
		asJokerObject:      asJokerObject,
		nullable:           nullable,
	}
}

func BadInfo(err string) Info {
	return &info{
		argExtractFunc:     err,
		argClojureArgType:  err,
		convertFromClojure: err,
		convertToClojure:   err,
	}
}

var Nil = info{}

var Error = info{
	argExtractFunc:    "Error",
	argClojureArgType: "Error",
	convertToClojure:  "Error(%s%s)",
	asJokerObject:     "Error(%s%s)",
	nullable:          true,
}

var Bool = info{
	argExtractFunc:    "Boolean",
	argClojureArgType: "Boolean",
	convertToClojure:  "Boolean(%s%s)",
	asJokerObject:     "Boolean(%s%s)",
}

var Byte = info{
	argExtractFunc:    "Byte",
	argClojureArgType: "Int",
	convertToClojure:  "Int(int(%s)%s)",
	asJokerObject:     "Int(int(%s)%s)",
}

var Rune = info{
	argExtractFunc:    "Char",
	argClojureArgType: "Char",
	convertToClojure:  "Char(%s%s)",
	asJokerObject:     "Char(%s%s)",
}

var String = info{
	argExtractFunc:    "String",
	argClojureArgType: "String",
	convertToClojure:  "String(%s%s)",
	asJokerObject:     "String(%s%s)",
}

var Int = info{
	argExtractFunc:    "Int",
	argClojureArgType: "Int",
	convertToClojure:  "Int(%s%s)",
	asJokerObject:     "Int(%s%s)",
}

var Int32 = info{
	argExtractFunc:    "Int32",
	argClojureArgType: "Int",
	convertToClojure:  "Int(int(%s)%s)",
	asJokerObject:     "Int(int(%s)%s)",
}

var Int64 = info{
	argExtractFunc:    "Int64",
	argClojureArgType: "Number",
	convertToClojure:  "BigInt(%s%s)",
	asJokerObject:     "BigInt(%s%s)",
}

var UInt = info{
	argExtractFunc:    "Uint",
	argClojureArgType: "Number",
	convertToClojure:  "BigIntU(uint64(%s)%s)",
	asJokerObject:     "BigIntU(uint64(%s)%s)",
}

var UInt8 = info{
	argExtractFunc:    "Uint8",
	argClojureArgType: "Int",
	convertToClojure:  "Int(int(%s)%s)",
	asJokerObject:     "Int(int(%s)%s)",
}

var UInt16 = info{
	argExtractFunc:    "Uint16",
	argClojureArgType: "Int",
	convertToClojure:  "Int(int(%s)%s)",
	asJokerObject:     "Int(int(%s)%s)",
}

var UInt32 = info{
	argExtractFunc:    "Uint32",
	argClojureArgType: "Number",
	convertToClojure:  "BigIntU(uint64(%s)%s)",
	asJokerObject:     "BigIntU(uint64(%s)%s)",
}

var UInt64 = info{
	argExtractFunc:    "Uint64",
	argClojureArgType: "Number",
	convertToClojure:  "BigIntU(%s%s)",
	asJokerObject:     "BigIntU(%s%s)",
}

var UIntPtr = info{
	argExtractFunc:    "UintPtr",
	argClojureArgType: "Number",
	asJokerObject:     "Number(%s%s)",
}

var Float32 = info{
	argExtractFunc:    "ABEND007(find these)",
	argClojureArgType: "Double",
	asJokerObject:     "Double(float64(%s)%s)",
}

var Float64 = info{
	argExtractFunc:    "ABEND007(find these)",
	argClojureArgType: "Double",
	asJokerObject:     "Double(%s%s)",
}

var Complex128 = info{
	argExtractFunc:    "ABEND007(find these)",
	argClojureArgType: "ABEND007(find these)",
}

func (jti info) ArgExtractFunc() string {
	return jti.argExtractFunc
}

func (jti info) ArgClojureArgType() string {
	return jti.argClojureArgType
}

func (jti info) ConvertFromClojure() string {
	return jti.convertFromClojure
}

func (jti info) ConvertToClojure() string {
	return jti.convertToClojure
}

func (jti info) AsJokerObject() string {
	return jti.asJokerObject
}

func (jti info) Nullable() bool {
	return jti.nullable
}
