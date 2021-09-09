package jtypes

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/astutils"
	"github.com/candid82/joker/tools/gostd/genutils"
	. "github.com/candid82/joker/tools/gostd/godb"
	. "go/ast"
	"go/types"
	"os"
)

// Info on Clojure types, including map of Clojure type names to said type
// info.  A Clojure type name is either unqualified (built-in, not
// namespace-rooted) or fully qualified by a namespace name
// (e.g. "go.std.example/SomeType").
type Info struct {
	Expr                 Expr   // [key] The canonical referencing expression (if any)
	FullName             string // [key] Full name of type as a Clojure expression
	GoType               types.Type
	GoTypeName           string // GoType.String() or something else
	FullNameDoc          string // Full name of type as a Clojure expression (for documentation); just e.g. "Int" for builtin global types
	Who                  string // who made me
	Pattern              string // E.g. "%s", "refTo%s" (for reference types), "arrayOf%s" (for array types)
	Namespace            string // E.g. "go.std.net.url", in which this type resides ("" denotes global namespace)
	BaseName             string // E.g. "Listener"
	BaseNameDoc          string // Might be e.g. "Int" when BaseName is "Int8"
	ArgClojureType       string // Can convert this type to a Go function arg with my type
	ArgFromClojureObject string // Append this to Clojure object to extract value of my type
	ArgExtractFunc       string
	ArgClojureArgType    string // Clojure argument type for a Go function arg with my type
	ConvertFromClojure   string // Pattern to convert a (scalar) %s to this type
	ConvertFromMap       string // Pattern to convert a map %s key %s to this type
	AsClojureObject      string // Pattern to convert this type to a normal Clojure type; empty string means wrap in a GoObject
	PromoteType          string // Pattern to promote to a canonical type (used by constant evaluation)
	IsUnsupported        bool   // Is this unsupported?
}

// Maps type-defining Expr or Clojure type names (with or without
// "<ns>/" prefixes, depending on globality) to exactly one struct
// describing that type.
var typesByExpr = map[Expr]*Info{}
var typesByFullName = map[string]*Info{}
var typesByGoTypeName = map[string]*Info{
	"nil":        Nil,
	"error":      Error,
	"bool":       Boolean,
	"byte":       Byte,
	"rune":       Rune,
	"string":     String,
	"int":        Int,
	"int8":       Int8,
	"int16":      Int16,
	"int32":      Int32,
	"int64":      Int64,
	"uint":       UInt,
	"uint8":      UInt8,
	"uint16":     UInt16,
	"uint32":     UInt32,
	"uint64":     UInt64,
	"uintptr":    UIntPtr,
	"float32":    Float32,
	"float64":    Float64,
	"complex128": Complex128,
}
var typesByGoType = map[types.Type]*Info{}

func patternForExpr(e Expr) (pattern string, ue Expr) {
	switch v := e.(type) {
	case *ArrayType:
		len, _ := astutils.IntExprToString(v.Len)
		pattern, e = patternForExpr(v.Elt)
		return "array" + len + "Of" + pattern, e
	case *StarExpr:
		pattern, e = patternForExpr(v.X)
		return "refTo" + pattern, e
	case *MapType:
		patternKey, _ := patternForExpr(v.Key)
		patternValue, eValue := patternForExpr(v.Value)
		res := "map_" + patternKey + "_Of_" + fmt.Sprintf(patternValue, "<whatever>")
		return fmt.Sprintf("ABEND777(jtypes.go: multiple underlying expressions not supported: %s)", res), eValue
	case *ChanType:
		pattern, e = patternForExpr(v.Value)
		baseName := "chan"
		switch v.Dir {
		case SEND:
			baseName = "chanSend"
		case RECV:
			baseName = "chanRecv"
		case SEND | RECV:
		default:
			baseName = fmt.Sprintf("ABEND737(jtypes.go: %s Dir=0x%x not supported)", astutils.ExprToString(v), v.Dir)
		}
		return baseName + "Of" + pattern, e
	default:
		return "%s", e
	}
}

func namingForExpr(e Expr) (pattern, ns, baseName, baseNameDoc, name, nameDoc string, info *Info) {
	var ue Expr
	pattern, ue = patternForExpr(e)

	switch v := ue.(type) {
	case *Ident:
		if !astutils.IsBuiltin(v.Name) {
			ns = ClojureNamespaceForExpr(ue)
			baseName = v.Name
			baseNameDoc = baseName
		} else {
			uInfo, found := typesByGoTypeName[v.Name]
			if !found {
				panic(fmt.Sprintf("no type info for builtin `%s'", v.Name))
			}
			baseName = uInfo.FullName
			baseNameDoc = uInfo.FullNameDoc
			if e == ue {
				info = uInfo
			}
		}
	case *SelectorExpr:
		pkgName := v.X.(*Ident).Name
		ns = ClojureNamespaceForGoFile(pkgName, GoFileForExpr(v))
		baseName = v.Sel.Name
		baseNameDoc = baseName
	case *InterfaceType:
		if !v.Incomplete && len(v.Methods.List) == 0 {
			baseName = "GoObject"
		} else {
			baseName = fmt.Sprintf("ABEND320(jtypes.go: %s not supported)", astutils.ExprToString(v))
		}
		baseNameDoc = baseName
	case *StructType:
		if v.Fields == nil || len(v.Fields.List) == 0 {
			baseName = "struct{}"
		} else {
			baseName = fmt.Sprintf("ABEND787(jtypes.go: %s not supported)", astutils.ExprToString(v))
		}
		baseNameDoc = baseName
	case *FuncType:
		baseName = fmt.Sprintf("ABEND727(jtypes.go: %s not supported)", astutils.ExprToString(v))
		baseNameDoc = baseName
	case *Ellipsis:
		baseName = fmt.Sprintf("ABEND747(jtypes.go: %s not supported)", astutils.ExprToString(v))
		baseNameDoc = baseName
	default:
		panic(fmt.Sprintf("unrecognized underlying expr %T for %T", ue, e))
	}

	name = genutils.CombineClojureName(ns, fmt.Sprintf(pattern, baseName))
	nameDoc = genutils.CombineClojureName(ns, fmt.Sprintf(pattern, baseNameDoc))

	//	fmt.Printf("jtypes.go/typeNameForExpr: %s (`%s' %s %s) %+v => `%s' %T at:%s\n", name, pattern, ns, baseName, e, pattern, ue, WhereAt(e.Pos()))

	// tav, found := astutils.TypeCheckerInfo.Types[e]
	// if found {
	// 	goTypeName = tav.Type.String()
	// } else {
	// 	fmt.Fprintf(os.Stderr, "jtypes.go/namingForExpr():cannot find type info for %s\n", name)
	// }

	return
}

func patternForType(ty types.Type) (pattern string, uty types.Type) {
	switch v := ty.(type) {
	case *types.Array:
		pattern, uty = patternForType(v.Elem())
		return fmt.Sprintf("array%dOf%s", v.Len(), pattern), uty
	case *types.Slice: // I guess this is what "[]foo" becomes when declared as a parameter, versus as a result?
		pattern, uty = patternForType(v.Elem())
		return fmt.Sprintf("arrayOf%s", pattern), uty
	case *types.Pointer:
		pattern, uty = patternForType(v.Elem())
		return "refTo" + pattern, uty
	case *types.Map:
		patternKey, _ := patternForType(v.Key())
		patternValue, eValue := patternForType(v.Elem())
		res := "map_" + patternKey + "_Of_" + fmt.Sprintf(patternValue, "<whatever>")
		return fmt.Sprintf("ABEND777(jtypes.go: multiple underlying expressions not supported: %s)", res), eValue
	case *types.Chan:
		pattern, uty = patternForType(v.Elem())
		baseName := "chan"
		switch v.Dir() {
		case types.SendOnly:
			baseName = "chanSend"
		case types.RecvOnly:
			baseName = "chanRecv"
		case types.SendRecv:
		default:
			baseName = fmt.Sprintf("ABEND737(jtypes.go: %s Dir=0x%x not supported)", v.String(), v.Dir())
		}
		return baseName + "Of" + pattern, uty
	default:
		return "%s", ty
	}
}

func namingForType(ty types.Type) (pattern, ns, baseName, baseNameDoc, name, nameDoc string, info *Info) {
	var uty types.Type
	pattern, uty = patternForType(ty)

	switch v := uty.(type) {
	case *types.Basic:
		if !astutils.IsBuiltin(v.Name()) { // E.g. unsafe.Pointer
			ns = ClojureNamespaceForType(uty)
			baseName = v.Name()
			baseNameDoc = baseName
		} else {
			uInfo, found := typesByGoTypeName[v.Name()]
			if !found {
				panic(fmt.Sprintf("no type info for builtin `%s'", v.Name()))
			}
			baseName = uInfo.FullName
			baseNameDoc = uInfo.FullNameDoc
			if ty == uty {
				info = uInfo
			}
		}
	case *types.Named:
		ns = ClojureNamespaceForType(uty)
		baseName = v.Obj().Name()
		baseNameDoc = baseName
	case *types.Interface:
		if v.NumMethods() == 0 {
			baseName = "GoObject"
		} else {
			baseName = fmt.Sprintf("ABEND320(jtypes.go: %s not supported)", v.String())
		}
		baseNameDoc = baseName
	case *types.Struct:
		if v.NumFields() == 0 {
			baseName = "struct{}"
		} else {
			baseName = fmt.Sprintf("ABEND787(jtypes.go: %s not supported)", v.String())
		}
		baseNameDoc = baseName
	case *types.Signature:
		baseName = fmt.Sprintf("ABEND727(jtypes.go: %s not supported)", v.String())
		baseNameDoc = baseName
	default:
		panic(fmt.Sprintf("unrecognized underlying %T Type %s for %T %s", uty, uty.String(), v, v.String()))
	}

	name = genutils.CombineClojureName(ns, fmt.Sprintf(pattern, baseName))
	nameDoc = genutils.CombineClojureName(ns, fmt.Sprintf(pattern, baseNameDoc))

	//	fmt.Printf("jtypes.go/typeNameForExpr: %s (`%s' %s %s) %+v => `%s' %T at:%s\n", name, pattern, ns, baseName, e, pattern, ue, WhereAt(e.Pos()))

	// tav, found := astutils.TypeCheckerInfo.Types[e]
	// if found {
	// 	goTypeName = tav.Type.String()
	// } else {
	// 	fmt.Fprintf(os.Stderr, "jtypes.go/namingForExpr():cannot find type info for %s\n", name)
	// }

	return
}

func Define(ts *TypeSpec, varExpr Expr) *Info {

	ns := ClojureNamespaceForPos(Fset.Position(ts.Name.NamePos))

	pattern, _ := patternForExpr(varExpr)

	name := genutils.CombineClojureName(ns, fmt.Sprintf(pattern, ts.Name.Name))

	jti := &Info{
		FullName:          name,
		FullNameDoc:       name,
		Who:               "TypeDefine",
		Pattern:           pattern,
		Namespace:         ns,
		BaseName:          ts.Name.Name,
		BaseNameDoc:       ts.Name.Name,
		ArgExtractFunc:    "Object",
		ArgClojureArgType: name,
		AsClojureObject:   "GoObjectIfNeeded(%s%s)",
	}

	jti.register()

	return jti

}

func InfoForGoTypeName(typeName string) *Info {
	return typesByGoTypeName[typeName]
}

func InfoForGoType(ty types.Type) *Info {
	if info, found := typesByGoType[ty]; found {
		return info
	}

	pattern, ns, baseName, baseNameDoc, fullName, fullNameDoc, info := namingForType(ty)

	if info != nil {
		// Already found info on builtin Go type, so just return that.
		typesByGoType[ty] = info
		return info
	}

	if inf, found := typesByFullName[fullName]; found {
		typesByGoType[ty] = inf
		return inf
	}

	info = &Info{
		Expr:              nil,
		FullName:          fullName,
		GoType:            ty,
		GoTypeName:        ty.String(),
		FullNameDoc:       fullNameDoc,
		Who:               fmt.Sprintf("InfoForGoType %s", ty.String()),
		Pattern:           pattern,
		Namespace:         ns,
		BaseName:          baseName,
		BaseNameDoc:       baseNameDoc,
		ArgClojureArgType: fullName,
	}

	typesByFullName[fullName] = info
	typesByGoTypeName[ty.String()] = info
	typesByGoType[ty] = info

	return info
}

func InfoForExpr(e Expr, goType types.Type) *Info {
	if info, ok := typesByExpr[e]; ok {
		return info
	}

	pattern, ns, baseName, baseNameDoc, fullName, fullNameDoc, info := namingForExpr(e)

	if info != nil {
		// Already found info on builtin Go type, so just return that.
		typesByExpr[e] = info
		return info
	}

	if inf, found := typesByFullName[fullName]; found {
		typesByExpr[e] = inf
		return inf
	}

	convertFromClojure, convertFromMap := ConversionsFn(e)

	if goType == nil {
		fmt.Fprintf(os.Stderr, "jtypes.go/InfoForExpr(): Nil goType for %s\n", astutils.ExprToString(e))
		return nil
	}

	goTypeName := goType.String()

	info = &Info{
		Expr:               e,
		FullName:           fullName,
		GoType:             goType,
		GoTypeName:         goTypeName,
		FullNameDoc:        fullNameDoc,
		Who:                fmt.Sprintf("TypeForExpr %T", e),
		Pattern:            pattern,
		Namespace:          ns,
		BaseName:           baseName,
		BaseNameDoc:        baseNameDoc,
		ArgClojureArgType:  fullName,
		ConvertFromClojure: convertFromClojure,
		ConvertFromMap:     convertFromMap,
	}

	typesByExpr[e] = info
	typesByFullName[fullName] = info
	typesByGoTypeName[goTypeName] = info
	typesByGoType[goType] = info

	return info
}

func (ti *Info) NameDoc(e Expr) string {
	if ti.Pattern == "" || ti.Namespace == "" {
		return ti.FullNameDoc
	}
	if e != nil && ClojureNamespaceForExpr(e) != ti.Namespace {
		//		fmt.Printf("jtypes.NameDoc(%+v at %s) => %s (in ns=%s) per %s\n", e, WhereAt(e.Pos()), ti.FullName, ti.Namespace, ClojureNamespaceForExpr(e))
		return ti.FullNameDoc
	}
	res := fmt.Sprintf(ti.Pattern, ti.BaseNameDoc)
	//	fmt.Printf("jtypes.NameDoc(%+v at %s) => just %s (in ns=%s) per %s\n", e, WhereAt(e.Pos()), res, ti.Namespace, ClojureNamespaceForExpr(e))
	return res
}

func (ti *Info) NameDocForType(pkg *types.Package) string {
	if ti.Pattern == "" || ti.Namespace == "" {
		return ti.FullNameDoc
	}
	if pkg != nil && ClojureNamespaceForPackage(pkg) != ti.Namespace {
		//		fmt.Printf("jtypes.NameDoc(%+v at %s) => %s (in ns=%s) per %s\n", e, WhereAt(e.Pos()), ti.FullName, ti.Namespace, ClojureNamespaceForExpr(e))
		return ti.FullNameDoc
	}
	res := fmt.Sprintf(ti.Pattern, ti.BaseNameDoc)
	//	fmt.Printf("jtypes.NameDoc(%+v at %s) => just %s (in ns=%s) per %s\n", e, WhereAt(e.Pos()), res, ti.Namespace, ClojureNamespaceForExpr(e))
	return res
}

func (ti *Info) register() {
	if _, found := typesByFullName[ti.FullName]; !found {
		typesByFullName[ti.FullName] = ti
	}
	if _, found := typesByGoTypeName[ti.GoTypeName]; !found {
		typesByGoTypeName[ti.GoTypeName] = ti
	}
	if _, found := typesByGoType[ti.GoType]; !found {
		typesByGoType[ti.GoType] = ti
	}
}

var Nil = &Info{}

var Error = &Info{
	FullName:             "Error",
	FullNameDoc:          "Error",
	BaseName:             "Error",
	BaseNameDoc:          "Error",
	ArgClojureType:       "Error",
	ArgFromClojureObject: "",
	ArgExtractFunc:       "Error",
	ArgClojureArgType:    "Error",
	ConvertFromMap:       `FieldAsError(%s, %s)`,
	AsClojureObject:      "Error(%s%s)",
	ConvertFromClojure:   "ObjectAsError(%s, %s)",
	PromoteType:          "%s",
}

var Boolean = &Info{
	FullName:             "Boolean",
	FullNameDoc:          "Boolean",
	BaseName:             "Boolean",
	BaseNameDoc:          "Boolean",
	ArgClojureType:       "Boolean",
	ArgFromClojureObject: ".B",
	ArgExtractFunc:       "Boolean",
	ArgClojureArgType:    "Boolean",
	ConvertFromMap:       "FieldAsBoolean(%s, %s)",
	AsClojureObject:      "Boolean(%s%s)",
	ConvertFromClojure:   "ObjectAsBoolean(%s, %s)",
	PromoteType:          "%s",
}

var Byte = &Info{
	FullName:             "Byte",
	FullNameDoc:          "Byte",
	BaseName:             "Byte",
	BaseNameDoc:          "Byte",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Byte",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsByte(%s, %s)`,
	AsClojureObject:      "Int(int(%s)%s)",
	ConvertFromClojure:   "ObjectAsByte(%s, %s)",
	PromoteType:          "int(%s)",
}

var Rune = &Info{
	FullName:             "Char",
	FullNameDoc:          "Char",
	BaseName:             "Char",
	BaseNameDoc:          "Char",
	ArgClojureType:       "Char",
	ArgFromClojureObject: ".Ch",
	ArgExtractFunc:       "Char",
	ArgClojureArgType:    "Char",
	ConvertFromMap:       `FieldAsChar(%s, %s)`,
	AsClojureObject:      "Char(%s%s)",
	ConvertFromClojure:   "ObjectAsChar(%s, %s)",
	PromoteType:          "%s",
}

var String = &Info{
	FullName:             "String",
	FullNameDoc:          "String",
	BaseName:             "String",
	BaseNameDoc:          "String",
	ArgClojureType:       "String",
	ArgFromClojureObject: ".S",
	ArgExtractFunc:       "String",
	ArgClojureArgType:    "String",
	ConvertFromMap:       `FieldAsString(%s, %s)`,
	AsClojureObject:      "String(%s%s)",
	ConvertFromClojure:   "ObjectAsString(%s, %s)",
	PromoteType:          "%s",
}

var Int = &Info{
	FullName:             "Int",
	FullNameDoc:          "Int",
	BaseName:             "Int",
	BaseNameDoc:          "Int",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Int",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsInt(%s, %s)`,
	AsClojureObject:      "Int(%s%s)",
	ConvertFromClojure:   "ObjectAsInt(%s, %s)",
	PromoteType:          "%s",
}

var Int8 = &Info{
	FullName:             "Int8",
	FullNameDoc:          "Int",
	BaseName:             "Int8",
	BaseNameDoc:          "Int",
	ArgClojureType:       "Int",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Int8",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsInt8(%s, %s)`,
	AsClojureObject:      "Int(int(%s)%s)",
	ConvertFromClojure:   "ObjectAsInt8(%s, %s)",
	PromoteType:          "int(%s)",
}

var Int16 = &Info{
	FullName:             "Int16",
	FullNameDoc:          "Int",
	BaseName:             "Int16",
	BaseNameDoc:          "Int",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Int16",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsInt16(%s, %s)`,
	AsClojureObject:      "Int(int(%s)%s)",
	ConvertFromClojure:   "ObjectAsInt16(%s, %s)",
	PromoteType:          "int(%s)",
}

var Int32 = &Info{
	FullName:             "Int32",
	FullNameDoc:          "Int",
	BaseName:             "Int32",
	BaseNameDoc:          "Int",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Int32",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsInt32(%s, %s)`,
	AsClojureObject:      "Int(int(%s)%s)",
	ConvertFromClojure:   "ObjectAsInt32(%s, %s)",
	PromoteType:          "int(%s)",
}

var Int64 = &Info{
	FullName:             "Int64",
	FullNameDoc:          "BigInt",
	BaseName:             "Int64",
	BaseNameDoc:          "BigInt",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".BigInt().Int64()",
	ArgExtractFunc:       "Int64",
	ArgClojureArgType:    "BigInt",
	ConvertFromMap:       `FieldAsInt64(%s, %s)`,
	AsClojureObject:      "Number(%s%s)",
	ConvertFromClojure:   "ObjectAsInt64(%s, %s)",
	PromoteType:          "int64(%s)",
}

var UInt = &Info{
	FullName:             "Uint",
	FullNameDoc:          "Number",
	BaseName:             "Uint",
	BaseNameDoc:          "Number",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Uint",
	ArgClojureArgType:    "Number",
	ConvertFromMap:       `FieldAsUint(%s, %s)`,
	AsClojureObject:      "Number(%s%s)",
	ConvertFromClojure:   "ObjectAsUint(%s, %s)",
	PromoteType:          "uint64(%s)",
}

var UInt8 = &Info{
	FullName:             "Uint8",
	FullNameDoc:          "Int",
	BaseName:             "Uint8",
	BaseNameDoc:          "Int",
	ArgClojureType:       "Int",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Uint8",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsUint8(%s, %s)`,
	AsClojureObject:      "Int(int(%s)%s)",
	ConvertFromClojure:   "ObjectAsUint8(%s, %s)",
	PromoteType:          "int(%s)",
}

var UInt16 = &Info{
	FullName:             "Uint16",
	FullNameDoc:          "Int",
	BaseName:             "Uint16",
	BaseNameDoc:          "Int",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Uint16",
	ArgClojureArgType:    "Int",
	ConvertFromMap:       `FieldAsUint16(%s, %s)`,
	AsClojureObject:      "Int(int(%s)%s)",
	ConvertFromClojure:   "ObjectAsUint16(%s, %s)",
	PromoteType:          "int(%s)",
}

var UInt32 = &Info{
	FullName:             "Uint32",
	FullNameDoc:          "Number",
	BaseName:             "Uint32",
	BaseNameDoc:          "Number",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".Int().I",
	ArgExtractFunc:       "Uint32",
	ArgClojureArgType:    "Number",
	ConvertFromMap:       `FieldAsUint32(%s, %s)`,
	AsClojureObject:      "Number(%s%s)",
	ConvertFromClojure:   "ObjectAsUint32(%s, %s)",
	PromoteType:          "int64(%s)",
}

var UInt64 = &Info{
	FullName:             "Uint64",
	FullNameDoc:          "Number",
	BaseName:             "Uint64",
	BaseNameDoc:          "Number",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".BigInt().Uint64()",
	ArgExtractFunc:       "Uint64",
	ArgClojureArgType:    "Number",
	ConvertFromMap:       `FieldAsUint64(%s, %s)`,
	AsClojureObject:      "Number(%s%s)",
	ConvertFromClojure:   "ObjectAsUint64(%s, %s)",
	PromoteType:          "uint64(%s)",
}

var UIntPtr = &Info{
	FullName:             "UintPtr",
	FullNameDoc:          "Number",
	BaseName:             "UintPtr",
	BaseNameDoc:          "Number",
	ArgClojureType:       "Number",
	ArgFromClojureObject: ".BigInt().Uint64()",
	ArgExtractFunc:       "UintPtr",
	ArgClojureArgType:    "Number",
	ConvertFromMap:       `FieldAsUintPtr(%s, %s)`,
	AsClojureObject:      "Number(%s%s)",
	ConvertFromClojure:   "ObjectAsUintPtr(%s, %s)",
	PromoteType:          "uint64(%s)",
}

var Float32 = &Info{
	FullName:             "Float32",
	FullNameDoc:          "Double",
	BaseName:             "Float32",
	BaseNameDoc:          "Double",
	ArgClojureType:       "",
	ArgFromClojureObject: "",
	ArgExtractFunc:       "ABEND007(find these)",
	ArgClojureArgType:    "Double",
	ConvertFromMap:       `FieldAsDouble(%s, %s)`,
	AsClojureObject:      "Double(float64(%s)%s)",
	ConvertFromClojure:   "ObjectAsFloat32(%s, %s)",
	PromoteType:          "float64(%s)",
}

var Float64 = &Info{
	FullName:             "Double",
	FullNameDoc:          "Double",
	BaseName:             "Double",
	BaseNameDoc:          "Double",
	ArgClojureType:       "Double",
	ArgFromClojureObject: "",
	ArgExtractFunc:       "ABEND007(find these)",
	ArgClojureArgType:    "Double",
	ConvertFromMap:       `FieldAsDouble(%s, %s)`,
	AsClojureObject:      "Double(%s%s)",
	ConvertFromClojure:   "ObjectAsFloat64(%s, %s)",
	PromoteType:          "%s",
}

var Complex128 = &Info{
	FullName:             "ABEND007(find these)",
	FullNameDoc:          "ABEND007(find these)",
	BaseName:             "ABEND007(find these)",
	BaseNameDoc:          "ABEND007(find these)",
	ArgClojureType:       "",
	ArgFromClojureObject: "",
	ArgExtractFunc:       "ABEND007(find these)",
	ArgClojureArgType:    "ABEND007(find these)",
	ConvertFromMap:       "", // TODO: support this in Clojure, even if via just [real imag]
	ConvertFromClojure:   "ObjectAsComplex128(%s, %s)",
	AsClojureObject:      "Complex(%s%s)",
}

var ConversionsFn func(e Expr) (fromClojure, fromMap string)
