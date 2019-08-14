package core

import (
	"fmt"
	"reflect"
)

type GoFn func(GoObject, Object) Object

type GoMembers map[string]GoFn

type GoTypeInfo struct {
	Members GoMembers
}

func GoLookupType(g interface{}) *GoTypeInfo {
	return GoTypes[reflect.TypeOf(g)]
}

func GoCheckArity(rcvr string, args Object, min, max int) *ArraySeq {
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

func GoCheckNth(rcvr, t string, args *ArraySeq, n int) GoObject {
	a := SeqNth(args, n)
	res, ok := a.(GoObject)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d passed to %s should be type net.GoObject[%s], but is %T",
			n, rcvr, t, a)))
	}
	return res
}

func GoCheckStringNth(rcvr string, args *ArraySeq, n int) string {
	a := SeqNth(args, n)
	res, ok := a.(String)
	if !ok {
		panic(RT.NewError(fmt.Sprintf("Argument %d passed to %s should be type core.String, but is %T",
			n, rcvr, a)))
	}
	return res.S
}

var GoTypes map[reflect.Type]*GoTypeInfo = map[reflect.Type]*GoTypeInfo{}
