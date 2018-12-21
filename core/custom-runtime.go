package core

func ConvertToArrayOfByte(o Object) []byte {
	switch obj := o.(type) {
	case String:
		return []byte(obj.S)
	case *Vector:
		return []byte("TODO(BYTE)")
	default:
		return []byte("FAIL(BYTE)")
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
	return []string{"FAIL", "ConvertToArrayOfString", "TOO"}
}

func ConvertToArrayOfObject(o Object) []Object {
	return []Object{}
}
