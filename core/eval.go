package core

import (
	"bytes"
	"fmt"
	"go/ast"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

type (
	Traceable interface {
		Name() string
		Pos() Position
	}
	EvalError struct {
		msg  string
		pos  Position
		rt   *Runtime
		hash uint32
	}
	Frame struct {
		traceable Traceable
	}
	Callstack struct {
		frames []Frame
	}
	Runtime struct {
		callstack   *Callstack
		currentExpr Expr
		GIL         sync.Mutex
	}
)

var RT *Runtime = &Runtime{
	callstack: &Callstack{frames: make([]Frame, 0, 50)},
}

func (rt *Runtime) clone() *Runtime {
	return &Runtime{
		callstack:   rt.callstack.clone(),
		currentExpr: rt.currentExpr,
	}
}

func (rt *Runtime) NewError(msg string) *EvalError {
	res := &EvalError{
		msg: msg,
		rt:  rt.clone(),
	}
	if rt.currentExpr != nil {
		res.pos = rt.currentExpr.Pos()
	}
	return res
}

func (rt *Runtime) NewArgTypeError(index int, obj Object, expectedType string) *EvalError {
	name := rt.currentExpr.(Traceable).Name()
	return rt.NewError(fmt.Sprintf("Arg[%d] of %s must have type %s, got %s", index, name, expectedType, obj.GetType().ToString(false)))
}

func (rt *Runtime) NewReceiverArgTypeError(index int, name, rcvr string, obj Object, expectedType string) *EvalError {
	return rt.NewError(fmt.Sprintf("Arg[%d] (%s) of %s must have type %s, got %s", index, name, rcvr, expectedType, obj.TypeToString(false)))
}

func (rt *Runtime) NewErrorWithPos(msg string, pos Position) *EvalError {
	return &EvalError{
		msg: msg,
		pos: pos,
		rt:  rt.clone(),
	}
}

func (rt *Runtime) stacktrace() string {
	var b bytes.Buffer
	pos := Position{}
	if rt.currentExpr != nil {
		pos = rt.currentExpr.Pos()
	}
	name := "global"
	for _, f := range rt.callstack.frames {
		framePos := f.traceable.Pos()
		b.WriteString(fmt.Sprintf("  %s %s:%d:%d\n", name, framePos.Filename(), framePos.startLine, framePos.startColumn))
		name = f.traceable.Name()
		if strings.HasPrefix(name, "#'") {
			name = name[2:]
		}
	}
	b.WriteString(fmt.Sprintf("  %s %s:%d:%d", name, pos.Filename(), pos.startLine, pos.startColumn))
	return b.String()
}

func (rt *Runtime) pushFrame() {
	// TODO: this is all wrong. We cannot rely on
	// currentExpr for stacktraces. Instead, each Callable
	// should know it's name / position.
	var tr Traceable
	if rt.currentExpr != nil {
		tr = rt.currentExpr.(Traceable)
	} else {
		tr = &CallExpr{}
	}
	rt.callstack.pushFrame(Frame{traceable: tr})
}

func (rt *Runtime) popFrame() {
	rt.callstack.popFrame()
}

func Eval(expr Expr, env *LocalEnv) Object {
	parentExpr := RT.currentExpr
	RT.currentExpr = expr
	defer (func() { RT.currentExpr = parentExpr })()
	return expr.Eval(env)
}

func EvalMaybeAutoDeref(expr Expr, env *LocalEnv, doAutoDeref bool) Object {
	parentExpr := RT.currentExpr
	RT.currentExpr = expr
	defer (func() { RT.currentExpr = parentExpr })()
	if v, yes := expr.(*VarRefExpr); yes {
		return v.vr.Resolve(doAutoDeref)
	}
	return expr.Eval(env)
}

func (s *Callstack) pushFrame(frame Frame) {
	s.frames = append(s.frames, frame)
}

func (s *Callstack) popFrame() {
	s.frames = s.frames[:len(s.frames)-1]
}

func (s *Callstack) clone() *Callstack {
	res := &Callstack{frames: make([]Frame, len(s.frames))}
	copy(res.frames, s.frames)
	return res
}

func (s *Callstack) String() string {
	var b bytes.Buffer
	for _, f := range s.frames {
		pos := f.traceable.Pos()
		b.WriteString(fmt.Sprintf("%s %s:%d:%d\n", f.traceable.Name(), pos.Filename(), pos.startLine, pos.startColumn))
	}
	if b.Len() > 0 {
		b.Truncate(b.Len() - 1)
	}
	return b.String()
}

func MakeEvalError(msg string, pos Position, rt *Runtime) *EvalError {
	res := &EvalError{msg, pos, rt, 0}
	res.hash = HashPtr(uintptr(unsafe.Pointer(res)))
	return res
}

func (err *EvalError) ToString(escape bool) string {
	return err.Error()
}

func (err *EvalError) TypeToString(escape bool) string {
	return err.GetType().ToString(escape)
}

func (err *EvalError) Equals(other interface{}) bool {
	return err == other
}

func (err *EvalError) GetInfo() *ObjectInfo {
	return nil
}

func (err *EvalError) GetType() *Type {
	return TYPE.EvalError
}

func (err *EvalError) Hash() uint32 {
	return err.hash
}

func (err *EvalError) WithInfo(info *ObjectInfo) Object {
	return err
}

func (err *EvalError) Message() Object {
	return MakeString(err.msg)
}

func (err *EvalError) Error() string {
	pos := err.pos
	if len(err.rt.callstack.frames) > 0 && !LINTER_MODE {
		return fmt.Sprintf("%s:%d:%d: Eval error: %s\nStacktrace:\n%s", pos.Filename(), pos.startLine, pos.startColumn, err.msg, err.rt.stacktrace())
	} else {
		if len(err.rt.callstack.frames) > 0 {
			pos = err.rt.callstack.frames[0].traceable.Pos()
		}
		return fmt.Sprintf("%s:%d:%d: Eval error: %s", pos.Filename(), pos.startLine, pos.startColumn, err.msg)
	}
}

func (expr *VarRefExpr) Eval(env *LocalEnv) Object {
	return expr.vr.Resolve(true)
}

func (expr *SetMacroExpr) Eval(env *LocalEnv) Object {
	expr.vr.isMacro = true
	expr.vr.isUsed = false
	if fn, ok := expr.vr.Value.(*Fn); ok {
		fn.isMacro = true
	}
	setMacroMeta(expr.vr)
	return expr.vr
}

func (expr *BindingExpr) Eval(env *LocalEnv) Object {
	for i := env.frame; i > expr.binding.frame; i-- {
		env = env.parent
	}
	return env.bindings[expr.binding.index]
}

func (expr *LiteralExpr) Eval(env *LocalEnv) Object {
	return expr.obj
}

func (expr *VectorExpr) Eval(env *LocalEnv) Object {
	res := EmptyArrayVector()
	for _, e := range expr.v {
		res.Append(Eval(e, env))
	}
	return res
}

func (expr *MapExpr) Eval(env *LocalEnv) Object {
	if int64(len(expr.keys)) > HASHMAP_THRESHOLD/2 {
		res := EmptyHashMap
		for i := range expr.keys {
			key := Eval(expr.keys[i], env)
			if res.containsKey(key) {
				panic(RT.NewError("Duplicate key: " + key.ToString(false)))
			}
			res = res.Assoc(key, Eval(expr.values[i], env)).(*HashMap)
		}
		return res
	}
	res := EmptyArrayMap()
	for i := range expr.keys {
		key := Eval(expr.keys[i], env)
		if !res.Add(key, Eval(expr.values[i], env)) {
			panic(RT.NewError("Duplicate key: " + key.ToString(false)))
		}
	}
	return res
}

func (expr *SetExpr) Eval(env *LocalEnv) Object {
	res := EmptySet()
	for _, elemExpr := range expr.elements {
		el := Eval(elemExpr, env)
		if !res.Add(el) {
			panic(RT.NewError("Duplicate set element: " + el.ToString(false)))
		}
	}
	return res
}

func (expr *DefExpr) Eval(env *LocalEnv) Object {
	if expr.value != nil {
		expr.vr.Value = Eval(expr.value, env)
	}
	meta := EmptyArrayMap()
	meta.Add(KEYWORDS.line, Int{I: expr.startLine})
	meta.Add(KEYWORDS.column, Int{I: expr.startColumn})
	meta.Add(KEYWORDS.file, String{S: *expr.filename})
	meta.Add(KEYWORDS.ns, expr.vr.ns)
	meta.Add(KEYWORDS.name, expr.vr.name)
	expr.vr.meta = meta
	if expr.meta != nil {
		expr.vr.meta = expr.vr.meta.Merge(Eval(expr.meta, env).(Map))
	}
	// isMacro can be set by set-macro__ during parse stage
	if expr.vr.isMacro {
		expr.vr.meta = expr.vr.meta.Assoc(KEYWORDS.macro, Boolean{B: true}).(Map)
	}
	return expr.vr
}

func (expr *MetaExpr) Eval(env *LocalEnv) Object {
	meta := Eval(expr.meta, env)
	res := Eval(expr.expr, env)
	return res.(Meta).WithMeta(meta.(Map))
}

func evalSeq(exprs []Expr, env *LocalEnv) []Object {
	res := make([]Object, len(exprs))
	for i, expr := range exprs {
		res[i] = Eval(expr, env)
	}
	return res
}

func (expr *CallExpr) Eval(env *LocalEnv) Object {
	callable := Eval(expr.callable, env)
	switch callable := callable.(type) {
	case Callable:
		args := evalSeq(expr.args, env)
		return callable.Call(args)
	default:
		panic(RT.NewErrorWithPos(callable.ToString(false)+" is not a Fn", expr.callable.Pos()))
	}
}

func varCallableString(v *Var) string {
	if v.ns == GLOBAL_ENV.CoreNamespace {
		return "core/" + v.name.ToString(false)
	}
	return v.ns.Name.ToString(false) + "/" + v.name.ToString(false)
}

func (expr *CallExpr) Name() string {
	switch c := expr.callable.(type) {
	case *VarRefExpr:
		return varCallableString(c.vr)
	case *BindingExpr:
		return c.binding.name.ToString(false)
	case *LiteralExpr:
		return c.obj.ToString(false)
	default:
		return "fn"
	}
}

func (expr *ThrowExpr) Eval(env *LocalEnv) Object {
	e := Eval(expr.e, env)
	switch e.(type) {
	case Error:
		panic(e)
	default:
		panic(RT.NewError("Cannot throw " + e.ToString(false)))
	}
}

func (expr *TryExpr) Eval(env *LocalEnv) (obj Object) {
	defer func() {
		defer func() {
			if expr.finallyExpr != nil {
				evalBody(expr.finallyExpr, env)
			}
		}()
		if r := recover(); r != nil {
			switch r := r.(type) {
			case Error:
				for _, catchExpr := range expr.catches {
					if IsInstance(catchExpr.excType, r) {
						obj = evalBody(catchExpr.body, env.addFrame([]Object{r}))
						return
					}
				}
				panic(r)
			default:
				panic(r)
			}
		}
	}()
	return evalBody(expr.body, env)
}

func (expr *CatchExpr) Eval(env *LocalEnv) (obj Object) {
	panic(RT.NewError("This should never happen!"))
}

func (expr *DotExpr) tryReceiver(env *LocalEnv, o GoObject, g *Type, member string, isField bool) (obj Object, ok bool) {
	f := g.members[member]
	if f != nil {
		// Currently only receivers/methods are listed in Members[].
		if isField {
			panic(RT.NewError(fmt.Sprintf("Not a field: %s", member)))
		}
		defer func() {
			if r := recover(); r != nil {
				switch r.(type) {
				case Error:
					panic(r)
				default:
					panic(RT.NewError(fmt.Sprintf("method/receiver invocation panic: %s", r)))
				}
			}
		}()
		var args Object = NIL
		if expr.args != nil && len(expr.args) > 0 {
			args = &ArraySeq{arr: evalSeq(expr.args, env)}
		}
		return (f.Value.(GoReceiver))(o, args), true
	}
	return NIL, false
}

func (expr *DotExpr) Eval(env *LocalEnv) (obj Object) {
	instance := EvalMaybeAutoDeref(expr.instance, env, !expr.isSetNow)
	memberSym := expr.member
	member := *memberSym.name

	isField := expr.isSetNow
	if len(member) > 0 && member[0] == '-' {
		isField = true
		member = member[1:]
	}

	o := EnsureObjectIsGoObject(instance, "")
	g, ok := LookupGoType(o.O).(*Type)
	if g == nil || !ok {
		panic(RT.NewError("Unsupported type " + o.TypeToString(false)))
	}
	if obj, ok := expr.tryReceiver(env, o, g, member, isField); ok {
		return obj
	}

	// Try exchanging a pointer with a value to access corresponding receivers.

	var o2 GoObject

	v := reflect.ValueOf(o.O)
	k := v.Kind()
	if k == reflect.Ptr {
		v = v.Elem()
		k = v.Kind()
		if v.CanInterface() {
			o2 = MakeGoObject(v.Interface())
		}
	} else if v.CanAddr() {
		v2 := v.Addr()
		if v2.CanInterface() {
			o2 = MakeGoObject(v2.Interface())
		}
	}

	if o2.O != nil {
		g2, ok := LookupGoType(o2.O).(*Type)
		if ok && g2 != nil {
			if obj, ok := expr.tryReceiver(env, o2, g2, member, isField); ok {
				return obj
			}
		}
	}

	// Still no receiver, so must be a field reference within a struct.

	if k != reflect.Struct {
		panic(RT.NewError(fmt.Sprintf("No such receiver/method %s/%s", o.TypeToString(false), member)))
	}
	field := v.FieldByName(member)
	if field.Kind() == reflect.Invalid {
		panic(RT.NewError(fmt.Sprintf("No such member (field) %s/%s", o.TypeToString(false), member)))
	}
	if !expr.isSetNow {
		if field.CanInterface() {
			return MakeGoObjectIfNeeded(field.Interface())
		}
		why := ""
		if !ast.IsExported(member) {
			why = "unexported "
		}
		panic(RT.NewError(fmt.Sprintf("Cannot obtain value of %sfield %s/%s", why, o.TypeToString(false), member)))
	}

	if len(expr.args) != 1 {
		panic(RT.NewError(fmt.Sprintf("Field %s cannot be set to multiple arguments", member)))
	}
	if !field.CanSet() {
		panic(RT.NewError(fmt.Sprintf("Not a settable value, but rather a %s: %s", field.Kind(), field.Type())))
	}
	arg := Eval(expr.args[0], env)
	exprValue, ok := arg.(Valuable)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Cannot obtain Value of %s (not a Valuable)", arg.TypeToString(false))))
	}
	exprVal := exprValue.ValueOf()
	if !exprVal.Type().AssignableTo(field.Type()) {
		if !reflect.Indirect(exprVal).Type().AssignableTo(field.Type()) {
			panic(RT.NewError(fmt.Sprintf("Cannot assign a %s to a %s", exprVal.Type(), field.Type())))
		}
		exprVal = reflect.Indirect(exprVal)
	}
	field.Set(exprVal)
	return arg
}

func (expr *DotExpr) Name() string {
	return "(." + *expr.member.name + ")"
}

func (expr *SetNowExpr) Eval(env *LocalEnv) (obj Object) {
	targetVar := expr.target.(*Var)
	val := targetVar.Value
	if val == nil {
		panic(RT.NewError(fmt.Sprintf("Unbound var %s cannot be set!", targetVar.Name())))
	}
	targetExpr := EnsureObjectIsValuable(val, "")
	target := targetExpr.ValueOf()
	valueExpr := Eval(expr.value, env)
	value := EnsureObjectIsValuable(valueExpr, "").ValueOf()

	if target.Kind() == reflect.Ptr {
		target = reflect.Indirect(target)
	}
	if !target.CanSet() {
		panic(RT.NewError(fmt.Sprintf("Not a set!-able value: %s (%s)", targetVar.Name(), target.Kind())))
	}

	if !value.Type().AssignableTo(target.Type()) {
		if !reflect.Indirect(value).Type().AssignableTo(target.Type()) {
			panic(RT.NewError(fmt.Sprintf("Cannot set! %s (%s) to value of type %s", targetVar.Name(), target.Type(), value.Type())))
		}
		value = reflect.Indirect(value)
	}

	target.Set(value)
	return valueExpr
}

func evalBody(body []Expr, env *LocalEnv) Object {
	var res Object = NIL
	for _, expr := range body {
		res = Eval(expr, env)
	}
	return res
}

func evalLoop(body []Expr, env *LocalEnv) Object {
	var res Object = NIL
loop:
	for _, expr := range body {
		res = Eval(expr, env)
	}
	switch res := res.(type) {
	default:
		return res
	case RecurBindings:
		env = env.replaceFrame(res)
		goto loop
	}
}

func (doExpr *DoExpr) Eval(env *LocalEnv) Object {
	return evalBody(doExpr.body, env)
}

func (expr *IfExpr) Eval(env *LocalEnv) Object {
	if ToBool(Eval(expr.cond, env)) {
		return Eval(expr.positive, env)
	}
	return Eval(expr.negative, env)
}

func (expr *FnExpr) Eval(env *LocalEnv) Object {
	res := &Fn{fnExpr: expr}
	if expr.self.name != nil {
		env = env.addFrame([]Object{res})
	}
	res.env = env
	return res
}

func (expr *FnArityExpr) Eval(env *LocalEnv) Object {
	panic(RT.NewError("This should never happen!"))
}

func (expr *LetExpr) Eval(env *LocalEnv) Object {
	env = env.addEmptyFrame(len(expr.names))
	for _, bindingExpr := range expr.values {
		env.addBinding(Eval(bindingExpr, env))
	}
	return evalBody(expr.body, env)
}

func (expr *LoopExpr) Eval(env *LocalEnv) Object {
	env = env.addEmptyFrame(len(expr.names))
	for _, bindingExpr := range expr.values {
		env.addBinding(Eval(bindingExpr, env))
	}
	return evalLoop(expr.body, env)
}

func (expr *RecurExpr) Eval(env *LocalEnv) Object {
	return RecurBindings(evalSeq(expr.args, env))
}

func (expr *MacroCallExpr) Eval(env *LocalEnv) Object {
	return expr.macro.Call(expr.args)
}

func (expr *MacroCallExpr) Name() string {
	return expr.name
}

func TryEval(expr Expr) (obj Object, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case *EvalError:
				err = r.(error)
			case *ExInfo:
				err = r.(error)
			default:
				panic(fmt.Sprintf("Unrecoverable error %s handling %s:%+v", strconv.Quote(fmt.Sprintf("%s", r)), AnyTypeToString(expr, false), expr))
			}
		}
	}()
	return Eval(expr, nil), nil
}

func PanicOnErr(err error) {
	if err != nil {
		panic(RT.NewError(err.Error()))
	}
}
