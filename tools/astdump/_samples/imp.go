package main

import (
	"fmt"
)

type I interface {
	DoSomething()
	DoOther()
	W
}

type W interface {
	DoOther()
}

type T struct {
	I
	n int
}

type i struct {
}

type w struct {
}

func (x i) DoSomething() {
	fmt.Println("here I am!")
}

func (x i) DoOther() {
	fmt.Println("generic other thing!")
}

func (o T) DoThis() {
	fmt.Println("well, hi!")
}

func (o T) DoOther() {
	fmt.Println("really other thing!")
}

func (w w) DoOther() {
	fmt.Println("wild other thing!")
}

func main() {
	var obj I = T{I: i{}}
	o := obj.(T)
	fmt.Printf("o: %+v\n", o)
	fmt.Printf("obj: %+v\n", obj)
	o.DoThis()
	o.DoSomething()
	o.DoOther()
}
