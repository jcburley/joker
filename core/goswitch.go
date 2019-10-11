// Placeholder for big Go switch on types. Overwritten by gostd.

package core

import (
)

var GoTypesVec [0]*GoTypeInfo

func SwitchGoType(g interface{}) *GoTypeInfo {
	switch g.(type) {
	}
	return nil
}
