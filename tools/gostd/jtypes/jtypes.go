package jtypes

type Info interface {
	ArgExtractFunc() string     // Call Extract<this>() for arg with my type
	ArgClojureArgType() string  // Clojure argument type for a Go function arg with my type
	ConvertFromClojure() string // Pattern to convert a (scalar) %s to this type
	ConvertToClojure() string   // Pattern to convert this type to an appropriate Clojure object
	Nullable() bool             // Can an instance of the type == nil (e.g. 'error' type)?
}

type info struct {
	argExtractFunc     string // Call Extract<this>() for arg with my type
	argClojureArgType  string // Clojure argument type for a Go function arg with my type
	convertFromClojure string // Pattern to convert a (scalar) %s to this type
	convertToClojure   string // Pattern to convert this type to an appropriate Clojure object
	nullable           bool   // Can an instance of the type == nil (e.g. 'error' type)?
}

var Nil = info{}

var String = info{
	argExtractFunc:    "String",
	argClojureArgType: "String",
	convertToClojure:  "String(%s%s)",
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

func (jti info) Nullable() bool {
	return jti.nullable
}
