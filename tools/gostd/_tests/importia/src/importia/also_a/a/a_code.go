package a

import (
	"../../b"
)

type ImportIa string

func ImportFunc() string {
	return "i am also a"
}

type somethingElse b.ImportIa
