package eval

import (
	. "go/ast"
	"go/token"
	"testing"
)

var v_1 = &BasicLit{
	Kind:  token.INT,
	Value: "1",
}

var one = &ValueSpec{
	Names:  []*Ident{&Ident{Name: "one"}},
	Type:   nil,
	Values: []Expr{v_1},
}

var stringType = &Ident{
	Name: "string",
}

var intType = &Ident{
	Name: "int",
}

var floatType = &Ident{
	Name: "float",
}

var charType = &Ident{
	Name: "char",
}

var cfg = &Config{
	StringType: stringType,
	IntType:    intType,
	FloatType:  floatType,
	CharType:   charType,
}

func TestEval(t *testing.T) {
	ts, val := cfg.Eval(one.Type, one.Values[0])
	if ts == nil || ts.(*Ident).Name != "int" {
		t.Errorf("ts==%+v", ts)
	}
	if val.(int) != 1 {
		t.Errorf("val==%+v", val)
	}
}
