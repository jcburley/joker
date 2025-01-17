// This file is generated by generate-std.joke script. Do not edit manually!

package runtime

import (
	. "github.com/candid82/joker/core"
	"runtime"
)

var __go_root__P ProcFn = __go_root_
var go_root_ Proc = Proc{Fn: __go_root__P, Name: "go_root_", Package: "std/runtime"}

func __go_root_(_args []Object) Object {
	_c := len(_args)
	switch {
	case _c == 0:
		_res := runtime.GOROOT()
		return MakeString(_res)

	default:
		PanicArity(_c)
	}
	return NIL
}

var __go_version__P ProcFn = __go_version_
var go_version_ Proc = Proc{Fn: __go_version__P, Name: "go_version_", Package: "std/runtime"}

func __go_version_(_args []Object) Object {
	_c := len(_args)
	switch {
	case _c == 0:
		_res := runtime.Version()
		return MakeString(_res)

	default:
		PanicArity(_c)
	}
	return NIL
}

var __joker_version__P ProcFn = __joker_version_
var joker_version_ Proc = Proc{Fn: __joker_version__P, Name: "joker_version_", Package: "std/runtime"}

func __joker_version_(_args []Object) Object {
	_c := len(_args)
	switch {
	case _c == 0:
		_res := VERSION
		return MakeString(_res)

	default:
		PanicArity(_c)
	}
	return NIL
}

func Init() {

	initNative()

	InternsOrThunks()
}

var runtimeNamespace = GLOBAL_ENV.EnsureSymbolIsLib(MakeSymbol("joker.runtime"))

func init() {
	runtimeNamespace.Lazy = Init
}
