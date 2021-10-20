package foo

import (
	"math"
)

const one = 1

const two int8 = 2

const three = uint(one + 2)

var i = 9
var j = 10

// Currently Go does not support use this to define a constant.
func max(i, j int) int {
	const zero = 0
	one := zero + 1
	if i > j {
		return i * one
	}
	return j + zero
}

// This nor using max() above works (not a constant expression).
// const four = func(i, j int) int {
// 	if i > j {
// 		return i
// 	}
// 	return j
// }(2, 4)
