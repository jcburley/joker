package core

import (
	"fmt"
	"math"
	"reflect"
)

type GoFn func(GoObject, Object) Object

type GoMembers map[string]GoFn

type GoMeta map[string]MetaHolder // TODO: Merge into GoMembers?

type GoTypeInfo struct {
	Name    string
	GoType  GoType
	Ctor    func(Object) Object
	Members GoMembers
	Meta    GoMeta
}

func LookupGoType(g interface{}) *GoTypeInfo {
	return GoTypes[reflect.TypeOf(g)]
}

func CheckGoArity(rcvr string, args Object, min, max int) *ArraySeq {
	n := 0
	switch s := args.(type) {
	case Nil:
		if min == 0 {
			return nil
		}
	case *ArraySeq:
		if max > 0 {
			return s
		}
		n = SeqCount(s)
	default:
	}
	panic(RT.NewError(fmt.Sprintf("Wrong number of args (%d) passed to %s; expects %s", n, rcvr, RangeString(min, max))))
}

func CheckGoNth(rcvr, t, name string, args *ArraySeq, n int) GoObject {
	a := SeqNth(args, n)
	res, ok := a.(GoObject)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type net.GoObject[%s], but is %T",
			n, name, rcvr, t, a)))
	}
	return res
}

func ExtractGoBoolean(rcvr, name string, args *ArraySeq, n int) bool {
	a := SeqNth(args, n)
	res, ok := a.(Boolean)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type core.Boolean, but is %T",
			n, name, rcvr, a)))
	}
	return res.B
}

func ExtractGoInt(rcvr, name string, args *ArraySeq, n int) int {
	a := SeqNth(args, n)
	res, ok := a.(Int)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type core.Int, but is %T",
			n, name, rcvr, a)))
	}
	return res.I
}

func ExtractGoUInt(rcvr, name string, args *ArraySeq, n int) uint {
	v := ExtractGoNumber(rcvr, name, args, n).BigInt().Uint64()
	if v > uint64(MAX_UINT) {
		panic(RT.NewArgTypeError(n, SeqNth(args, n), "uint"))
	}
	return uint(v)
}

func ExtractGoByte(rcvr, name string, args *ArraySeq, n int) byte {
	v := ExtractGoInt(rcvr, name, args, n)
	if v < 0 || v > 255 {
		panic(RT.NewArgTypeError(n, SeqNth(args, n), "byte"))
	}
	return byte(v)
}

func ExtractGoNumber(rcvr, name string, args *ArraySeq, n int) Number {
	a := SeqNth(args, n)
	res, ok := a.(Number)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type core.Number, but is %T",
			n, name, rcvr, a)))
	}
	return res
}

func ExtractGoInt32(rcvr, name string, args *ArraySeq, n int) int32 {
	v := ExtractGoNumber(rcvr, name, args, n).BigInt().Int64()
	if v > math.MaxInt32 {
		panic(RT.NewArgTypeError(n, SeqNth(args, n), "int32"))
	}
	return int32(v)
}

func ExtractGoUInt32(rcvr, name string, args *ArraySeq, n int) uint32 {
	v := ExtractGoNumber(rcvr, name, args, n).BigInt().Uint64()
	if v > math.MaxUint32 {
		panic(RT.NewArgTypeError(n, SeqNth(args, n), "uint32"))
	}
	return uint32(v)
}

func ExtractGoInt64(rcvr, name string, args *ArraySeq, n int) int64 {
	return ExtractGoNumber(rcvr, name, args, n).BigInt().Int64()
}

func ExtractGoUInt64(rcvr, name string, args *ArraySeq, n int) uint64 {
	return ExtractGoNumber(rcvr, name, args, n).BigInt().Uint64()
}

func ExtractGoChar(rcvr, name string, args *ArraySeq, n int) rune {
	a := SeqNth(args, n)
	res, ok := a.(Char)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type core.Char, but is %T",
			n, name, rcvr, a)))
	}
	return res.Ch
}

func ExtractGoString(rcvr, name string, args *ArraySeq, n int) string {
	a := SeqNth(args, n)
	res, ok := a.(String)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type core.String, but is %T",
			n, name, rcvr, a)))
	}
	return res.S
}

func ExtractGoError(rcvr, name string, args *ArraySeq, n int) error {
	a := SeqNth(args, n)
	res, ok := a.(Error)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type core.Error, but is %T",
			n, name, rcvr, a)))
	}
	return res
}

func ExtractGoUIntPtr(rcvr, name string, args *ArraySeq, n int) uintptr {
	return uintptr(ExtractGoUInt64(rcvr, name, args, n))
}

func GoObjectGet(o interface{}, key Object) (bool, Object) {
	v := reflect.Indirect(reflect.ValueOf(o))
	switch v.Kind() {
	case reflect.Struct:
		f := v.FieldByName(key.(String).S)
		if f != reflect.ValueOf(nil) {
			return true, MakeGoObject(f.Interface())
		}
	case reflect.Map:
		// Ignore key, return vector of keys (assuming they're strings)
		keys := v.MapKeys()
		objs := make([]Object, 0, 32)
		for _, k := range keys {
			objs = append(objs, MakeString(k.String()))
		}
		return true, NewVectorFrom(objs...)
	}
	panic(fmt.Sprintf("type=%T kind=%s\n", o, reflect.TypeOf(o).Kind().String()))
}

var GoTypes map[reflect.Type]*GoTypeInfo = map[reflect.Type]*GoTypeInfo{}
