package gtypes

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/genutils"
	"github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/paths"
	. "go/ast"
	"go/token"
	"os"
	"path"
	"strings"
)

const Concrete = ^uint(0) /* MaxUint */

var NumExprHits uint

type Info struct {
	Expr           Expr      // [key] The canonical referencing expression (if any)
	FullName       string    // [key] E.g. "bool", "*net.Listener", "[]net/url.Userinfo"
	who            string    // who made me
	Type           Expr      // The actual type (if any)
	TypeSpec       *TypeSpec // The definition of the named type (if any)
	UnderlyingType *Info
	Pattern        string // E.g. "%s", "*%s" (for reference types), "[]%s" (for array types)
	Package        string // E.g. "net/url", "" (always Unix style)
	LocalName      string // E.g. "Listener"
	Doc            string
	DefPos         token.Pos
	File           *godb.GoFile
	Specificity    uint // Concrete means concrete type; else # of methods defined for interface{} (abstract) type
	IsNullable     bool // Can an instance of the type == nil (e.g. 'error' type)?
	IsExported     bool
	IsBuiltin      bool
}

func Combine(pkg, name string) string {
	if pkg == "" {
		return name
	}
	return pkg + "." + name
}

// Maps type-defining Expr or string to exactly one struct describing that type
var typesByExpr = map[Expr]*Info{}
var typesByFullName = map[string]*Info{}

func getInfo(pattern, pkg, name string, nullable bool) *Info {
	if pattern == "" {
		pattern = "%s"
	}
	fullName := fmt.Sprintf(pattern, Combine(pkg, name))

	if info, found := typesByFullName[fullName]; found {
		return info
	}

	info := &Info{
		who:        "getInfo",
		FullName:   fullName,
		Pattern:    pattern,
		Package:    pkg,
		LocalName:  fmt.Sprintf(pattern, name),
		IsNullable: nullable,
		IsExported: pkg == "" || IsExported(name),
		IsBuiltin:  true,
	}

	typesByFullName[fullName] = info

	return info
}

var Nil = Info{}

var Error = getInfo("", "", "error", true)

var Bool = getInfo("", "", "bool", false)

var Byte = getInfo("", "", "byte", false)

var Rune = getInfo("", "", "rune", false)

var String = getInfo("", "", "string", false)

var Int = getInfo("", "", "int", false)

var Int8 = getInfo("", "", "int8", false)

var Int16 = getInfo("", "", "int16", false)

var Int32 = getInfo("", "", "int32", false)

var Int64 = getInfo("", "", "int64", false)

var UInt = getInfo("", "", "uint", false)

var UInt8 = getInfo("", "", "uint8", false)

var UInt16 = getInfo("", "", "uint16", false)

var UInt32 = getInfo("", "", "uint32", false)

var UInt64 = getInfo("", "", "uint64", false)

var UIntPtr = getInfo("", "", "uintptr", false)

var Float32 = getInfo("", "", "float32", false)

var Float64 = getInfo("", "", "float64", false)

var Complex128 = getInfo("", "", "complex128", false)

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

func (ti *Info) computeFullName() string {
	n := ti.FullName
	if n == "" {
		n = fmt.Sprintf(ti.Pattern, Combine(ti.Package, ti.LocalName))
		ti.FullName = n
	}
	return n
}

func finish(ti *Info) {
	fullName := ti.computeFullName()

	if _, ok := typesByFullName[fullName]; ok {
		fmt.Fprintf(os.Stderr, "") // "gtypes.finish(): already seen/defined type %s at %s (%p) and again at %s (%p)\n", fullName, godb.WhereAt(existingTi.DefPos), existingTi, godb.WhereAt(ti.DefPos), ti)
		return
	}

	typesByFullName[fullName] = ti

	if e := ti.Expr; e != nil {
		tiByExpr, found := typesByExpr[e]
		if found && tiByExpr != ti {
			panic(fmt.Sprintf("different expr for type %s", fullName))
		}
		typesByExpr[e] = ti
	}
}

func defineAndFinish(ti *Info) {
	// Might be more to do here at some point.
	finish(ti)
}

func finishVariant(pattern string, innerInfo *Info, te Expr) *Info {
	ti := &Info{
		Expr:           te,
		who:            "finishVariant",
		IsExported:     innerInfo.IsExported,
		File:           innerInfo.File,
		Pattern:        pattern,
		Package:        innerInfo.Package,
		LocalName:      innerInfo.LocalName,
		DefPos:         innerInfo.DefPos,
		UnderlyingType: innerInfo,
		Specificity:    Concrete,
	}

	finish(ti)

	return ti
}

func Define(ts *TypeSpec, gf *godb.GoFile, parentDoc *CommentGroup) []*Info {
	localName := ts.Name.Name

	doc := ts.Doc // Try block comments for this specific decl
	if doc == nil {
		doc = ts.Comment // Use line comments if no preceding block comments are available
	}
	if doc == nil {
		doc = parentDoc // Use 'var'/'const' statement block comments as last resort
	}

	types := []*Info{}

	ti := &Info{
		who:         "TypeDefine",
		Type:        ts.Type,
		TypeSpec:    ts,
		IsExported:  IsExported(localName),
		Doc:         genutils.CommentGroupAsString(doc),
		DefPos:      ts.Name.NamePos,
		File:        gf,
		Pattern:     "%s",
		Package:     godb.GoPackageForTypeSpec(ts),
		LocalName:   localName,
		Specificity: specificity(ts),
	}
	defineAndFinish(ti)
	types = append(types, ti)

	if ti.Specificity == Concrete {
		// Concrete types get reference-to variants, allowing Joker code to access them.
		tiPtrTo := &Info{
			Expr:           &StarExpr{X: nil},
			who:            "*TypeDefine*",
			Type:           &StarExpr{X: ti.Type},
			IsExported:     ti.IsExported,
			Doc:            ti.Doc,
			DefPos:         ti.DefPos,
			File:           gf,
			Pattern:        fmt.Sprintf(ti.Pattern, "*%s"),
			Package:        ti.Package,
			LocalName:      ti.LocalName,
			UnderlyingType: ti,
			Specificity:    Concrete,
		}
		finish(tiPtrTo)
		types = append(types, tiPtrTo)
	}

	return types
}

func InfoForName(fullName string) *Info {
	if ti, ok := typesByFullName[fullName]; ok {
		return ti
	}
	return nil
}

func InfoForExpr(e Expr) *Info {
	if ti, ok := typesByExpr[e]; ok {
		NumExprHits++
		return ti
	}

	localName := ""
	fullName := ""

	if id, yes := e.(*Ident); yes {
		pkg := godb.GoPackageForExpr(e)
		fullName = Combine(pkg, id.Name)
		if ti, ok := typesByFullName[fullName]; ok {
			typesByExpr[e] = ti
			return ti
		}
		ti := &Info{
			Expr:       e,
			who:        "TypeForExpr",
			IsExported: true,
			DefPos:     id.Pos(),
			FullName:   fullName,
			Pattern:    "%s",
			Package:    pkg,
			LocalName:  id.Name,
		}

		typesByExpr[e] = ti
		finish(ti)

		return ti
	}

	var innerInfo *Info
	pattern := "%s"

	switch v := e.(type) {
	case *StarExpr:
		innerInfo = InfoForExpr(v.X)
		pattern = fmt.Sprintf("*%s", innerInfo.Pattern)
		localName = innerInfo.LocalName
	case *ArrayType:
		innerInfo = InfoForExpr(v.Elt)
		len := exprToString(v.Len)
		pattern = "[" + len + "]%s"
		if innerInfo == nil {
			localName = fmt.Sprintf("%v", v.Elt)
		} else {
			localName = innerInfo.LocalName
			pattern = fmt.Sprintf(pattern, innerInfo.Pattern)
		}
	case *InterfaceType:
		localName = "interface{"
		methods := methodsToString(v.Methods.List)
		if v.Incomplete {
			methods = strings.Join([]string{methods, "..."}, ", ")
		}
		localName += methods + "}"
	case *MapType:
		key := InfoForExpr(v.Key)
		value := InfoForExpr(v.Value)
		localName = "map[" + key.RelativeName(e.Pos()) + "]" + value.RelativeName(e.Pos())
	case *SelectorExpr:
		pkgName := v.X.(*Ident).Name
		localName = v.Sel.Name
		fullPathUnix := paths.Unix(godb.FileAt(v.Pos()))
		rf := godb.GoFileForExpr(v)
		if fullPkgName, found := (*rf.Spaces)[pkgName]; found {
			if !godb.IsAvailable(fullPkgName) {
				localName = fmt.Sprintf("ABEND002(reference to unavailable package `%s' looking for type `%s')", fullPkgName, localName)
			}
			fullName = fullPkgName.String() + "." + localName
		} else {
			panic(fmt.Sprintf("processing %s: could not find %s in %s",
				godb.WhereAt(v.Pos()), pkgName, fullPathUnix))
		}
	case *ChanType:
		ty := InfoForExpr(v.Value)
		localName = "chan"
		switch v.Dir & (SEND | RECV) {
		case SEND:
			localName += "<-"
		case RECV:
			localName = "<-" + localName
		default:
		}
		localName += " " + ty.RelativeName(e.Pos())
	case *StructType:
		localName = "struct{}" // TODO: add more info here
	case *FuncType:
		localName = "func{}" // TODO: add more info here
	}

	if innerInfo == nil {
		if localName == "" && fullName == "" {
			localName = fmt.Sprintf("ABEND001(gtypes.go:NO GO NAME for %T)", e)
		}
		if fullName != "" {
			if ti, ok := typesByFullName[fullName]; ok {
				return ti
			}
		}
		ti := &Info{
			Expr:           e,
			who:            fmt.Sprintf("[InfoForExpr %T]", e),
			Pattern:        pattern,
			FullName:       fullName,
			LocalName:      localName,
			DefPos:         e.Pos(),
			UnderlyingType: innerInfo,
		}
		finish(ti)
		return ti
	}

	return finishVariant(pattern, innerInfo, e)
}

func fieldToString(f *Field) string {
	ti := InfoForExpr(f.Type)
	// Don't bother implementing this until it's actually needed:
	return "ABEND041(gtypes.go/fieldToString found something: " + ti.LocalName + "!)"
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

func (ti *Info) Reflected() (packageImport, pattern string) {
	t := ""
	suffix := ".Elem()"
	if tiu := ti.UnderlyingType; tiu != nil {
		t = "_" + path.Base(tiu.Package) + "." + fmt.Sprintf(ti.Pattern, ti.LocalName)
		suffix = ""
	} else {
		t = "_" + path.Base(ti.Package) + "." + fmt.Sprintf(ti.Pattern, ti.LocalName)
	}
	return "reflect", fmt.Sprintf("%%s.TypeOf((*%s)(nil))%s", t, suffix)
}

func (ti *Info) RelativeName(pos token.Pos) string {
	pkgPrefix := ti.Package
	if pkgPrefix == godb.GoPackageForPos(pos) {
		pkgPrefix = ""
	} else if pkgPrefix != "" {
		pkgPrefix += "."
	}
	return fmt.Sprintf(ti.Pattern, pkgPrefix+ti.LocalName)
}

func (ti *Info) AbsoluteName() string {
	pkgPrefix := ti.Package
	if pkgPrefix != "" {
		pkgPrefix += "."
	}
	return fmt.Sprintf(ti.Pattern, pkgPrefix+ti.LocalName)
}

func (ti *Info) NameDoc(e Expr) string {
	if e != nil && godb.GoPackageForExpr(e) != ti.Package {
		return ti.FullName
	}
	return fmt.Sprintf(ti.Pattern, ti.LocalName)
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
