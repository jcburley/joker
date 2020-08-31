package main

import (
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

type TypeInfo interface {
	ArgExtractFunc() string     // Call Extract<this>() for arg with my type
	ArgClojureArgType() string  // Clojure argument type for a Go function arg with my type
	ConvertFromClojure() string // Pattern to convert a (scalar) %s to this type
	ConvertToClojure() string   // Pattern to convert this type to an appropriate Clojure object
	AsJokerObject() string      // Pattern to convert this type to a normal Joker type, or empty string to simply wrap in a GoObject
	ClojureDecl() string
	ClojureDeclDoc() string
	GoDecl() string
	GoDeclDoc() string
	GoCode() string
	Nullable() bool // Can an instance of the type == nil (e.g. 'error' type)?
}

type typeInfo struct {
	jti jtypes.Info
	gti gtypes.Info
}

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

var typeMap = map[gtypes.Info]TypeInfo{}

var TypeDefsToGoTypes = map[*gtypes.GoType]*GoTypeInfo{}

func RegisterTypeDecl(ts *TypeSpec, gf *godb.GoFile, pkg string, parentDoc *CommentGroup) bool {
	name := ts.Name.Name
	goTypeName := pkg + "." + name

	// if pkg == "unsafe" && name == "ArbitraryType" {
	// 	if godb.Verbose {
	// 		fmt.Printf("Excluding mythical type %s.%s\n", pkg, name)
	// 	}
	// 	return false
	// }

	if WalkDump {
		fmt.Printf("Type %s at %s:\n", goTypeName, godb.WhereAt(ts.Pos()))
		Print(godb.Fset, ts)
	}

	tdiVec := gtypes.TypeDefine(ts, gf, parentDoc)
	for _, tdi := range tdiVec {
		if !strings.Contains(tdi.ClojureName, "[") { // exclude arrays
			ClojureCode[pkg].InitTypes[tdi] = struct{}{}
			GoCode[pkg].InitTypes[tdi] = struct{}{}
		}
	}

	gt := registerTypeGOT(gf, goTypeName, ts)
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

	ti := &typeInfo{
		jti: jtypes.Info{
			ArgClojureArgType:  gt.ArgClojureArgType,
			ConvertFromClojure: gt.ConvertFromClojure,
			ConvertToClojure:   gt.ConvertToClojure,
			AsJokerObject:      gt.ConvertToClojure,
		},
		gti: gtypes.NewInfo(
			goTypeName,
			gt.Nullable,
		)}

	typeMap[ti.gti] = ti

	return true
}

// Maps type-defining Expr to exactly one struct describing that type
var typesByExpr = map[Expr]TypeInfo{}

func BadInfo(err string) typeInfo {
	return typeInfo{
		jti: jtypes.Info{
			ArgExtractFunc:     err,
			ArgClojureArgType:  err,
			ConvertFromClojure: err + "%0s%0s",
			ConvertToClojure:   err + "%0s%0s",
			AsJokerObject:      err + "%0s%0s",
		},
	}
}

func TypeInfoForExpr(e Expr) TypeInfo {
	gti := gtypes.TypeInfoForExpr(e)

	if ti, found := typeMap[gti]; found {
		return ti
	}

	ti := &typeInfo{
		gti: gti,
	}

	typeMap[gti] = ti

	return ti
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

func (ti typeInfo) ArgExtractFunc() string {
	return ti.jti.ArgExtractFunc
}

func (ti typeInfo) ArgClojureArgType() string {
	return ti.jti.ArgClojureArgType
}

func (ti typeInfo) ConvertFromClojure() string {
	return ti.jti.ConvertFromClojure
}

func (ti typeInfo) ConvertToClojure() string {
	return ti.jti.ConvertToClojure
}

func (ti typeInfo) AsJokerObject() string {
	return ti.jti.AsJokerObject
}

func (ti typeInfo) ClojureDecl() string {
	return ti.jti.ArgClojureArgType // TODO: can this just be renamed "ClojureType" at some point?
}

func (ti typeInfo) ClojureDeclDoc() string {
	return ti.ClojureDecl() // TODO: Probably need difference soon.
}

func (ti typeInfo) GoDecl() string {
	return ti.gti.Name
}

func (ti typeInfo) GoDeclDoc() string {
	s := strings.Split(ti.GoDecl(), ".")
	return s[len(s)-1]
}

func (ti typeInfo) GoCode() string {
	return "" // TODO: Probably need something here in some cases? Seems to generic of a name though.
}

func (ti typeInfo) Nullable() bool {
	return ti.gti.Nullable
}

func init() {
	typeMap[gtypes.Bool] = typeInfo{
		jti: jtypes.Bool,
		gti: gtypes.Bool,
	}
	typeMap[gtypes.Byte] = typeInfo{
		jti: jtypes.Byte,
		gti: gtypes.Byte,
	}
	typeMap[gtypes.Complex128] = typeInfo{
		jti: jtypes.Complex128,
		gti: gtypes.Complex128,
	}
	typeMap[gtypes.Error] = typeInfo{
		jti: jtypes.Error,
		gti: gtypes.Error,
	}
	typeMap[gtypes.Float32] = typeInfo{
		jti: jtypes.Float32,
		gti: gtypes.Float32,
	}
	typeMap[gtypes.Float64] = typeInfo{
		jti: jtypes.Float64,
		gti: gtypes.Float64,
	}
	typeMap[gtypes.Int] = typeInfo{
		jti: jtypes.Int,
		gti: gtypes.Int,
	}
	typeMap[gtypes.Int32] = typeInfo{
		jti: jtypes.Int32,
		gti: gtypes.Int32,
	}
	typeMap[gtypes.Int64] = typeInfo{
		jti: jtypes.Int64,
		gti: gtypes.Int64,
	}
	typeMap[gtypes.Rune] = typeInfo{
		jti: jtypes.Rune,
		gti: gtypes.Rune,
	}
	typeMap[gtypes.String] = typeInfo{
		jti: jtypes.String,
		gti: gtypes.String,
	}
	typeMap[gtypes.UInt] = typeInfo{
		jti: jtypes.UInt,
		gti: gtypes.UInt,
	}
	typeMap[gtypes.UInt16] = typeInfo{
		jti: jtypes.UInt16,
		gti: gtypes.UInt16,
	}
	typeMap[gtypes.UInt32] = typeInfo{
		jti: jtypes.UInt32,
		gti: gtypes.UInt32,
	}
	typeMap[gtypes.UInt64] = typeInfo{
		jti: jtypes.UInt64,
		gti: gtypes.UInt64,
	}
	typeMap[gtypes.UInt8] = typeInfo{
		jti: jtypes.UInt8,
		gti: gtypes.UInt8,
	}
	typeMap[gtypes.UIntPtr] = typeInfo{
		jti: jtypes.UIntPtr,
		gti: gtypes.UIntPtr,
	}
}
