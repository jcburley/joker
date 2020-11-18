package gtypes

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/utils"
	. "go/ast"
	"go/token"
	"go/types"
	"path"
	"strings"
)

const Concrete = ^uint(0) /* MaxUint */

var NumExprHits uint

// Info from the definition of the type (if any)
type GoType struct {
	Type             Expr      // The actual type (if any)
	TypeSpec         *TypeSpec // The definition of the named type (if any)
	IsExported       bool
	Doc              string
	DefPos           token.Pos
	GoFile           *godb.GoFile
	GoPattern        string // E.g. "%s", "*%s" (for reference types), "[]%s" (for array types)
	GoPackage        string // E.g. a/b/c (always Unix style)
	GoName           string // Base name of type (without any prefix/pattern applied)
	underlyingGoType *GoType
	Ord              uint // Slot in []*GoTypeInfo and position of case statement in big switch in goswitch.go
	Specificity      uint // Concrete means concrete type; else # of methods defined for interface{} (abstract) type
	Nullable         bool
}

type Info struct {
	FullName         string // E.g. "bool", "*net.Listener", "[]net/url.Userinfo"
	Pattern          string // E.g. "%s", "*%s" (for reference types), "[]%s" (for array types)
	Package          string // E.g. "net/url", "" (always Unix style)
	LocalName        string // E.g. "*Listener"
	UnderlyingGoType *Info
	Doc              string
	DefPos           token.Pos
	GoFile           *godb.GoFile
	TypeSpec         *TypeSpec // The definition of the named type (if any)
	Ord              uint      // Slot in []*GoTypeInfo and position of case statement in big switch in goswitch.go
	Specificity      uint      // Concrete means concrete type; else # of methods defined for interface{} (abstract) type
	IsNullable       bool      // Can an instance of the type == nil (e.g. 'error' type)?
	IsExported       bool
}

func combine(pkg, name string) string {
	if pkg == "" {
		return name
	}
	return pkg + "." + name
}

var fullNameToInfo = map[string]*Info{}

func GetInfo(pattern, pkg, name string, nullable bool) *Info {
	if pattern == "" {
		pattern = "%s"
	}
	fullName := fmt.Sprintf(pattern, combine(pkg, name))

	if info, found := fullNameToInfo[fullName]; found {
		return info
	}

	info := &Info{
		FullName:   fullName,
		Pattern:    pattern,
		Package:    pkg,
		LocalName:  fmt.Sprintf(pattern, name),
		IsNullable: nullable,
		IsExported: pkg == "" || IsExported(name),
	}

	fullNameToInfo[fullName] = info

	return info
}

var Nil = Info{}

var Error = GetInfo("", "", "error", true)

var Bool = GetInfo("", "", "bool", false)

var Byte = GetInfo("", "", "byte", false)

var Rune = GetInfo("", "", "rune", false)

var String = GetInfo("", "", "string", false)

var Int = GetInfo("", "", "int", false)

var Int32 = GetInfo("", "", "int32", false)

var Int64 = GetInfo("", "", "int64", false)

var UInt = GetInfo("", "", "uint", false)

var UInt8 = GetInfo("", "", "uint8", false)

var UInt16 = GetInfo("", "", "uint16", false)

var UInt32 = GetInfo("", "", "uint32", false)

var UInt64 = GetInfo("", "", "uint64", false)

var UIntPtr = GetInfo("", "", "uintptr", false)

var Float32 = GetInfo("", "", "float32", false)

var Float64 = GetInfo("", "", "float64", false)

var Complex128 = GetInfo("", "", "complex128", false)

var gtToInfo = map[*GoType]*Info{}

func TypeInfoForExpr(e Expr) *Info {
	gt := TypeLookup(e)
	if gt == nil {
		panic(fmt.Sprintf("cannot find type for %v", e))
	}

	if gti, found := gtToInfo[gt]; found {
		return gti
	}

	gti := GetInfo(gt.GoPattern, gt.GoPackage, gt.GoName, gt.Nullable)
	gtToInfo[gt] = gti

	// fmt.Fprintf(os.Stderr, "gtypes.TypeInfoForExpr(%T) => \"%s\"\n", e, gti.FullName)

	return gti
}

func specificityOfInterface(ts *InterfaceType) uint {
	var sp uint
	for _, m := range ts.Methods.List {
		if m.Names != nil {
			sp += (uint)(len(m.Names))
			continue
		}
		ts := godb.Resolve(m.Type)
		if ts == nil {
			continue
		}
		sp += specificityOfInterface(ts.(*TypeSpec).Type.(*InterfaceType))
	}
	return sp
}

func specificity(ts *TypeSpec) uint {
	if iface, ok := ts.Type.(*InterfaceType); ok {
		return specificityOfInterface(iface)
	}
	return Concrete
}

// Maps type-defining Expr or string to exactly one struct describing that type
var typesByExpr = map[Expr]*GoType{}
var typesByFullName = map[string]*GoType{}

func define(tdi *GoType) *Info {
	name := fmt.Sprintf(tdi.GoPattern, combine(tdi.GoPackage, tdi.GoName))
	if existingTdi, ok := typesByFullName[name]; ok {
		// fmt.Fprintf(os.Stderr, "gtypes.define(): already defined type %s at %s (%p) and again at %s (%p)\n", name, godb.WhereAt(existingTdi.DefPos), existingTdi, godb.WhereAt(tdi.DefPos), tdi)
		tdi = existingTdi
	} else {
		typesByFullName[name] = tdi
	}

	if tdi.Type != nil {
		tdiByExpr, found := typesByExpr[tdi.Type]
		if found && tdiByExpr != tdi {
			panic(fmt.Sprintf("different expr for type %s", name))
		}
		typesByExpr[tdi.Type] = tdi
	}

	var ugt *Info = nil
	if tdi.underlyingGoType != nil {
		ugt = gtToInfo[tdi.underlyingGoType]
	}
	gti := &Info{
		FullName:         name,
		Pattern:          tdi.GoPattern,
		Package:          tdi.GoPackage,
		LocalName:        tdi.GoName,
		UnderlyingGoType: ugt,
		Doc:              tdi.Doc,
		DefPos:           tdi.DefPos,
		GoFile:           tdi.GoFile,
		TypeSpec:         tdi.TypeSpec,
		Ord:              tdi.Ord,
		Specificity:      tdi.Specificity,
		IsNullable:       tdi.Nullable,
		IsExported:       tdi.IsExported,
	}
	fullNameToInfo[name] = gti
	gtToInfo[tdi] = gti

	return gti
}

func defineVariant(pattern string, innerTdi *GoType, te Expr) *GoType {
	tdi := &GoType{
		Type:             te,
		IsExported:       innerTdi.IsExported,
		GoPattern:        pattern,
		GoPackage:        innerTdi.GoPackage,
		GoName:           innerTdi.GoName,
		DefPos:           innerTdi.DefPos,
		underlyingGoType: innerTdi,
	}

	define(tdi)

	return tdi
}

func TypeDefineBuiltin(name string, nullable bool) *GoType {
	tdi := &GoType{
		Type:       &Ident{Name: name},
		IsExported: true,
		GoPattern:  "%s",
		GoName:     name,
		Nullable:   nullable,
	}

	define(tdi)

	return tdi
}

func TypeDefine(ts *TypeSpec, gf *godb.GoFile, parentDoc *CommentGroup) []*Info {
	localName := ts.Name.Name

	doc := ts.Doc // Try block comments for this specific decl
	if doc == nil {
		doc = ts.Comment // Use line comments if no preceding block comments are available
	}
	if doc == nil {
		doc = parentDoc // Use 'var'/'const' statement block comments as last resort
	}

	types := []*Info{}

	tdi := &GoType{
		Type:        ts.Type,
		TypeSpec:    ts,
		IsExported:  IsExported(localName),
		Doc:         utils.CommentGroupAsString(doc),
		DefPos:      ts.Name.NamePos,
		GoFile:      gf,
		GoPattern:   "%s",
		GoPackage:   godb.GoPackageForTypeSpec(ts),
		GoName:      localName,
		Specificity: specificity(ts),
	}
	ti := define(tdi)
	types = append(types, ti)

	if tdi.Specificity == Concrete {
		// Concrete types get reference-to variants, allowing Joker code to access them.
		tdiPtrTo := &GoType{
			Type:             &StarExpr{X: tdi.Type},
			IsExported:       tdi.IsExported,
			Doc:              "",
			DefPos:           tdi.DefPos,
			GoPattern:        fmt.Sprintf(tdi.GoPattern, "*%s"),
			GoPackage:        tdi.GoPackage,
			GoName:           tdi.GoName,
			underlyingGoType: tdi,
			Specificity:      Concrete,
		}
		ti = define(tdiPtrTo)
		types = append(types, ti)
	}

	return types
}

func TypeLookup(e Expr) (ty *GoType) {
	if tdi, ok := typesByExpr[e]; ok {
		NumExprHits++
		return tdi
	}

	goName := ""

	if id, yes := e.(*Ident); yes {
		goName = id.Name
		if tdi, ok := typesByFullName[goName]; ok {
			typesByExpr[e] = tdi
			return tdi
		}
		tdi := &GoType{
			Type:       e,
			IsExported: true,
			DefPos:     id.Pos(),
			GoPattern:  "%s",
			GoName:     goName,
		}
		typesByExpr[e] = tdi
		typesByFullName[goName] = tdi
		return tdi
	}

	var innerTdi *GoType
	pattern := "%s"

	switch v := e.(type) {
	case *StarExpr:
		innerTdi = TypeLookup(v.X)
		pattern = "*%s"
		goName = innerTdi.GoName
	case *ArrayType:
		innerTdi = TypeLookup(v.Elt)
		len := exprToString(v.Len)
		pattern = "[" + len + "]%s"
		if innerTdi == nil {
			goName = fmt.Sprintf("%v", v.Elt)
		} else {
			goName = innerTdi.GoName
		}
	case *InterfaceType:
		goName = "interface{"
		methods := methodsToString(v.Methods.List)
		if v.Incomplete {
			methods = strings.Join([]string{methods, "..."}, ", ")
		}
		goName += methods + "}"
	case *MapType:
		key := TypeLookup(v.Key)
		value := TypeLookup(v.Value)
		goName = "map[" + key.RelativeGoName(e.Pos()) + "]" + value.RelativeGoName(e.Pos())
	case *SelectorExpr:
		left := fmt.Sprintf("%s", v.X)
		goName = left + "." + v.Sel.Name
	case *ChanType:
		ty := TypeLookup(v.Value)
		goName = "chan"
		switch v.Dir & (SEND | RECV) {
		case SEND:
			goName += "<-"
		case RECV:
			goName = "<-" + goName
		default:
		}
		goName += " " + ty.RelativeGoName(e.Pos())
	case *StructType:
		goName = "struct{}"
	}

	if innerTdi == nil {
		if goName == "" {
			goName = fmt.Sprintf("ABEND001(NO GO NAME??!!)")
		}
		tdi := &GoType{
			Type:      e,
			GoPattern: pattern,
			GoName:    goName,
			DefPos:    e.Pos(),
		}
		define(tdi)
		return tdi
	}

	return defineVariant(pattern, innerTdi, e)
}

func fieldToString(f *Field) string {
	ti := TypeLookup(f.Type)
	// Don't bother implementing this until it's actually needed:
	return "ABEND041(gtypes.go/fieldToString found something: " + ti.GoName + "!)"
}

func methodsToString(methods []*Field) string {
	mStrings := make([]string, len(methods))
	for i, m := range methods {
		mStrings[i] = fieldToString(m)
	}
	return strings.Join(mStrings, ", ")
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

func (tdi *GoType) TypeReflected() (packageImport, pattern string) {
	t := ""
	suffix := ".Elem()"
	if tdiu := tdi.underlyingGoType; tdiu != nil {
		t = "_" + path.Base(tdiu.GoPackage) + "." + fmt.Sprintf(tdi.GoPattern, tdi.GoName)
		suffix = ""
	} else {
		t = "_" + path.Base(tdi.GoPackage) + "." + fmt.Sprintf(tdi.GoPattern, tdi.GoName)
	}
	return "reflect", fmt.Sprintf("%%s.TypeOf((*%s)(nil))%s", t, suffix)
}

func (tdi *GoType) RelativeGoName(pos token.Pos) string {
	pkgPrefix := tdi.GoPackage
	if pkgPrefix == godb.GoPackageForPos(pos) {
		pkgPrefix = ""
	} else if pkgPrefix != "" {
		pkgPrefix += "."
	}
	return fmt.Sprintf(tdi.GoPattern, pkgPrefix+tdi.GoName)
}

func (tdi *GoType) AbsoluteGoName() string {
	pkgPrefix := tdi.GoPackage
	if pkgPrefix != "" {
		pkgPrefix += "."
	}
	return fmt.Sprintf(tdi.GoPattern, pkgPrefix+tdi.GoName)
}

func TypeName(e Expr) string {
	switch x := e.(type) {
	case *Ident:
		break
	case *ArrayType:
		return "[" + goExprToString(x.Len) + "]" + TypeName(x.Elt)
	case *StarExpr:
		return "*" + TypeName(x.X)
	case *MapType:
		return "map[" + TypeName(x.Key) + "]" + TypeName(x.Value)
	case *SelectorExpr:
		return fmt.Sprintf("%s", x.X) + "." + x.Sel.Name
	default:
		return fmt.Sprintf("ABEND699(types.go:TypeName: unrecognized node %T)", e)
	}

	x := e.(*Ident)
	local := x.Name
	prefix := ""
	if types.Universe.Lookup(local) == nil {
		prefix = godb.GoPackageForExpr(e) + "."
	}

	return prefix + local
}

func goExprToString(e Expr) string {
	if e == nil {
		return ""
	}
	switch v := e.(type) {
	case *Ellipsis:
		return "..." + goExprToString(v.Elt)
	case *BasicLit:
		return v.Value
	}
	return fmt.Sprintf("%v", e)
}
