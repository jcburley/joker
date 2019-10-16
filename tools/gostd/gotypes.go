package main

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/godb"
	. "github.com/candid82/joker/tools/gostd/gowalk"
	"github.com/candid82/joker/tools/gostd/imports"
	. "go/ast"
	gotypes "go/types"
)

func registerType(gf *godb.GoFile, fullGoTypeName string, ts *TypeSpec) *GoTypeInfo {
	if ti, found := GoTypes[fullGoTypeName]; found {
		return ti
	}
	ti := &GoTypeInfo{
		LocalName:         ts.Name.Name,
		FullGoName:        fullGoTypeName,
		SourceFile:        gf,
		UnderlyingType:    &ts.Type,
		ArgClojureArgType: FullTypeNameAsClojure(gf.NsRoot, fullGoTypeName),
		Exported:          IsExported(ts.Name.Name),
		Custom:            true,
		Uncompleted:       true,
		ConvertToClojure:  "GoObject(%s%s)",
		ArgExtractFunc:    "Object",
	}
	GoTypes[fullGoTypeName] = ti
	return ti
}

func toGoTypeNameInfo(pkgDirUnix, baseName string, e *Expr) *GoTypeInfo {
	if ti, found := GoTypes[baseName]; found {
		return ti
	}
	fullGoName := pkgDirUnix + "." + baseName
	if ti, found := GoTypes[fullGoName]; found {
		return ti
	}

	var ti *GoTypeInfo
	if gotypes.Universe.Lookup(baseName) != nil {
		ti = &GoTypeInfo{
			LocalName:          baseName,
			FullGoName:         fmt.Sprintf("ABEND046(gotypes.go: unsupported builtin type %s for %s)", baseName, pkgDirUnix),
			ArgClojureType:     baseName,
			ArgClojureArgType:  baseName,
			ConvertFromClojure: baseName + "(%s)",
			ConvertFromMap:     baseName + "(%s)",
			ConvertToClojure:   "GoObject(%s%s)",
			Unsupported:        true,
			Exported:           true,
		}
	} else {
		ti = &GoTypeInfo{
			LocalName:          baseName,
			FullGoName:         fmt.Sprintf("ABEND051(gotypes.go: unsupported underlying type %s for %s)", baseName, pkgDirUnix),
			ArgClojureType:     baseName,
			ArgClojureArgType:  baseName,
			ConvertFromClojure: baseName + "(%s)",
			ConvertFromMap:     baseName + "(%s)",
			ConvertToClojure:   "GoObject(%s%s)",
			Unsupported:        true,
		}
	}
	GoTypes[baseName] = ti
	return ti
}

func toGoTypeInfo(src *godb.GoFile, ts *TypeSpec) *GoTypeInfo {
	return toGoExprInfo(src, &ts.Type)
}

func toGoExprInfo(src *godb.GoFile, e *Expr) *GoTypeInfo {
	localName := ""
	fullGoName := ""
	convertFromClojure := ""
	exported := true
	var underlyingType *Expr
	unsupported := false
	switch td := (*e).(type) {
	case *Ident:
		ti := toGoTypeNameInfo(src.Package.DirUnix, td.Name, e)
		if ti == nil {
			return nil
		}
		if ti.Uncompleted {
			// Fill in other info now that all types are registered.
			ut := toGoExprInfo(src, ti.UnderlyingType)
			/*			if ut.unsupported {
						ti.fullGoName = ut.fullGoName
						ti.unsupported = true
					}*/
			if ut.ConvertFromClojure != "" {
				if ut.Constructs {
					ti.ConvertFromClojure = "*" + ut.ConvertFromClojure
				} else {
					ti.ConvertFromClojure = fmt.Sprintf("_%s.%s(%s)", ti.SourceFile.Package.BaseName, ti.LocalName, ut.ConvertFromClojure)
				}
			}
			ti.ConvertFromClojureImports = ut.ConvertFromClojureImports
			ti.Uncompleted = false
		}
		return ti
	case *ArrayType:
		return goArrayType(src, &td.Len, &td.Elt)
	case *StarExpr:
		return goStarExpr(src, &td.X)
	}
	if localName == "" || fullGoName == "" {
		localName = fmt.Sprintf("%T", *e)
		fullGoName = fmt.Sprintf("ABEND047(gotypes.go: unsupported type %s)", localName)
		unsupported = true
	}
	v := &GoTypeInfo{
		LocalName:          localName,
		FullGoName:         fullGoName,
		UnderlyingType:     underlyingType,
		Exported:           exported,
		Unsupported:        unsupported,
		ConvertFromClojure: convertFromClojure,
		ConvertToClojure:   "GoObject(%s%s)",
	}
	GoTypes[fullGoName] = v
	return v
}

func toGoExprString(src *godb.GoFile, e *Expr) string {
	if e == nil {
		return "-"
	}
	t := toGoExprInfo(src, e)
	if t != nil {
		return t.FullGoName
	}
	return fmt.Sprintf("%T", e)
}

func toGoExprTypeName(src *godb.GoFile, e *Expr) string {
	if e == nil {
		return "-"
	}
	t := toGoExprInfo(src, e)
	if t != nil {
		return t.LocalName
	}
	return fmt.Sprintf("%T", e)
}

func lenString(len *Expr) string {
	if len == nil || *len == nil {
		return ""
	}
	l := *len
	switch n := l.(type) {
	case *Ident:
		return n.Name
	case *BasicLit:
		return n.Value
	}
	return fmt.Sprintf("%T", l)
}

func goArrayType(src *godb.GoFile, len *Expr, elt *Expr) *GoTypeInfo {
	var fullGoName string
	e := toGoExprInfo(src, elt)
	fullGoName = "[" + lenString(len) + "]" + e.FullGoName
	if v, ok := GoTypes[fullGoName]; ok {
		return v
	}
	v := &GoTypeInfo{
		LocalName:        e.LocalName,
		FullGoName:       fullGoName,
		UnderlyingType:   elt,
		Custom:           true,
		Unsupported:      e.Unsupported,
		Constructs:       e.Constructs,
		ConvertToClojure: "GoObject(%s%s)",
		Exported:         e.Exported,
	}
	GoTypes[fullGoName] = v
	return v
}

func ptrTo(expr string) string {
	if expr[0] == '*' {
		return expr[1:]
	}
	return expr
}

func goStarExpr(src *godb.GoFile, x *Expr) *GoTypeInfo {
	e := toGoExprInfo(src, x)
	fullGoName := "*" + e.FullGoName
	if v, ok := GoTypes[fullGoName]; ok {
		return v
	}
	convertFromClojure := ""
	if e.ConvertFromClojure != "" {
		if e.Constructs {
			convertFromClojure = e.ConvertFromClojure
		} else if e.ArgClojureArgType == e.ArgExtractFunc {
			/* Not a conversion, so can take address of the Clojure object's internals. */
			convertFromClojure = "&" + e.ConvertFromClojure
		}
	}
	v := &GoTypeInfo{
		LocalName:          e.LocalName,
		FullGoName:         fullGoName,
		UnderlyingType:     x,
		ConvertFromClojure: convertFromClojure,
		Custom:             true,
		Exported:           e.Exported,
		Unsupported:        e.Unsupported,
		ConvertToClojure:   "GoObject(%s%s)",
	}
	GoTypes[fullGoName] = v
	return v
}

func init() {
	GoTypes["bool"] = &GoTypeInfo{
		LocalName:            "bool",
		FullGoName:           "bool",
		ArgClojureType:       "Boolean",
		ArgFromClojureObject: ".B",
		ArgClojureArgType:    "Boolean",
		ArgExtractFunc:       "Boolean",
		ConvertFromClojure:   "ToBool(%s)",
		ConvertFromMap:       "FieldAsBoolean(%s, %s)",
		ConvertToClojure:     "Boolean(%s%s)",
		PromoteType:          "%s",
		Exported:             true,
	}
	GoTypes["string"] = &GoTypeInfo{
		LocalName:            "string",
		FullGoName:           "string",
		ArgClojureType:       "String",
		ArgFromClojureObject: ".S",
		ArgClojureArgType:    "String",
		ArgExtractFunc:       "String",
		ConvertFromClojure:   `AssertString(%s, "").S`,
		ConvertFromMap:       `FieldAsString(%s, %s)`,
		ConvertToClojure:     "String(%s%s)",
		PromoteType:          "%s",
		Exported:             true,
	}
	GoTypes["rune"] = &GoTypeInfo{
		LocalName:            "rune",
		FullGoName:           "rune",
		ArgClojureType:       "Char",
		ArgFromClojureObject: ".Ch",
		ArgClojureArgType:    "Char",
		ArgExtractFunc:       "Char",
		ConvertFromClojure:   `AssertChar(%s, "").Ch`,
		ConvertFromMap:       `FieldAsChar(%s, %s)`,
		ConvertToClojure:     "Char(%s%s)",
		PromoteType:          "%s",
		Exported:             true,
	}
	GoTypes["byte"] = &GoTypeInfo{
		LocalName:            "byte",
		FullGoName:           "byte",
		ArgClojureType:       "Number",
		ArgFromClojureObject: ".Int().I",
		ArgClojureArgType:    "Int",
		ArgExtractFunc:       "Byte",
		ConvertFromClojure:   `byte(AssertInt(%s, "").I)`,
		ConvertFromMap:       `FieldAsByte(%s, %s)`,
		ConvertToClojure:     "Int(int(%s)%s)",
		PromoteType:          "int(%s)",
		Exported:             true,
	}
	GoTypes["int"] = &GoTypeInfo{
		LocalName:            "int",
		FullGoName:           "int",
		ArgClojureType:       "Number",
		ArgFromClojureObject: ".Int().I",
		ArgClojureArgType:    "Int",
		ArgExtractFunc:       "Int",
		ConvertFromClojure:   `AssertInt(%s, "").I`,
		ConvertFromMap:       `FieldAsInt(%s, %s)`,
		ConvertToClojure:     "Int(%s%s)",
		PromoteType:          "%s",
		Exported:             true,
	}
	GoTypes["uint"] = &GoTypeInfo{
		LocalName:            "uint",
		FullGoName:           "uint",
		ArgClojureType:       "Number",
		ArgFromClojureObject: ".Int().I",
		ArgClojureArgType:    "Number",
		ArgExtractFunc:       "Uint",
		ConvertFromClojure:   `uint(AssertInt(%s, "").I)`,
		ConvertFromMap:       `FieldAsUint(%s, %s)`,
		ConvertToClojure:     "BigIntU(uint64(%s)%s)",
		PromoteType:          "uint64(%s)",
		Exported:             true,
	}
	GoTypes["int8"] = &GoTypeInfo{
		LocalName:            "int8",
		FullGoName:           "int8",
		ArgClojureType:       "Int",
		ArgFromClojureObject: ".Int().I",
		ArgClojureArgType:    "Int",
		ArgExtractFunc:       "Int8",
		ConvertFromClojure:   `int8(AssertInt(%s, "").I)`,
		ConvertFromMap:       `FieldAsInt8(%s, %s)`,
		ConvertToClojure:     "Int(int(%s)%s)",
		PromoteType:          "int(%s)",
		Exported:             true,
	}
	GoTypes["uint8"] = &GoTypeInfo{
		LocalName:            "uint8",
		FullGoName:           "uint8",
		ArgClojureType:       "Int",
		ArgFromClojureObject: ".Int().I",
		ArgClojureArgType:    "Int",
		ArgExtractFunc:       "Uint8",
		ConvertFromClojure:   `uint8(AssertInt(%s, "").I)`,
		ConvertFromMap:       `FieldAsUint8(%s, %s)`,
		ConvertToClojure:     "Int(int(%s)%s)",
		PromoteType:          "int(%s)",
		Exported:             true,
	}
	GoTypes["int16"] = &GoTypeInfo{
		LocalName:            "int16",
		FullGoName:           "int16",
		ArgClojureType:       "Number",
		ArgFromClojureObject: ".Int().I",
		ArgClojureArgType:    "Int",
		ArgExtractFunc:       "Int16",
		ConvertFromClojure:   `int16(AssertInt(%s, "").I)`,
		ConvertFromMap:       `FieldAsInt16(%s, %s)`,
		ConvertToClojure:     "Int(int(%s)%s)",
		PromoteType:          "int(%s)",
		Exported:             true,
	}
	GoTypes["uint16"] = &GoTypeInfo{
		LocalName:            "uint16",
		FullGoName:           "uint16",
		ArgClojureType:       "Number",
		ArgFromClojureObject: ".Int().I",
		ArgClojureArgType:    "Int",
		ArgExtractFunc:       "Uint16",
		ConvertFromClojure:   `uint16(AssertInt(%s, "").I)`,
		ConvertFromMap:       `FieldAsUint16(%s, %s)`,
		ConvertToClojure:     "Int(int(%s)%s)",
		PromoteType:          "int(%s)",
		Exported:             true,
	}
	GoTypes["int32"] = &GoTypeInfo{
		LocalName:            "int32",
		FullGoName:           "int32",
		ArgClojureType:       "Number",
		ArgFromClojureObject: ".Int().I",
		ArgClojureArgType:    "Int",
		ArgExtractFunc:       "Int32",
		ConvertFromClojure:   `int32(AssertInt(%s, "").I)`,
		ConvertFromMap:       `FieldAsInt32(%s, %s)`,
		ConvertToClojure:     "Int(int(%s)%s)",
		PromoteType:          "int(%s)",
		Exported:             true,
	}
	GoTypes["uint32"] = &GoTypeInfo{
		LocalName:            "uint32",
		FullGoName:           "uint32",
		ArgClojureType:       "Number",
		ArgFromClojureObject: ".Int().I",
		ArgClojureArgType:    "Number",
		ArgExtractFunc:       "Uint32",
		ConvertFromClojure:   `uint32(AssertNumber(%s, "").BigInt().Uint64())`,
		ConvertFromMap:       `FieldAsUint32(%s, %s)`,
		ConvertToClojure:     "BigIntU(uint64(%s)%s)",
		PromoteType:          "int64(%s)",
		Exported:             true,
	}
	GoTypes["int64"] = &GoTypeInfo{
		LocalName:            "int64",
		FullGoName:           "int64",
		ArgClojureType:       "Number",
		ArgFromClojureObject: ".BigInt().Int64()",
		ArgClojureArgType:    "Number",
		ArgExtractFunc:       "Int64",
		ConvertFromClojure:   `AssertNumber(%s, "").BigInt().Int64()`,
		ConvertFromMap:       `FieldAsInt64(%s, %s)`,
		ConvertToClojure:     "BigInt(%s%s)",
		PromoteType:          "int64(%s)", // constants are not auto-promoted, so promote them explicitly for MakeNumber()
		Exported:             true,
	}
	GoTypes["uint64"] = &GoTypeInfo{
		LocalName:            "uint64",
		FullGoName:           "uint64",
		ArgClojureType:       "Number",
		ArgFromClojureObject: ".BigInt().Uint64()",
		ArgClojureArgType:    "Number",
		ArgExtractFunc:       "Uint64",
		ConvertFromClojure:   `AssertNumber(%s, "").BigInt().Uint64()`,
		ConvertFromMap:       `FieldAsUint64(%s, %s)`,
		ConvertToClojure:     "BigIntU(%s%s)",
		PromoteType:          "uint64(%s)", // constants are not auto-promoted, so promote them explicitly for MakeNumber()
		Exported:             true,
	}
	GoTypes["uintptr"] = &GoTypeInfo{
		LocalName:            "uintptr",
		FullGoName:           "uintptr",
		ArgClojureType:       "Number",
		ArgFromClojureObject: ".BigInt().Uint64()",
		ArgClojureArgType:    "Number",
		ArgExtractFunc:       "UintPtr",
		ConvertFromClojure:   `uintptr(AssertNumber(%s, "").BigInt().Uint64())`,
		ConvertFromMap:       `FieldAsUintPtr(%s, %s)`,
		PromoteType:          "int64(%s)",
		Exported:             true,
	}
	GoTypes["float32"] = &GoTypeInfo{
		LocalName:            "float32",
		FullGoName:           "float32",
		ArgClojureType:       "Double",
		ArgFromClojureObject: "",
		ArgClojureArgType:    "Double",
		ArgExtractFunc:       "ABEND007(find these)",
		ConvertFromClojure:   `float32(AssertDouble(%s, "").D)`,
		ConvertFromMap:       `FieldAsDouble(%s, %s)`,
		PromoteType:          "double(%s)",
		Exported:             true,
	}
	GoTypes["float64"] = &GoTypeInfo{
		LocalName:            "float64",
		FullGoName:           "float64",
		ArgClojureType:       "Double",
		ArgFromClojureObject: "",
		ArgClojureArgType:    "Double",
		ArgExtractFunc:       "ABEND007(find these)",
		ConvertFromClojure:   `float64(AssertDouble(%s, "").D)`,
		ConvertFromMap:       `FieldAsDouble(%s, %s)`,
		PromoteType:          "%s",
		Exported:             true,
	}
	GoTypes["complex64"] = &GoTypeInfo{
		LocalName:            "complex64",
		FullGoName:           "complex64",
		ArgClojureType:       "",
		ArgFromClojureObject: "",
		ArgClojureArgType:    "",
		ArgExtractFunc:       "ABEND007(find these)",
		ConvertFromClojure:   "", // TODO: support this in Joker, even if via just [real imag]
		ConvertFromMap:       "", // TODO: support this in Joker, even if via just [real imag]
		Exported:             true,
	}
	GoTypes["complex128"] = &GoTypeInfo{
		LocalName:            "complex128",
		FullGoName:           "complex128",
		ArgClojureType:       "",
		ArgFromClojureObject: "",
		ArgClojureArgType:    "",
		ArgExtractFunc:       "ABEND007(find these)",
		ConvertFromClojure:   "", // TODO: support this in Joker, even if via just [real imag]
		Exported:             true,
	}
	GoTypes["error"] = &GoTypeInfo{
		LocalName:                 "error",
		FullGoName:                "error",
		ArgClojureType:            "Error",
		ArgFromClojureObject:      "",
		ArgClojureArgType:         "Error",
		ArgExtractFunc:            "Error",
		ConvertFromClojure:        `_errors.New(AssertString(%s, "").S)`,
		ConvertFromMap:            `FieldAsError(%s, %s)`,
		ConvertFromClojureImports: []imports.Import{{Local: "_errors", LocalRef: "_errors", Full: "errors"}},
		ConvertToClojure:          "Error(%s%s)",
		Nullable:                  true,
		Exported:                  true,
	}
}

func init() {
	RegisterType_func = registerType // TODO: Remove this kludge (allowing gowalk to call this fn) when able
}
