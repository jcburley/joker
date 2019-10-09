package core

import (
	"os"
)

var GoTypesVec [4]*GoTypeInfo

func SwitchGoType(g interface{}) *GoTypeInfo {
	switch g.(type) {
	case os.FileInfo:
		return GoTypesVec[0]
	case *os.FileInfo:
		return GoTypesVec[1]
	case os.File:
		return GoTypesVec[2]
	case *os.File:
		return GoTypesVec[3]
	}
	return nil
}
