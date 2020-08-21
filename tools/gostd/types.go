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
	"go/types"
	"sort"
	"strings"
)

type GoTypeInfo struct {
	LocalName                 string       // empty (not a declared type) or the basic type name ("foo" for "x/y.foo")
	FullGoName                string       // empty ("struct {...}" etc.), localName (built-in), path/to/pkg.LocalName, or ABEND if unsupported
	SourceFile                *godb.GoFile // location of the type defintion
	Td                        *TypeSpec
	Type                      *gtypes.GoType // Primary type in the new package
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

var TypeDefsToGoTypes = map[*gtypes.GoType]*GoTypeInfo{}

func registerType(ts *TypeSpec, gf *godb.GoFile, pkg string, parentDoc *CommentGroup) bool {
	name := ts.Name.Name
	typeName := pkg + "." + name

	// if pkg == "unsafe" && name == "ArbitraryType" {
	// 	if godb.Verbose {
	// 		fmt.Printf("Excluding mythical type %s.%s\n", pkg, name)
	// 	}
	// 	return false
	// }

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

	typeMap[typeName] = jtypes.NewInfo(
		gt.ArgExtractFunc,
		gt.ArgClojureArgType,
		gt.ConvertFromClojure,
		gt.ConvertToClojure,
		gt.ConvertToClojure,
		gt.Nullable)

	return true
}

// Maps type-defining Expr to exactly one struct describing that type
var typesByExpr = map[Expr]jtypes.Info{}
var NumGoExprHits uint
var NumGoNameHits uint

func JokerTypeInfoForExpr(e Expr) jtypes.Info {
	if jti, ok := typesByExpr[e]; ok {
		NumGoExprHits++
		return jti
	}
	goName := typeName(e)
	if jti, ok := typeMap[goName]; ok {
		NumGoNameHits++
		typesByExpr[e] = jti
		return jti
	}

	switch e.(type) {
	case *Ident:
		if jti, found := typeMap[goName]; found {
			return jti
		}
		return jtypes.BadInfo(fmt.Sprintf("ABEND620(types.go:JokerTypeInfoForExpr: unrecognized identifier %s)", goName))
	}
	return jtypes.BadInfo(fmt.Sprintf("ABEND621(types.go:JokerTypeInfoForExpr: unsupported expr type %T)", e))
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

func typeName(e Expr) string {
	switch x := e.(type) {
	case *Ident:
		break
	case *ArrayType:
		return "[" + exprToString(x.Len) + "]" + typeName(x.Elt)
	case *StarExpr:
		return "*" + typeName(x.X)
	case *MapType:
		return "map[" + typeName(x.Key) + "]" + typeName(x.Value)
	case *SelectorExpr:
		return fmt.Sprintf("%s", x.X) + "." + x.Sel.Name
	default:
		return fmt.Sprintf("ABEND699(types.go:typeName: unrecognized node %T)", e)
	}

	x := e.(*Ident)
	local := x.Name
	prefix := ""
	if types.Universe.Lookup(local) == nil {
		prefix = godb.GoPackageForExpr(e) + "."
	}

	return prefix + local
}

func exprToString(e Expr) string {
	if e == nil {
		return ""
	}
	switch v := e.(type) {
	case *Ellipsis:
		return "..." + exprToString(v.Elt)
	case *BasicLit:
		return v.Value
	}
	return fmt.Sprintf("%v", e)
}

func init() {
	typeMap["bool"] = jtypes.Bool
	typeMap["byte"] = jtypes.Byte
	typeMap["complex128"] = jtypes.Complex128
	typeMap["error"] = jtypes.Error
	typeMap["float32"] = jtypes.Float32
	typeMap["float64"] = jtypes.Float64
	typeMap["int"] = jtypes.Int
	typeMap["int32"] = jtypes.Int32
	typeMap["int64"] = jtypes.Int64
	typeMap["rune"] = jtypes.Rune
	typeMap["string"] = jtypes.String
	typeMap["uint"] = jtypes.UInt
	typeMap["uint16"] = jtypes.UInt16
	typeMap["uint32"] = jtypes.UInt32
	typeMap["uint64"] = jtypes.UInt64
	typeMap["uint8"] = jtypes.UInt8
	typeMap["uintptr"] = jtypes.UIntPtr
}
