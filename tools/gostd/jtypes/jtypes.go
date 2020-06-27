package jtypes

import (
	. "go/ast"
)

func JokerTypeInfo(e Expr) *Info {
	return nil
}

type Info struct {
	ArgExtractFunc     string // Call Extract<this>() for arg with my type
	ArgClojureArgType  string // Clojure argument type for a Go function arg with my type
	ConvertFromClojure string // Pattern to convert a (scalar) %s to this type
	ConvertToClojure   string // Pattern to convert this type to an appropriate Clojure object
	Nullable           bool   // Can an instance of the type == nil (e.g. 'error' type)?
}
