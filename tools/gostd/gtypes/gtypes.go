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
	DocPattern     string // E.g. "[SomeConst]%s", vs "[5]%s" in Pattern
	Doc            string
	DefPos         token.Pos
	File           *godb.GoFile
	Specificity    uint // Concrete means concrete type; else # of methods defined for interface{} (abstract) type
	IsNullable     bool // Can an instance of the type == nil (e.g. 'error' type)?
	IsExported     bool
	IsBuiltin      bool
	IsSwitchable   bool // Can type's Go name be used in a "case" statement?
	IsAddressable  bool // Is "&instance" going to pass muster, even with 'go vet'?
}

// Maps type-defining Expr or string to exactly one struct describing that type
var typesByExpr = map[Expr]*Info{}
var typesByFullName = map[string]*Info{}

func getInfo(fullName string, nullable bool) *Info {
	if info, found := typesByFullName[fullName]; found {
		return info
	}

	info := &Info{
		who:           "getInfo",
		FullName:      fullName,
		Pattern:       "%s",
		Package:       "",
		LocalName:     fullName,
		DocPattern:    "%s",
		IsNullable:    nullable,
		IsExported:    true,
		IsBuiltin:     true,
		IsSwitchable:  true,
		IsAddressable: true,
	}

	typesByFullName[fullName] = info

	return info
}

var Nil = Info{}

var Error = getInfo("error", true)

var Bool = getInfo("bool", false)

var Byte = getInfo("byte", false)

var Rune = getInfo("rune", false)

var String = getInfo("string", false)

var Int = getInfo("int", false)

var Int8 = getInfo("int8", false)

var Int16 = getInfo("int16", false)

var Int32 = getInfo("int32", false)

var Int64 = getInfo("int64", false)

var UInt = getInfo("uint", false)

var UInt8 = getInfo("uint8", false)

var UInt16 = getInfo("uint16", false)

var UInt32 = getInfo("uint32", false)

var UInt64 = getInfo("uint64", false)

var UIntPtr = getInfo("uintptr", false)

var Float32 = getInfo("float32", false)

var Float64 = getInfo("float64", false)

var Complex128 = getInfo("complex128", false)

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
		n = fmt.Sprintf(ti.Pattern, genutils.CombineGoName(ti.Package, ti.LocalName))
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

func finishVariant(pattern, docPattern string, switchable bool, innerInfo *Info, te Expr) *Info {
	ti := &Info{
		Expr:           te,
		who:            "finishVariant",
		IsExported:     innerInfo.IsExported,
		File:           innerInfo.File,
		Pattern:        pattern,
		Package:        innerInfo.Package,
		LocalName:      innerInfo.LocalName,
		DocPattern:     docPattern,
		DefPos:         innerInfo.DefPos,
		UnderlyingType: innerInfo,
		Specificity:    Concrete,
		IsSwitchable:   switchable && innerInfo.IsSwitchable,
		IsAddressable:  innerInfo.IsAddressable,
	}

	finish(ti)

	return ti
}

func isAddressable(pkg, name string) bool {
	// See: https://github.com/golang/go/issues/40701
	return !(pkg == "reflect" && (name == "StringHeader" || name == "SliceHeader"))
}

func Define(ts *TypeSpec, gf *godb.GoFile, parentDoc *CommentGroup) []*Info {
	localName := ts.Name.Name
	pkg := godb.GoPackageForTypeSpec(ts)

	doc := ts.Doc // Try block comments for this specific decl
	if doc == nil {
		doc = ts.Comment // Use line comments if no preceding block comments are available
	}
	if doc == nil {
		doc = parentDoc // Use 'var'/'const' statement block comments as last resort
	}

	types := []*Info{}

	ti := &Info{
		who:           "TypeDefine",
		Type:          ts.Type,
		TypeSpec:      ts,
		IsExported:    IsExported(localName),
		Doc:           genutils.CommentGroupAsString(doc),
		DefPos:        ts.Name.NamePos,
		File:          gf,
		Pattern:       "%s",
		Package:       pkg,
		LocalName:     localName,
		DocPattern:    "%s",
		Specificity:   specificity(ts),
		IsSwitchable:  ts.Assign == token.NoPos,
		IsAddressable: isAddressable(pkg, localName),
	}
	defineAndFinish(ti)
	types = append(types, ti)

	if ti.Specificity == Concrete {
		// Concrete types get reference-to variants, allowing Clojure code to access them.
		newPattern := fmt.Sprintf(ti.Pattern, "*%s")
		tiPtrTo := &Info{
			Expr:           &StarExpr{X: nil},
			who:            "*TypeDefine*",
			Type:           &StarExpr{X: ti.Type},
			IsExported:     ti.IsExported,
			Doc:            ti.Doc,
			DefPos:         ti.DefPos,
			File:           gf,
			Pattern:        newPattern,
			Package:        ti.Package,
			LocalName:      ti.LocalName,
			DocPattern:     newPattern,
			UnderlyingType: ti,
			Specificity:    Concrete,
			IsSwitchable:   ti.IsSwitchable,
			IsAddressable:  ti.IsAddressable,
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
	pkgName := ""

	if id, yes := e.(*Ident); yes {
		pkgName = godb.GoPackageForExpr(e)
		fullName = genutils.CombineGoName(pkgName, id.Name)
		if ti, ok := typesByFullName[fullName]; ok {
			typesByExpr[e] = ti
			return ti
		}
		ti := &Info{
			Expr:         e,
			who:          "TypeForExpr",
			IsExported:   true,
			DefPos:       id.Pos(),
			FullName:     fullName,
			Pattern:      "%s",
			Package:      pkgName,
			LocalName:    id.Name,
			DocPattern:   "%s",
			IsSwitchable: true,
		}

		typesByExpr[e] = ti
		finish(ti)

		return ti
	}

	var innerInfo *Info
	pattern := "%s"
	docPattern := pattern
	switchable := true

	switch v := e.(type) {
	case *StarExpr:
		innerInfo = InfoForExpr(v.X)
		pattern = fmt.Sprintf("*%s", innerInfo.Pattern)
		docPattern = fmt.Sprintf("*%s", innerInfo.DocPattern)
		localName = innerInfo.LocalName
		pkgName = innerInfo.Package
	case *ArrayType:
		innerInfo = InfoForExpr(v.Elt)
		len, docLen := intExprToString(v.Len)
		pattern = "[" + len + "]%s"
		docPattern = "[" + docLen + "]%s"
		if innerInfo == nil {
			localName = fmt.Sprintf("%v", v.Elt)
		} else {
			localName = innerInfo.LocalName
			pattern = fmt.Sprintf(pattern, innerInfo.Pattern)
			docPattern = fmt.Sprintf(docPattern, innerInfo.DocPattern)
		}
		pkgName = innerInfo.Package
		if strings.Contains(pattern, "ABEND") {
			switchable = false
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
		pkg := v.X.(*Ident).Name
		localName = v.Sel.Name
		fullPathUnix := paths.Unix(godb.FileAt(v.Pos()))
		rf := godb.GoFileForExpr(v)
		if fullPkgName, found := (*rf.Spaces)[pkg]; found {
			if !godb.IsAvailable(fullPkgName) {
				localName = fmt.Sprintf("ABEND002(reference to unavailable package `%s' looking for type `%s')", fullPkgName, localName)
			}
			pkgName = fullPkgName.String()
			fullName = pkgName + "." + localName
		} else {
			panic(fmt.Sprintf("processing %s: could not find %s in %s",
				godb.WhereAt(v.Pos()), pkg, fullPathUnix))
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
	case *Ellipsis:
		ty := InfoForExpr(v.Elt)
		localName = "..." + ty.RelativeName(e.Pos())
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
		if strings.Contains(localName, "ABEND") {
			pkgName = ""
		}
		ti := &Info{
			Expr:           e,
			who:            fmt.Sprintf("[InfoForExpr %T]", e),
			Pattern:        pattern,
			Package:        pkgName,
			FullName:       fullName,
			LocalName:      localName,
			DocPattern:     docPattern,
			DefPos:         e.Pos(),
			UnderlyingType: innerInfo,
			IsSwitchable:   switchable,
		}
		finish(ti)
		return ti
	}

	return finishVariant(pattern, docPattern, switchable, innerInfo, e)
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

func intExprToString(e Expr) (real, doc string) {
	if e == nil {
		return
	}

	res := eval(e)
	switch r := res.(type) {
	case int:
		real = fmt.Sprintf("%d", r)
	default:
		real = fmt.Sprintf("ABEND229(non-int expression %T at %s)", res, godb.WhereAt(e.Pos()))
	}

	doc = intExprToDocString(e)

	return
}

func intExprToDocString(e Expr) string {
	switch v := e.(type) {
	case *BasicLit:
		return v.Value
	case *Ident:
		return v.Name
	}
	return "???"
}

var eval func(e Expr) interface{}

func SetEvalFn(fn func(e Expr) interface{}) {
	eval = fn
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
		return fmt.Sprintf(ti.DocPattern, genutils.CombineGoName(ti.Package, ti.LocalName))
	}
	return fmt.Sprintf(ti.DocPattern, ti.LocalName)
}
