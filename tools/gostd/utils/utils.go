package utils

import ()

func Check(e error) {
	if e != nil {
		panic(e)
	}
}
