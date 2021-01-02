package core

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
)

type GoMembers map[string]*Var

type GoTypeInfo struct {
	Name    string
	GoType  *GoType
	Ctor    func(Object) Object
	Members GoMembers
	Type    reflect.Type
}

// This seems quite good, e.g.:
// "Eval error: expected GoObject[go.std.net/IP], got GoObject[go.std.net/arrayOfIP]"
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

func Extract_bool(obj Object, pattern string) bool {
	return EnsureObjectIsBoolean(obj, pattern).B
}

func ExtractGoBoolean(rcvr, name string, args *ArraySeq, n int) bool {
	a := SeqNth(args, n)
	res, ok := a.(Boolean)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type Boolean, but is %T",
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
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type Boolean, but is %T",
			k, v)))
	}
	return res.B
}

func Extract_int(obj Object, pattern string) int {
	return EnsureObjectIsInt(obj, pattern).I
}

func ExtractGoInt(rcvr, name string, args *ArraySeq, n int) int {
	a := SeqNth(args, n)
	res, ok := a.(Int)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type Int, but is %T",
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

func Extract_uint(obj Object, pattern string) uint {
	v := EnsureObjectIsNumber(obj, pattern).BigInt().Uint64()
	if v > uint64(MAX_UINT) {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.FormatUint(uint64(v), 10)+" out of range for uint")))
	}
	return uint(v)
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

func Extract_byte(obj Object, pattern string) byte {
	v := EnsureObjectIsInt(obj, pattern).I
	if v < 0 || v > 255 {
		panic(RT.NewError(fmt.Sprintf(pattern, "Int "+strconv.Itoa(v)+" out of range for byte")))
	}
	return byte(v)
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
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type Number, but is %T",
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
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type Number, but is %T",
			k, v)))
	}
	return res
}

func Extract_int8(obj Object, pattern string) int8 {
	v := EnsureObjectIsInt(obj, pattern).I
	if v > math.MaxInt8 {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.Itoa(v)+" out of range for int8")))
	}
	return int8(v)
}

func FieldAsInt8(o Map, k string) int8 {
	v := FieldAsNumber(o, k).BigInt().Int64()
	if v > math.MaxInt8 || v < math.MinInt8 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type int8, but its magnitude is too large",
			v, k)))
	}
	return int8(v)
}

func Extract_int16(obj Object, pattern string) int16 {
	v := EnsureObjectIsInt(obj, pattern).I
	if v > math.MaxInt16 {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.Itoa(v)+" out of range for int16")))
	}
	return int16(v)
}

func FieldAsInt16(o Map, k string) int16 {
	v := FieldAsNumber(o, k).BigInt().Int64()
	if v > math.MaxInt16 || v < math.MinInt16 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type int16, but its magnitude is too large",
			v, k)))
	}
	return int16(v)
}

func Extract_int32(obj Object, pattern string) int32 {
	v := EnsureObjectIsInt(obj, pattern).I
	if v > math.MaxInt32 {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.Itoa(v)+" out of range for int32")))
	}
	return int32(v)
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

func Extract_uint8(obj Object, pattern string) uint8 {
	v := EnsureObjectIsInt(obj, pattern).I
	if v > math.MaxUint8 {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.Itoa(v)+" out of range for uint8")))
	}
	return uint8(v)
}

func FieldAsUint8(o Map, k string) uint8 {
	v := FieldAsNumber(o, k).BigInt().Uint64()
	if v > math.MaxUint8 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type uint8, but is too large",
			v, k)))
	}
	return uint8(v)
}

func Extract_uint16(obj Object, pattern string) uint16 {
	v := EnsureObjectIsInt(obj, pattern).I
	if v > math.MaxUint16 {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.Itoa(v)+" out of range for uint16")))
	}
	return uint16(v)
}

func FieldAsUint16(o Map, k string) uint16 {
	v := FieldAsNumber(o, k).BigInt().Uint64()
	if v > math.MaxUint16 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type uint16, but is too large",
			v, k)))
	}
	return uint16(v)
}

func Extract_uint32(obj Object, pattern string) uint32 {
	v := EnsureObjectIsNumber(obj, pattern).BigInt().Uint64()
	if v > math.MaxUint32 {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.FormatUint(uint64(v), 10)+" out of range for uint32")))
	}
	return uint32(v)
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

func Extract_int64(obj Object, pattern string) int64 {
	return EnsureObjectIsNumber(obj, pattern).BigInt().Int64()
}

func FieldAsInt64(o Map, k string) int64 {
	return FieldAsNumber(o, k).BigInt().Int64()
}

func Extract_uint64(obj Object, pattern string) uint64 {
	return EnsureObjectIsNumber(obj, pattern).BigInt().Uint64()
}

func ExtractGoUint64(rcvr, name string, args *ArraySeq, n int) uint64 {
	return ExtractGoNumber(rcvr, name, args, n).BigInt().Uint64()
}

func FieldAsUint64(o Map, k string) uint64 {
	return FieldAsNumber(o, k).BigInt().Uint64()
}

func Extract_uintptr(obj Object, pattern string) uintptr {
	v := EnsureObjectIsNumber(obj, pattern).BigInt().Uint64()
	if uint64(uintptr(v)) != v {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.FormatUint(v, 10)+" out of range for uintptr")))
	}
	return uintptr(v)
}

func ExtractGoUintPtr(rcvr, name string, args *ArraySeq, n int) uintptr {
	return uintptr(ExtractGoUint64(rcvr, name, args, n))
}

func FieldAsUintPtr(o Map, k string) uintptr {
	return uintptr(FieldAsUint64(o, k))
}

func Extract_float64(obj Object, pattern string) float64 {
	return EnsureObjectIsDouble(obj, pattern).D
}

func FieldAsDouble(o Map, k string) float64 {
	ok, v := o.Get(MakeKeyword(k))
	if !ok {
		return 0
	}
	res, ok := v.(Double)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type Double, but is %T",
			k, v)))
	}
	return res.D
}

func Extract_rune(obj Object, pattern string) rune {
	return EnsureObjectIsChar(obj, pattern).Ch
}

func ExtractGoChar(rcvr, name string, args *ArraySeq, n int) rune {
	a := SeqNth(args, n)
	res, ok := a.(Char)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type Char, but is %T",
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
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type Char, but is %T",
			k, v)))
	}
	return res.Ch
}

func Extract_string(obj Object, pattern string) string {
	return EnsureObjectIsString(obj, pattern).S
}

func ExtractGoString(rcvr, name string, args *ArraySeq, n int) string {
	a := SeqNth(args, n)
	res, ok := a.(String)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type String, but is %T",
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
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type String, but is %T",
			k, v)))
	}
	return res.S
}

func Extract_error(obj Object, pattern string) error {
	return EnsureObjectIsError(obj, pattern)
}

func ExtractGoError(rcvr, name string, args *ArraySeq, n int) error {
	a := SeqNth(args, n)
	if s, ok := a.(String); ok {
		return errors.New(s.S)
	}
	res, ok := a.(Error)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type Error or String, but is %T",
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
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type Error or String, but is %T",
			k, v)))
	}
	return res
}

func FieldAsGoObject(o Map, k string) interface{} {
	ok, v := o.Get(MakeKeyword(k))
	if !ok {
		return ""
	}
	res, ok := v.(GoObject)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type GoObject, but is %T",
			k, v)))
	}
	return res.O
}

func GoObjectGet(o interface{}, key Object) (bool, Object) {
	defer func() {
		if r := recover(); r != nil {
			panic(RT.NewError(fmt.Sprintf("%v", r)))
		}
	}()
	v := reflect.Indirect(reflect.ValueOf(o))
	switch v.Kind() {
	case reflect.Struct:
		if key.Equals(NIL) {
			// Special case for nil key: return vector of field names
			n := v.NumField()
			ty := reflect.TypeOf(o)
			objs := make([]Object, n)
			for i := 0; i < n; i++ {
				objs[i] = MakeString(ty.Field(i).Name)
			}
			return true, NewVectorFrom(objs...) // TODO: Someday: MakeGoObject(v.MapKeys())
		}
		f := v.FieldByName(key.(Fieldable).AsFieldName())
		return true, MakeGoObjectIfNeeded(f.Interface())
	case reflect.Map:
		if key.Equals(NIL) {
			// Special case for nil key (used during doc generation): return vector of keys as reflect.Value's
			return true, MakeGoObject(v.MapKeys())
		}
		return true, MakeGoObjectIfNeeded(v.MapIndex(key.(Valuable).ValueOf()).Interface())
	case reflect.Array, reflect.Slice, reflect.String:
		i := EnsureObjectIsInt(key, "")
		return true, MakeGoObjectIfNeeded(v.Index(i.I).Interface())
	}
	panic(fmt.Sprintf("Unsupported type=%T kind=%s for getting", o, reflect.TypeOf(o).Kind().String()))
}

func GoObjectCount(o interface{}) int {
	defer func() {
		if r := recover(); r != nil {
			panic(RT.NewError(fmt.Sprintf("%v", r)))
		}
	}()
	v := reflect.Indirect(reflect.ValueOf(o))
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len()
	case reflect.Struct:
		return v.NumField()
	}
	panic(fmt.Sprintf("Unsupported type=%T kind=%s for counting", o, reflect.TypeOf(o).Kind().String()))
}

func GoObjectSeq(o interface{}) Seq {
	defer func() {
		if r := recover(); r != nil {
			panic(RT.NewError(fmt.Sprintf("%v", r)))
		}
	}()
	v := reflect.Indirect(reflect.ValueOf(o))
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		n := v.Len()
		elements := make([]Object, n)
		for i := 0; i < n; i++ {
			elements[i] = MakeGoObjectIfNeeded(v.Index(i).Interface())
		}
		return &ArraySeq{arr: elements}
	case reflect.Struct:
		n := v.NumField()
		ty := v.Type()
		elements := make([]Object, n)
		for i := 0; i < n; i++ {
			elements[i] = NewVectorFrom(MakeKeyword(ty.Field(i).Name), MakeGoObjectIfNeeded(v.Field(i).Interface()))
		}
		return &ArraySeq{arr: elements}
	}
	panic(fmt.Sprintf("Unsupported type=%T kind=%s for sequencing", o, reflect.TypeOf(o).Kind().String()))
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

func ExtractarrayOfByte(args []Object, index int) []byte {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case []byte:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[]byte]"))
}

func Extractarray4OfByte(args []Object, index int) [4]byte {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case [4]byte:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[4]byte]"))
}

func ExtractarrayOfarrayOfByte(args []Object, index int) [][]byte {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case [][]byte:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[][]byte]"))
}

func ExtractarrayOfarray32OfByte(args []Object, index int) [][32]byte {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case [][32]byte:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[][32]byte]"))
}

func ConvertToarrayOfByte(o Object) []byte {
	switch obj := o.(type) {
	case String:
		return []byte(obj.S)
	case *Vector:
		vec := make([]byte, obj.Count())
		for i := 0; i < obj.Count(); i++ {
			el := obj.Nth(i)
			if val, ok := el.(Int); ok {
				b := val.I
				if b >= 0 && b <= 255 {
					vec[i] = byte(b)
				} else {
					panic(RT.NewError(fmt.Sprintf("Element %d out of range (%d) for Byte: %s", i, b, obj.ToString(false))))
				}
			} else {
				panic(RT.NewError(fmt.Sprintf("Element %d not convertible to Byte: %s", i, el.ToString(true))))
			}
		}
		return vec
	case Native:
		switch g := obj.Native().(type) {
		case []byte:
			return g
		default:
			panic(RT.NewError(fmt.Sprintf("Not an array of byte: %T", g)))
		}
	default:
		panic(RT.NewError(fmt.Sprintf("Not convertible to array of byte: %s", obj.ToString(true))))
	}
}

func ExtractarrayOfInt(args []Object, index int) []int {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case []int:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[]int]"))
}

func ConvertToarrayOfInt(o Object) []int {
	switch obj := o.(type) {
	case *Vector:
		vec := make([]int, obj.Count())
		for i := 0; i < obj.Count(); i++ {
			el := obj.Nth(i)
			if val, ok := el.(Int); ok {
				v := val.I
				if v >= MIN_INT && v <= MAX_INT {
					vec[i] = v
				} else {
					panic(RT.NewError(fmt.Sprintf("Element %d out of range (%d) for Int: %s", i, v, obj.ToString(false))))
				}
			} else {
				panic(RT.NewError(fmt.Sprintf("Element %d not convertible to Int: %s", i, el.ToString(true))))
			}
		}
		return vec
	case Native:
		switch g := obj.Native().(type) {
		case []int:
			return g
		default:
			panic(RT.NewError(fmt.Sprintf("Not an array of int: %T", g)))
		}
	default:
		panic(RT.NewError(fmt.Sprintf("Not convertible to array of int: %s", obj.ToString(true))))
	}
}

func ExtractarrayOfUint16(args []Object, index int) []uint16 {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case []uint16:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[]uint16]"))
}

func ExtractarrayOfUintPtr(args []Object, index int) []uintptr {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case []uintptr:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[]uintptr]"))
}

func ExtractarrayOfDouble(args []Object, index int) []float64 {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case []float64:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[]float64]"))
}

func ExtractarrayOfChar(args []Object, index int) []rune {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case []rune:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[]rune]"))
}

func ExtractarrayOfString(args []Object, index int) []string {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case []string:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[]string]"))
}

func ExtractarrayOfarrayOfString(args []Object, index int) [][]string {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case [][]string:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[][]string]"))
}

func ConvertToarrayOfString(o Object) []string {
	switch obj := o.(type) {
	case *Vector:
		vec := make([]string, obj.Count())
		for i := 0; i < obj.Count(); i++ {
			el := obj.Nth(i)
			if val, ok := el.(String); ok {
				v := val.S
				vec[i] = v
			} else {
				panic(RT.NewError(fmt.Sprintf("Element %d not convertible to String: %s", i, el.ToString(true))))
			}
		}
		return vec
	case Native:
		switch g := obj.Native().(type) {
		case []string:
			return g
		default:
			panic(RT.NewError(fmt.Sprintf("Not an array of string: %T", g)))
		}
	default:
		panic(RT.NewError(fmt.Sprintf("Not convertible to array of string: %s", obj.ToString(true))))
	}
}
