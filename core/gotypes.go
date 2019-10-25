package core

import (
	"errors"
	"fmt"
	"math"
	"reflect"
)

type GoMembers map[string]*Var

type GoTypeInfo struct {
	Name    string
	GoType  *GoType
	Ctor    func(Object) Object
	Members GoMembers
	Type    reflect.Type
}

func LookupGoType(g interface{}) *GoTypeInfo {
	ix := SwitchGoType(g)
	if ix < 0 || ix >= len(GoTypesVec) || GoTypesVec[ix] == nil {
		return nil
		// panic(fmt.Sprintf("LookupGoType: %T returned %d (max=%d)\n", g, ix, len(GoTypesVec)-1))
	}
	return GoTypesVec[ix]
}

func GoTypeToString(ty reflect.Type) string {
	switch ty.Kind() {
	case reflect.Array:
		return "[]" + GoTypeToString(ty.Elem())
	case reflect.Ptr:
		return "*" + GoTypeToString(ty.Elem())
	}
	return ty.PkgPath() + "." + ty.Name()
}

func GoObjectTypeToString(o interface{}) string {
	return GoTypeToString(reflect.TypeOf(o))
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

func FieldAsBoolean(o Map, k string) bool {
	ok, v := o.Get(MakeKeyword(k))
	if !ok {
		return false
	}
	res, ok := v.(Boolean)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type core.Boolean, but is %T",
			k, v)))
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

func FieldAsInt(o Map, k string) int {
	v := FieldAsNumber(o, k).BigInt().Int64()
	if v > int64(MAX_INT) || v < int64(MIN_INT) {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type int, but is too large",
			v, k)))
	}
	return int(v)
}

func ExtractGoUint(rcvr, name string, args *ArraySeq, n int) uint {
	v := ExtractGoNumber(rcvr, name, args, n).BigInt().Uint64()
	if v > uint64(MAX_UINT) {
		panic(RT.NewArgTypeError(n, SeqNth(args, n), "uint"))
	}
	return uint(v)
}

func FieldAsUint(o Map, k string) uint {
	v := FieldAsNumber(o, k).BigInt().Uint64()
	if v > uint64(MAX_UINT) {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type uint, but is too large",
			v, k)))
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

func FieldAsByte(o Map, k string) byte {
	v := FieldAsInt(o, k)
	if v < 0 || v > 255 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type byte, but its magnitude is too large",
			v, k)))
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

func FieldAsNumber(o Map, k string) Number {
	ok, v := o.Get(MakeKeyword(k))
	if !ok {
		return MakeNumber(uint64(0))
	}
	res, ok := v.(Number)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type core.Number, but is %T",
			k, v)))
	}
	return res
}

func FieldAsInt8(o Map, k string) int8 {
	v := FieldAsNumber(o, k).BigInt().Int64()
	if v > math.MaxInt8 || v < math.MinInt8 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type int8, but its magnitude is too large",
			v, k)))
	}
	return int8(v)
}

func FieldAsInt16(o Map, k string) int16 {
	v := FieldAsNumber(o, k).BigInt().Int64()
	if v > math.MaxInt16 || v < math.MinInt16 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type int16, but its magnitude is too large",
			v, k)))
	}
	return int16(v)
}

func ExtractGoInt32(rcvr, name string, args *ArraySeq, n int) int32 {
	v := ExtractGoNumber(rcvr, name, args, n).BigInt().Int64()
	if v > math.MaxInt32 || v < math.MinInt32 {
		panic(RT.NewArgTypeError(n, SeqNth(args, n), "int32"))
	}
	return int32(v)
}

func FieldAsInt32(o Map, k string) int32 {
	v := FieldAsNumber(o, k).BigInt().Int64()
	if v > math.MaxInt32 || v < math.MinInt32 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type int32, but its magnitude is too large",
			v, k)))
	}
	return int32(v)
}

func FieldAsUint8(o Map, k string) uint8 {
	v := FieldAsNumber(o, k).BigInt().Uint64()
	if v > math.MaxUint8 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type uint8, but is too large",
			v, k)))
	}
	return uint8(v)
}

func FieldAsUint16(o Map, k string) uint16 {
	v := FieldAsNumber(o, k).BigInt().Uint64()
	if v > math.MaxUint16 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type uint16, but is too large",
			v, k)))
	}
	return uint16(v)
}

func ExtractGoUint32(rcvr, name string, args *ArraySeq, n int) uint32 {
	v := ExtractGoNumber(rcvr, name, args, n).BigInt().Uint64()
	if v > math.MaxUint32 {
		panic(RT.NewArgTypeError(n, SeqNth(args, n), "uint32"))
	}
	return uint32(v)
}

func FieldAsUint32(o Map, k string) uint32 {
	v := FieldAsNumber(o, k).BigInt().Uint64()
	if v > math.MaxUint32 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type uint32, but is too large",
			v, k)))
	}
	return uint32(v)
}

func ExtractGoInt64(rcvr, name string, args *ArraySeq, n int) int64 {
	return ExtractGoNumber(rcvr, name, args, n).BigInt().Int64()
}

func FieldAsInt64(o Map, k string) int64 {
	return FieldAsNumber(o, k).BigInt().Int64()
}

func ExtractGoUint64(rcvr, name string, args *ArraySeq, n int) uint64 {
	return ExtractGoNumber(rcvr, name, args, n).BigInt().Uint64()
}

func FieldAsUint64(o Map, k string) uint64 {
	return FieldAsNumber(o, k).BigInt().Uint64()
}

func ExtractGoUintPtr(rcvr, name string, args *ArraySeq, n int) uintptr {
	return uintptr(ExtractGoUint64(rcvr, name, args, n))
}

func FieldAsUintPtr(o Map, k string) uintptr {
	return uintptr(FieldAsUint64(o, k))
}

func FieldAsDouble(o Map, k string) float64 {
	ok, v := o.Get(MakeKeyword(k))
	if !ok {
		return 0
	}
	res, ok := v.(Double)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type core.Double, but is %T",
			k, v)))
	}
	return res.D
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

func FieldAsChar(o Map, k string) rune {
	ok, v := o.Get(MakeKeyword(k))
	if !ok {
		return 0
	}
	res, ok := v.(Char)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type core.Char, but is %T",
			k, v)))
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

func FieldAsString(o Map, k string) string {
	ok, v := o.Get(MakeKeyword(k))
	if !ok {
		return ""
	}
	res, ok := v.(String)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type core.String, but is %T",
			k, v)))
	}
	return res.S
}

func ExtractGoError(rcvr, name string, args *ArraySeq, n int) error {
	a := SeqNth(args, n)
	if s, ok := a.(String); ok {
		return errors.New(s.S)
	}
	res, ok := a.(Error)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type core.Error, but is %T",
			n, name, rcvr, a)))
	}
	return res
}

func FieldAsError(o Map, k string) error {
	ok, v := o.Get(MakeKeyword(k))
	if !ok || v.Equals(NIL) {
		return nil
	}
	if s, ok := v.(String); ok {
		return errors.New(s.S)
	}
	res, ok := v.(error)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type core.Error, but is %T",
			k, v)))
	}
	return res
}

func GoObjectGet(o interface{}, key Object) (bool, Object) {
	v := reflect.Indirect(reflect.ValueOf(o))
	switch v.Kind() {
	case reflect.Struct:
		f := v.FieldByName(key.(String).S)
		if f != reflect.ValueOf(nil) {
			return true, MakeGoObjectIfNeeded(f.Interface())
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

func MakeGoReceiver(name string, f func(GoObject, Object) Object, doc, added string, arglist *Vector) *Var {
	v := &Var{
		name:  MakeSymbol(name),
		Value: &GoReceiver{R: f},
	}
	m := MakeMeta(NewListFrom(arglist), doc, added)
	m.Add(KEYWORDS.name, v.name)
	v.meta = m
	return v
}
