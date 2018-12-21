package core

import (
	"fmt"
)

func ConvertToArrayOfByte(o Object) []byte {
	switch obj := o.(type) {
	case String:
		return []byte(obj.S)
	case *Vector:
		by := make([]byte, obj.Count())
		for i := 0; i < obj.Count(); i++ {
			el := obj.Nth(i)
			if val, ok := el.(Int); ok {
				b := val.I
				if b >= 0 && b <= 255 {
					by[i] = byte(b)
				} else {
					panic(RT.NewError(fmt.Sprintf("Element %d out of range (%d) for Byte: %s", i, b, obj.ToString(false))))
				}
			} else {
				panic(RT.NewError(fmt.Sprintf("Element %d not convertible to Byte: %s", i, el.ToString(true))))
			}
		}
		return by
	default:
		panic(RT.NewError(fmt.Sprintf("Not convertible to array of Byte: %s", obj.ToString(true))))
	}
}

func ConvertToArrayOfInt(o Object) []int {
	switch o.(type) {
	case *Vector:
		return []int{123, 456}
	default:
		return []int{789, 101112}
	}
}

func ConvertToArrayOfString(o Object) []string {
	switch o.(type) {
	case *Vector:
	default:
	}
	return []string{"NOT", "YET"}
}

func ConvertToArrayOfObject(o Object) []Object {
	return []Object{}
}
