package core

import (
	"fmt"
)

/* IMPORTANT: The public functions defined herein should be listed in
   this set in gostd's main.go:

     var customRuntimeImplemented = map[string]struct{} {
     }

   That's how gostd knows to not actually generate calls to
   as-yet-unimplemented (or stubbed-out) functions, saving the
   developer the hassle of getting most of the way through a build
   before hitting undefined-func errors.

*/

func ConvertToArrayOfByte(o Object) []byte {
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
	default:
		panic(RT.NewError(fmt.Sprintf("Not convertible to array of Byte: %s", obj.ToString(true))))
	}
}

func ConvertToArrayOfInt(o Object) []int {
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
	default:
		panic(RT.NewError(fmt.Sprintf("Not convertible to array of Int: %s", obj.ToString(true))))
	}
}

func ConvertToArrayOfString(o Object) []string {
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
	default:
		panic(RT.NewError(fmt.Sprintf("Not convertible to array of String: %s", obj.ToString(true))))
	}
}
