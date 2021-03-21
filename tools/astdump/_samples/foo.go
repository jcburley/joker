package foo

import (
	"math"
)

const one = 1

const two int8 = 2

const three = uint(one + 2)

// Currently Go does not support use this to define a constant.
func max(i, j int) int {
	if i > j {
		return i
	}
	return j
}

// This nor using max() above works (not a constant expression).
// const four = func(i, j int) int {
// 	if i > j {
// 		return i
// 	}
// 	return j
// }(2, 4)
