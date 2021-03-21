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

var v_2 = &BasicLit{
	Kind:  token.INT,
	Value: "2",
}

var two = &ValueSpec{
	Names:  []*Ident{&Ident{Name: "two"}},
	Type:   &Ident{Name: "int8"},
	Values: []Expr{v_2},
}

var cfg = &Config{}

func TestEvalOne(t *testing.T) {
	ts, val := cfg.Eval(one.Type, one.Values[0])
	if ts == nil || ts.(*Ident).Name != "int" {
		t.Errorf("ts==%+v", ts)
	}
	if val.(int) != 1 {
		t.Errorf("val==%+v", val)
	}
}

func TestEvalTwo(t *testing.T) {
	ts, val := cfg.Eval(two.Type, two.Values[0])
	if ts == nil || ts.(*Ident).Name != "int8" {
		t.Errorf("ts==%+v", ts)
	}
	if val.(int) != 2 {
		t.Errorf("val==%+v", val)
	}
}
