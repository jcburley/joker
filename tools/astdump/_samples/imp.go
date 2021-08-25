package main

import (
	"fmt"
)

type I interface {
	DoSomething()
}

type T struct {
	I
	n int
}

type i struct {
}

func (x i) DoSomething() {
	fmt.Println("here I am!")
}

func (o T) DoThis() {
	fmt.Println("well, hi!")
}

func main() {
	var obj I = T{I: i{}}
	o := obj.(T)
	fmt.Printf("o: %+v\n", o)
	fmt.Printf("obj: %+v\n", obj)
	o.DoThis()
	obj.DoSomething()
}
