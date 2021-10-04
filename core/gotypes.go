package core

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
)

type GoMembers map[string]*Var

func MakeType(name string, ctor Ctor, mem GoMembers) Type {
	return Type{name: name, ctor: ctor, members: mem}
}

func LookupGoType(g interface{}) interface{} {
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

func CheckReceiverArity(rcvr string, args Object, min, max int) *ArraySeq {
	n := 0
	switch s := args.(type) {
	case Nil:
		if min == 0 {
			return nil
		}
	case *ArraySeq:
		n = SeqCount(s)
		if n >= min && n <= max {
			return s
		}
	default:
	}
	panic(RT.NewError(fmt.Sprintf("Wrong number of args (%d) passed to %s; expects %s", n, rcvr, RangeString(min, max))))
}

func ObjectAs_bool(obj Object, pattern string) bool {
	return EnsureObjectIsBoolean(obj, pattern).B
}

func ReceiverArgAs_bool(name, rcvr string, args *ArraySeq, n int) bool {
	a := SeqNth(args, n)
	res, ok := a.(Boolean)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type Boolean, but is %s",
			n, name, rcvr, a.TypeToString(false))))
	}
	return res.B
}

func FieldAs_bool(o Map, k string) bool {
	ok, v := o.Get(MakeKeyword(k))
	if !ok {
		return false
	}
	res, ok := v.(Boolean)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type Boolean, but is %s",
			k, v.TypeToString(false))))
	}
	return res.B
}

func ObjectAs_int(obj Object, pattern string) int {
	return EnsureObjectIsInt(obj, pattern).I
}

func ReceiverArgAs_int(name, rcvr string, args *ArraySeq, n int) int {
	a := SeqNth(args, n)
	res, ok := a.(Int)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type Int, but is %s",
			n, name, rcvr, a.TypeToString(false))))
	}
	return res.I
}

func FieldAs_int(o Map, k string) int {
	v := FieldAsNumber(o, k).BigInt().Int64()
	if v > int64(MAX_INT) || v < int64(MIN_INT) {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type int, but is too large",
			v, k)))
	}
	return int(v)
}

func ObjectAs_uint(obj Object, pattern string) uint {
	v := EnsureObjectIsNumber(obj, pattern).BigInt().Uint64()
	if v > uint64(MAX_UINT) {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.FormatUint(uint64(v), 10)+" out of range for uint")))
	}
	return uint(v)
}

func ReceiverArgAs_uint(name, rcvr string, args *ArraySeq, n int) uint {
	v := ReceiverArgAsNumber(name, rcvr, args, n).BigInt().Uint64()
	if v > uint64(MAX_UINT) {
		panic(RT.NewArgTypeError(n, SeqNth(args, n), "uint"))
	}
	return uint(v)
}

func FieldAs_uint(o Map, k string) uint {
	v := FieldAsNumber(o, k).BigInt().Uint64()
	if v > uint64(MAX_UINT) {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type uint, but is too large",
			v, k)))
	}
	return uint(v)
}

func ReceiverArgAs_uint8(name, rcvr string, args *ArraySeq, n int) uint8 {
	v := ReceiverArgAs_int(name, rcvr, args, n)
	if v < 0 || v > 255 {
		panic(RT.NewArgTypeError(n, SeqNth(args, n), "uint8"))
	}
	return byte(v)
}

func ReceiverArgAsNumber(name, rcvr string, args *ArraySeq, n int) Number {
	a := SeqNth(args, n)
	res, ok := a.(Number)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type Number, but is %s",
			n, name, rcvr, a.TypeToString(false))))
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
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type Number, but is %s",
			k, v.TypeToString(false))))
	}
	return res
}

func ObjectAs_int8(obj Object, pattern string) int8 {
	v := EnsureObjectIsInt(obj, pattern).I
	if v > math.MaxInt8 {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.Itoa(v)+" out of range for int8")))
	}
	return int8(v)
}

func FieldAs_int8(o Map, k string) int8 {
	v := FieldAsNumber(o, k).BigInt().Int64()
	if v > math.MaxInt8 || v < math.MinInt8 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type int8, but its magnitude is too large",
			v, k)))
	}
	return int8(v)
}

func ObjectAs_int16(obj Object, pattern string) int16 {
	v := EnsureObjectIsInt(obj, pattern).I
	if v > math.MaxInt16 {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.Itoa(v)+" out of range for int16")))
	}
	return int16(v)
}

func FieldAs_int16(o Map, k string) int16 {
	v := FieldAsNumber(o, k).BigInt().Int64()
	if v > math.MaxInt16 || v < math.MinInt16 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type int16, but its magnitude is too large",
			v, k)))
	}
	return int16(v)
}

func ReceiverArgAs_uint16(name, rcvr string, args *ArraySeq, n int) uint16 {
	v := ReceiverArgAsNumber(name, rcvr, args, n).BigInt().Int64()
	if v > math.MaxUint16 || v < 0 {
		panic(RT.NewArgTypeError(n, SeqNth(args, n), "uint16"))
	}
	return uint16(v)
}

func ObjectAs_int32(obj Object, pattern string) int32 {
	v := EnsureObjectIsInt(obj, pattern).I
	if v > math.MaxInt32 {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.Itoa(v)+" out of range for int32")))
	}
	return int32(v)
}

func ReceiverArgAs_int32(name, rcvr string, args *ArraySeq, n int) int32 {
	v := ReceiverArgAsNumber(name, rcvr, args, n).BigInt().Int64()
	if v > math.MaxInt32 || v < math.MinInt32 {
		panic(RT.NewArgTypeError(n, SeqNth(args, n), "int32"))
	}
	return int32(v)
}

func FieldAs_int32(o Map, k string) int32 {
	v := FieldAsNumber(o, k).BigInt().Int64()
	if v > math.MaxInt32 || v < math.MinInt32 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type int32, but its magnitude is too large",
			v, k)))
	}
	return int32(v)
}

func ObjectAs_uint8(obj Object, pattern string) uint8 {
	v := EnsureObjectIsInt(obj, pattern).I
	if v > math.MaxUint8 || v < 0 {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.Itoa(v)+" out of range for uint8")))
	}
	return uint8(v)
}

func FieldAs_uint8(o Map, k string) uint8 {
	v := FieldAsNumber(o, k).BigInt().Uint64()
	if v > math.MaxUint8 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type uint8, but is too large",
			v, k)))
	}
	return uint8(v)
}

func ObjectAs_uint16(obj Object, pattern string) uint16 {
	v := EnsureObjectIsInt(obj, pattern).I
	if v > math.MaxUint16 || v < 0 {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.Itoa(v)+" out of range for uint16")))
	}
	return uint16(v)
}

func FieldAs_uint16(o Map, k string) uint16 {
	v := FieldAsNumber(o, k).BigInt().Uint64()
	if v > math.MaxUint16 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type uint16, but is too large",
			v, k)))
	}
	return uint16(v)
}

func ObjectAs_uint32(obj Object, pattern string) uint32 {
	v := EnsureObjectIsNumber(obj, pattern).BigInt().Uint64()
	if v > math.MaxUint32 {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.FormatUint(uint64(v), 10)+" out of range for uint32")))
	}
	return uint32(v)
}

func ReceiverArgAs_uint32(name, rcvr string, args *ArraySeq, n int) uint32 {
	v := ReceiverArgAsNumber(name, rcvr, args, n).BigInt().Uint64()
	if v > math.MaxUint32 {
		panic(RT.NewArgTypeError(n, SeqNth(args, n), "uint32"))
	}
	return uint32(v)
}

func FieldAs_uint32(o Map, k string) uint32 {
	v := FieldAsNumber(o, k).BigInt().Uint64()
	if v > math.MaxUint32 {
		panic(RT.NewError(fmt.Sprintf("Value %v for key %s should be type uint32, but is too large",
			v, k)))
	}
	return uint32(v)
}

func ReceiverArgAs_int64(name, rcvr string, args *ArraySeq, n int) int64 {
	return ReceiverArgAsNumber(name, rcvr, args, n).BigInt().Int64()
}

func ObjectAs_int64(obj Object, pattern string) int64 {
	return EnsureObjectIsNumber(obj, pattern).BigInt().Int64()
}

func FieldAs_int64(o Map, k string) int64 {
	return FieldAsNumber(o, k).BigInt().Int64()
}

func ObjectAs_uint64(obj Object, pattern string) uint64 {
	return EnsureObjectIsNumber(obj, pattern).BigInt().Uint64()
}

func ReceiverArgAs_uint64(name, rcvr string, args *ArraySeq, n int) uint64 {
	return ReceiverArgAsNumber(name, rcvr, args, n).BigInt().Uint64()
}

func FieldAs_uint64(o Map, k string) uint64 {
	return FieldAsNumber(o, k).BigInt().Uint64()
}

func ObjectAs_uintptr(obj Object, pattern string) uintptr {
	v := EnsureObjectIsNumber(obj, pattern).BigInt().Uint64()
	if uint64(uintptr(v)) != v {
		panic(RT.NewError(fmt.Sprintf(pattern, "Number "+strconv.FormatUint(v, 10)+" out of range for uintptr")))
	}
	return uintptr(v)
}

func ReceiverArgAs_uintptr(name, rcvr string, args *ArraySeq, n int) uintptr {
	return uintptr(ReceiverArgAs_uint64(name, rcvr, args, n))
}

func FieldAs_uintptr(o Map, k string) uintptr {
	return uintptr(FieldAs_uint64(o, k))
}

func Extractfloat32(args []Object, index int) float32 {
	o := ExtractObject(args, index)
	if g, ok := o.(GoObject); ok {
		if f, ok := g.O.(float32); ok {
			return f
		}
	}
	return float32(EnsureArgIsDouble(args, index).D)
}

func ObjectAs_float64(obj Object, pattern string) float64 {
	return EnsureObjectIsDouble(obj, pattern).D
}

func FieldAs_float64(o Map, k string) float64 {
	ok, v := o.Get(MakeKeyword(k))
	if !ok {
		return 0
	}
	res, ok := v.(Double)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type Double, but is %s",
			k, v.TypeToString(false))))
	}
	return res.D
}

func ReceiverArgAs_float64(name, rcvr string, args *ArraySeq, n int) float64 {
	a := SeqNth(args, n)
	res, ok := a.(Double)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type Double, but is %s",
			n, name, rcvr, a.TypeToString(false))))
	}
	return res.D
}

func MaybeIs_complex128(o Object) (complex128, string) {
	if g, ok := o.(GoObject); ok {
		if res, ok := g.O.(complex128); ok {
			return res, ""
		}
	}
	return 0, "GoObject[complex128]"
}

func FieldAs_complex128(o Map, k string) complex128 {
	ok, v := o.Get(MakeKeyword(k))
	if !ok {
		return 0
	}
	res, sb := MaybeIs_complex128(v)
	if sb == "" {
		return res
	}
	panic(RT.NewError(fmt.Sprintf("Value for key %s should be type %s, but is %s",
		k, sb, v.TypeToString(false))))
}

func ReceiverArgAs_complex128(name, rcvr string, args *ArraySeq, n int) complex128 {
	a := SeqNth(args, n)
	res, sb := MaybeIs_complex128(a)
	if sb == "" {
		return res
	}
	panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type %s, but is %s",
		n, name, rcvr, sb, a.TypeToString(false))))
}

func ObjectAs_rune(obj Object, pattern string) rune {
	return EnsureObjectIsChar(obj, pattern).Ch
}

func ReceiverArgAs_rune(name, rcvr string, args *ArraySeq, n int) rune {
	a := SeqNth(args, n)
	res, ok := a.(Char)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type Char, but is %s",
			n, name, rcvr, a.TypeToString(false))))
	}
	return res.Ch
}

func FieldAs_rune(o Map, k string) rune {
	ok, v := o.Get(MakeKeyword(k))
	if !ok {
		return 0
	}
	res, ok := v.(Char)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type Char, but is %s",
			k, v.TypeToString(false))))
	}
	return res.Ch
}

func ObjectAs_string(obj Object, pattern string) string {
	return EnsureObjectIsString(obj, pattern).S
}

func ReceiverArgAs_string(name, rcvr string, args *ArraySeq, n int) string {
	a := SeqNth(args, n)
	res, sb := MaybeIsString(a)
	if sb == "" {
		return res.S
	}
	panic(RT.NewReceiverArgTypeError(n, name, rcvr, a, sb))
}

func ReceiverArgAs_strings(name, rcvr string, args *ArraySeq, n int) []string {
	vec := make([]string, 0)
	count := SeqCount(args)
	for i := n; i < count; i++ {
		vec = append(vec, ReceiverArgAs_string(name, rcvr, args, i))
	}
	return vec
}

func FieldAs_string(o Map, k string) string {
	ok, v := o.Get(MakeKeyword(k))
	if !ok {
		return ""
	}
	res, ok := v.(String)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type String, but is %s",
			k, v.TypeToString(false))))
	}
	return res.S
}

func MaybeIs_error(o Object) (error, string) {
	switch res := o.(type) {
	case String:
		return errors.New(res.S), ""
	case Error:
		return res, ""
	case GoObject:
		if g, ok := res.O.(error); ok {
			return g, ""
		}
	}
	return nil, "String or GoObject[error]"
}

// This would normally be autogenerated into types_assert_gen.go, but is a customized version.
func EnsureArgIs_error(args []Object, index int) error {
	obj := args[index]
	res, sb := MaybeIs_error(obj)
	if sb == "" {
		return res
	}
	panic(FailArg(obj, sb, index))
}

func EnsureArgImplements_error(args []Object, index int) Object {
	obj := args[index]
	_, sb := MaybeIs_error(obj)
	if sb == "" {
		return obj
	}
	panic(FailArg(obj, sb, index))
}

func ReceiverArgAs_error(name, rcvr string, args *ArraySeq, n int) error {
	a := SeqNth(args, n)
	res, sb := MaybeIs_error(a)
	if sb == "" {
		return res
	}
	panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be of type %s, but is %s",
		n, name, rcvr, sb, a.TypeToString(false))))
}

func FieldAs_error(o Map, k string) error {
	ok, v := o.Get(MakeKeyword(k))
	if !ok || v.Equals(NIL) {
		return nil
	}
	res, sb := MaybeIs_error(v)
	if sb == "" {
		return res
	}
	panic(RT.NewError(fmt.Sprintf("Value for key %s should be type %s, but is %s",
		k, sb, v.TypeToString(false))))
}

func ReceiverArgAs_GoObject(name, rcvr string, args *ArraySeq, n int) interface{} {
	a := SeqNth(args, n)
	res, sb := MaybeIsNative(a)
	if sb == "" {
		return res
	}
	panic(RT.NewReceiverArgTypeError(n, name, rcvr, a, sb))
}

func ReceiverArgAs_GoObjects(name, rcvr string, args *ArraySeq, n int) []interface{} {
	vec := make([]interface{}, 0)
	count := SeqCount(args)
	for i := n; i < count; i++ {
		vec = append(vec, ReceiverArgAs_GoObject(name, rcvr, args, i))
	}
	return vec
}

func FieldAs_GoObject(o Map, k string) interface{} {
	ok, v := o.Get(MakeKeyword(k))
	if !ok {
		return ""
	}
	res, ok := v.(GoObject)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Value for key %s should be type GoObject, but is %s",
			k, v.TypeToString(false))))
	}
	return res.O
}

func MaybeIsFn(o Object) (*Fn, string) {
	if res, yes := o.(*Fn); yes {
		return res, ""
	}
	return nil, "Fn"
}

func EnsureObjectIsFn(obj Object, pattern string) *Fn {
	res, sb := MaybeIsFn(obj)
	if sb == "" {
		return res
	}
	panic(FailObject(obj, sb, pattern))
}

func EnsureArgIsFn(args []Object, index int) *Fn {
	obj := args[index]
	res, sb := MaybeIsFn(obj)
	if sb == "" {
		return res
	}
	panic(FailArg(obj, sb, index))
}

func ReceiverArgAs_func(name, rcvr string, args *ArraySeq, n int) func() {
	a := SeqNth(args, n)
	res, sb := MaybeIs_func(a)
	if sb == "" {
		return res
	}
	panic(RT.NewReceiverArgTypeError(n, name, rcvr, a, sb))
}

func FieldAs_func(o Map, k string) func() {
	ok, v := o.Get(MakeKeyword(k))
	if !ok || v.Equals(NIL) {
		return nil
	}
	res, sb := MaybeIs_func(v)
	if sb == "" {
		return res
	}
	panic(RT.NewError(fmt.Sprintf("Value for key %s should be type %s, but is %s",
		k, sb, v.TypeToString(false))))
}

func buildFunc(fn *Fn) func() {
	var body []Expr
	min := math.MaxInt32
	max := -1
	for _, arity := range fn.fnExpr.arities {
		a := len(arity.args)
		if a == 0 {
			body = arity.body
		} else {
			if min > a {
				min = a
			}
			if max < a {
				max = a
			}
		}
	}
	if body == nil {
		v := fn.fnExpr.variadic
		if v == nil || 0 < len(v.args)-1 {
			if v != nil {
				min = len(v.args)
				max = math.MaxInt32
			}
			c := 0
			PanicArityMinMax(c, min, max)
		}
		body = v.body
	}
	return func() {
		RT.pushFrame()
		defer RT.popFrame()
		evalLoop(body, fn.env.addFrame(nil))
		return
	}
}

func MaybeIs_func(o Object) (func(), string) {
	if res, yes := o.(Native); yes {
		if f, yes := res.Native().(func()); yes {
			return f, ""
		}
	}
	if fn, sb := MaybeIsFn(o); sb == "" {
		if fn.fn == nil {
			fn.fn = buildFunc(fn)
		}
		if fn, yes := fn.fn.(func()); yes {
			return fn, ""
		}
	}
	return nil, "Fn or GoObject[func()]"
}

func Extractfunc(args []Object, index int) func() {
	obj := args[index]
	res, sb := MaybeIs_func(obj)
	if sb == "" {
		return res
	}
	panic(FailArg(obj, sb, index))
}

func ReceiverArgAsfunc(name, rcvr string, args *ArraySeq, n int) func() {
	a := SeqNth(args, n)
	res, sb := MaybeIs_func(a)
	if sb == "" {
		return res
	}
	panic(RT.NewReceiverArgTypeError(n, name, rcvr, a, sb))
}

func FieldAsfunc(o Map, k string) func() {
	ok, v := o.Get(MakeKeyword(k))
	if !ok || v.Equals(NIL) {
		return nil
	}
	res, sb := MaybeIs_func(v)
	if sb == "" {
		return res
	}
	panic(RT.NewError(fmt.Sprintf("Value for key %s should be of type %s, but is %s",
		k, sb, v.TypeToString(false))))
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
		if sym, ok := key.(Symbol); ok {
			f := v.FieldByName(sym.Name())
			if f.IsValid() {
				return true, MakeGoObjectIfNeeded(f.Interface())
			}
			return false, NIL
		}
		panic(fmt.Sprintf("Key must evaluate to a symbol, not type %T", key))
	case reflect.Map:
		if key.Equals(NIL) {
			// Special case for nil key (used during doc generation): return vector of keys as reflect.Value's
			return true, MakeGoObject(v.MapKeys())
		}
		if res := v.MapIndex(key.(Valuable).ValueOf()); res.IsValid() {
			return true, MakeGoObjectIfNeeded(res.Interface())
		}
		return false, NIL
	case reflect.Array, reflect.Slice, reflect.String:
		i := EnsureObjectIsInt(key, "")
		return func() (ok bool, obj Object) {
			defer func() {
				if r := recover(); r != nil {
					obj = NIL
					return
				}
			}()
			return true, MakeGoObjectIfNeeded(v.Index(i.I).Interface())
		}()
	}
	panic(fmt.Sprintf("Unsupported type=%s kind=%s for getting", AnyTypeToString(o, false), reflect.TypeOf(o).Kind().String()))
}

func GoObjectCount(o interface{}) int {
	defer func() {
		if r := recover(); r != nil {
			panic(RT.NewError(fmt.Sprintf("%v", r)))
		}
	}()
	v := reflect.Indirect(reflect.ValueOf(o))
	switch k := v.Kind(); k {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len()
	case reflect.Struct:
		return v.NumField()
	default:
		panic(fmt.Sprintf("Unsupported type=%s kind=%s for counting", AnyTypeToString(o, false), k))
	}
}

func GoObjectSeq(o interface{}) Seq {
	defer func() {
		if r := recover(); r != nil {
			panic(RT.NewError(fmt.Sprintf("%v", r)))
		}
	}()
	v := reflect.Indirect(reflect.ValueOf(o))
	switch k := v.Kind(); k {
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
	default:
		panic(fmt.Sprintf("Unsupported type=%s kind=%s for sequencing", AnyTypeToString(o, false), k))
	}
}

func MakeGoReceiver(name string, f func(GoObject, Object) Object, doc, added string, arglist *Vector) *Var {
	v := &Var{
		name:  MakeSymbol(name),
		Value: GoReceiver(f),
	}
	m := MakeMeta(NewListFrom(arglist), doc, added)
	m.Add(KEYWORDS.name, v.name)
	v.meta = m
	return v
}

func MaybeIs_arrayOfuint8(o Object) ([]uint8, string) {
	switch obj := o.(type) {
	case Native:
		switch r := obj.Native().(type) {
		case []uint8:
			return r, ""
		case string:
			return []uint8(r), ""
		}
	}
	return nil, "GoObject[[]uint8]"
}

func ExtractarrayOfuint8(args []Object, index int) []uint8 {
	a := args[index]
	res, sb := MaybeIs_arrayOfuint8(a)
	if sb == "" {
		return res
	}
	panic(FailArg(a, sb, index))
}

func ReceiverArgAs_arrayOfuint8(name, rcvr string, args *ArraySeq, n int) []uint8 {
	a := SeqNth(args, n)
	res, sb := MaybeIs_arrayOfuint8(a)
	if sb == "" {
		return res
	}
	panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s must be type %s, but is %s",
		n, name, rcvr, sb, a.TypeToString(false))))
}

func FieldAs_arrayOfuint8(o Map, k string) []uint8 {
	ok, v := o.Get(MakeKeyword(k))
	if !ok || v.Equals(NIL) {
		return []uint8{}
	}
	res, sb := MaybeIs_arrayOfuint8(v)
	if sb == "" {
		return res
	}
	panic(FailObject(v, sb, ""))
}

func Extractarray4Ofuint8(args []Object, index int) [4]uint8 {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case [4]uint8:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[4]uint8]"))
}

func ExtractarrayOfarrayOfuint8(args []Object, index int) [][]uint8 {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case [][]uint8:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[][]uint8]"))
}

func ExtractarrayOfarray32Ofuint8(args []Object, index int) [][32]uint8 {
	o := args[index]
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case [][32]uint8:
			return g
		}
	}
	panic(RT.NewArgTypeError(index, o, "GoObject[[][32]uint8]"))
}

func ReceiverArgAs_arrayOfarray32Ofuint8(name, rcvr string, args *ArraySeq, n int) [][32]uint8 {
	a := SeqNth(args, n)
	switch obj := a.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case [][32]uint8:
			return g
		}
	}
	panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type GoObject[[][32]uint8], but is %s",
		n, name, rcvr, a.TypeToString(false))))
}

func ConvertTo_arrayOfuint8(o Object) []uint8 {
	switch obj := o.(type) {
	case String:
		return []uint8(obj.S)
	case *Vector:
		vec := make([]uint8, obj.Count())
		for i := 0; i < obj.Count(); i++ {
			el := obj.Nth(i)
			if val, ok := el.(Int); ok {
				b := val.I
				if b >= 0 && b <= 255 {
					vec[i] = uint8(b)
				} else {
					panic(RT.NewError(fmt.Sprintf("Element %d out of range (%d) for Uint8: %s", i, b, obj.ToString(false))))
				}
			} else {
				panic(RT.NewError(fmt.Sprintf("Element %d not convertible to Uint8: %s", i, el.ToString(true))))
			}
		}
		return vec
	case Native:
		switch g := obj.Native().(type) {
		case []uint8:
			return g
		default:
			panic(RT.NewError(fmt.Sprintf("Not an array of uint8: %s", AnyTypeToString(g, false))))
		}
	default:
		panic(RT.NewError(fmt.Sprintf("Not convertible to array of uint8: %s", obj.ToString(true))))
	}
}

func ExtractarrayOfint(args []Object, index int) []int {
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

func ReceiverArgAs_arrayOfint(name, rcvr string, args *ArraySeq, n int) []int {
	a := SeqNth(args, n)
	switch obj := a.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case []int:
			return g
		}
	}
	panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type GoObject[[]int], but is %s",
		n, name, rcvr, a.TypeToString(false))))
}

func ConvertTo_arrayOfint(o Object) []int {
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
			panic(RT.NewError(fmt.Sprintf("Not an array of int: %s", AnyTypeToString(g, false))))
		}
	default:
		panic(RT.NewError(fmt.Sprintf("Not convertible to array of int: %s", obj.ToString(true))))
	}
}

func ExtractarrayOfuint16(args []Object, index int) []uint16 {
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

func ExtractarrayOfuintptr(args []Object, index int) []uintptr {
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

func ExtractarrayOffloat64(args []Object, index int) []float64 {
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

func ExtractarrayOfrune(args []Object, index int) []rune {
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

func MaybeIs_arrayOfstring(o Object) ([]string, string) {
	switch obj := o.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case []string:
			return g, ""
		}
	case *Vector:
		vec := make([]string, obj.Count())
		for i := 0; i < obj.Count(); i++ {
			el := obj.Nth(i)
			if val, ok := el.(String); ok {
				vec[i] = val.S
			} else {
				panic(RT.NewError(fmt.Sprintf("Element %d not convertible to String: %s", i, el.ToString(true))))
			}
		}
		return vec, ""
	}
	return nil, "Vector of String or GoObject[[]string]"
}

func ExtractarrayOfstring(args []Object, index int) []string {
	a := args[index]
	res, sb := MaybeIs_arrayOfstring(a)
	if sb == "" {
		return res
	}
	panic(FailArg(a, sb, index))
}

func ReceiverArgAs_arrayOfstring(name, rcvr string, args *ArraySeq, n int) []string {
	a := SeqNth(args, n)
	res, sb := MaybeIs_arrayOfstring(a)
	if sb == "" {
		return res
	}
	panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be of type %s, but is %s",
		n, name, rcvr, sb, a.TypeToString(false))))
}

func ExtractarrayOfarrayOfstring(args []Object, index int) [][]string {
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

func ReceiverArgAs_arrayOfarrayOfstring(name, rcvr string, args *ArraySeq, n int) [][]string {
	a := SeqNth(args, n)
	switch obj := a.(type) {
	case Native:
		switch g := obj.Native().(type) {
		case [][]string:
			return g
		}
	}
	panic(RT.NewError(fmt.Sprintf("Argument %d (%s) passed to %s should be type GoObject[[][]string], but is %s",
		n, name, rcvr, a.TypeToString(false))))
}

func ConvertTo_arrayOfstring(o Object) []string {
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
			panic(RT.NewError(fmt.Sprintf("Not an array of string: %s", AnyTypeToString(g, false))))
		}
	default:
		panic(RT.NewError(fmt.Sprintf("Not convertible to array of string: %s", obj.ToString(true))))
	}
}
