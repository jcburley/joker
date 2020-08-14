package main

import (
	//	"github.com/candid82/joker/tools/gostd/gtypes"
	"fmt"
	"github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/gtypes"
	"github.com/candid82/joker/tools/gostd/imports"
	"github.com/candid82/joker/tools/gostd/jtypes"
	. "go/ast"
	"go/token"
	"sort"
	"strings"
)

type GoTypeInfo struct {
	LocalName                 string       // empty (not a declared type) or the basic type name ("foo" for "x/y.foo")
	FullGoName                string       // empty ("struct {...}" etc.), localName (built-in), path/to/pkg.LocalName, or ABEND if unsupported
	SourceFile                *godb.GoFile // location of the type defintion
	Td                        *TypeSpec
	Type                      *gtypes.Type // Primary type in the new package
	Where                     token.Pos
	UnderlyingType            Expr             // nil if not a declared type
	ArgClojureType            string           // Can convert this type to a Go function arg with my type
	ArgFromClojureObject      string           // Append this to Clojure object to extract value of my type
	ArgClojureArgType         string           // Clojure argument type for a Go function arg with my type
	ArgExtractFunc            string           // Call Extract<this>() for arg with my type
	ConvertFromClojure        string           // Pattern to convert a (scalar) %s to this type
	ConvertFromClojureImports []imports.Import // Imports needed to support the above
	ConvertFromMap            string           // Pattern to convert a map %s key %s to this type
	ConvertToClojure          string           // Pattern to convert this type to an appropriate Clojure object
	PromoteType               string           // Pattern to convert type to next larger Go type that Joker supports
	ClojureCode               string
	GoCode                    string
	RequiredImports           *imports.Imports
	Uncompleted               bool // Has this type's info been filled in beyond the registration step?
	Custom                    bool // Is this not a builtin Go type?
	Exported                  bool // Is this an exported type?
	Unsupported               bool // Is this unsupported?
	Constructs                bool // Does the convertion from Clojure actually construct (via &sometype{}), returning ptr?
	Nullable                  bool // Can an instance of the type == nil (e.g. 'error' type)?
}

type GoTypeMap map[string]*GoTypeInfo

/* These map fullGoNames to type info. */
var GoTypes = GoTypeMap{}

var typeMap = map[string]jtypes.Info{}

var TypeDefsToGoTypes = map[*gtypes.Type]*GoTypeInfo{}

func registerType(ts *TypeSpec, gf *godb.GoFile, pkg string, parentDoc *CommentGroup) bool {
	name := ts.Name.Name
	typeName := pkg + "." + name

	if pkg == "unsafe" && name == "ArbitraryType" {
		if godb.Verbose {
			fmt.Printf("Excluding mythical type %s.%s\n", pkg, name)
		}
		return false
	}

	if WalkDump {
		fmt.Printf("Type %s at %s:\n", typeName, godb.WhereAt(ts.Pos()))
		Print(godb.Fset, ts)
	}

	tdiVec := gtypes.TypeDefine(ts, gf, parentDoc)
	for _, tdi := range tdiVec {
		if !strings.Contains(tdi.ClojureName, "[") { // exclude arrays
			ClojureCode[pkg].InitTypes[tdi] = struct{}{}
			GoCode[pkg].InitTypes[tdi] = struct{}{}
		}
	}

	gt := registerTypeGOT(gf, typeName, ts)
	gt.Td = ts
	gt.Type = tdiVec[0]
	gt.Where = ts.Pos()
	gt.RequiredImports = &imports.Imports{}

	TypeDefsToGoTypes[tdiVec[0]] = gt

	if IsExported(name) {
		NumTypes++
		if tdiVec[0].Specificity == gtypes.Concrete {
			NumCtableTypes++
		}
	}

	return true
}

func JokerTypeInfoForExpr(e Expr) jtypes.Info {
	switch td := e.(type) {
	case *Ident:
		if jti, found := typeMap[td.Name]; found {
			return jti
		}
	}
	return jtypes.Nil
}

func SortedTypeInfoMap(m map[string]*GoTypeInfo, f func(k string, v *GoTypeInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

func init() {
	typeMap["error"] = jtypes.Error
	typeMap["string"] = jtypes.String
	MaybeRegisterType_func = registerType // TODO: Remove this kludge (allowing gowalk to call this fn) when able
}
