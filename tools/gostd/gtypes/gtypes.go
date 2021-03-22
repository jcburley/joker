package gtypes

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/astutils"
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
	Expr              Expr      // [key] The canonical referencing expression (if any)
	FullName          string    // [key] E.g. "bool", "*net.Listener", "[]net/url.Userinfo"
	who               string    // who made me
	Type              Expr      // The actual type (if any)
	TypeSpec          *TypeSpec // The definition of the named type (if any)
	UnderlyingType    *Info
	Pattern           string // E.g. "%s", "*%s" (for reference types), "[]%s" (for array types)
	Package           string // E.g. "net/url", "" (always Unix style)
	LocalName         string // E.g. "Listener"
	DocPattern        string // E.g. "[SomeConst]%s", vs "[5]%s" in Pattern
	Doc               string
	DefPos            token.Pos
	File              *godb.GoFile
	Specificity       uint   // Concrete means concrete type; else # of methods defined for interface{} (abstract) type
	NilPattern        string // 'nil%.s' or e.g. '%s{}'
	IsNullable        bool   // Can an instance of the type == nil (e.g. 'error' type)?
	IsExported        bool   // Builtin, typename exported, or type representable outside package (e.g. map[x.Foo][y.Bar])
	IsBuiltin         bool
	IsSwitchable      bool // Can type's Go name be used in a "case" statement?
	IsAddressable     bool // Is "&instance" going to pass muster, even with 'go vet'?
	IsPassedByAddress bool // Whether Joker passes only references to these around (excludes builtins, some complex, and interface{} types)
	IsArbitraryType   bool // Is unsafe.ArbitraryType, which gets treated as interface{}
	IsUnsupported     bool
	IsCtorable        bool // Whether a ctor for this type can (and will) be created
}

// Maps type-defining Expr or string to exactly one struct describing that type
var typesByExpr = map[Expr]*Info{}
var typesByFullName = map[string]*Info{}

func getInfo(fullName, nilPattern string, nullable bool) *Info {
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
		NilPattern:    nilPattern,
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

var Error = getInfo("error", "nil%.s", true)

var Bool = getInfo("bool", "false%.s", false)

var Byte = getInfo("byte", "0%.s", false)

var Rune = getInfo("rune", "0%.s", false)

var String = getInfo("string", "\"\"%.s", false)

var Int = getInfo("int", "0%.s", false)

var Int8 = getInfo("int8", "0%.s", false)

var Int16 = getInfo("int16", "0%.s", false)

var Int32 = getInfo("int32", "0%.s", false)

var Int64 = getInfo("int64", "0%.s", false)

var UInt = getInfo("uint", "0%.s", false)

var UInt8 = getInfo("uint8", "0%.s", false)

var UInt16 = getInfo("uint16", "0%.s", false)

var UInt32 = getInfo("uint32", "0%.s", false)

var UInt64 = getInfo("uint64", "0%.s", false)

var UIntPtr = getInfo("uintptr", "0%.s", false)

var Float32 = getInfo("float32", "0%.s", false)

var Float64 = getInfo("float64", "0%.s", false)

var Complex128 = getInfo("complex128", "0%.s", false)

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

func calculateSpecificity(ts *TypeSpec) uint {
	if iface, ok := ts.Type.(*InterfaceType); ok {
		return specificityOfInterface(iface)
	}
	return Concrete
}

func computeFullName(pattern, pkg, localName string) string {
	return fmt.Sprintf(pattern, genutils.CombineGoName(pkg, localName))
}

func insert(ti *Info) {
	fullName := ti.FullName

	if existingTi, ok := typesByFullName[fullName]; ok {
		fmt.Fprintf(os.Stderr, "gtypes.insert(): %s already seen/defined type %s at %s (%p) and again at %s (%p)\n", ti.who, fullName, godb.WhereAt(existingTi.DefPos), existingTi, godb.WhereAt(ti.DefPos), ti)
		return
	}

	typesByFullName[fullName] = ti

	if fullName == "[][]*crypto/x509.Certificate XXX DISABLED XXX" {
		fmt.Printf("gtypes.go/insert(): %s %+v\n", fullName, ti)
	}

	if strings.Contains(fullName, "ABEND") {
		ti.IsUnsupported = true
	}

	if e := ti.Expr; e != nil {
		tiByExpr, found := typesByExpr[e]
		if found {
			if tiByExpr != ti {
				panic(fmt.Sprintf("different expr for type %s", fullName))
			}
		} else {
			typesByExpr[e] = ti
		}
	}
}

func finishVariant(fullName, pattern, docPattern, nilPattern string, switchable, isPassedByAddress bool, innerInfo *Info, te Expr) *Info {
	ti := &Info{
		Expr:              te,
		FullName:          fullName,
		who:               "finishVariant",
		IsExported:        innerInfo.IsExported,
		File:              innerInfo.File,
		Pattern:           pattern,
		Package:           innerInfo.Package,
		LocalName:         innerInfo.LocalName,
		DocPattern:        docPattern,
		DefPos:            innerInfo.DefPos,
		UnderlyingType:    innerInfo,
		Specificity:       Concrete,
		NilPattern:        nilPattern,
		IsSwitchable:      switchable && innerInfo.IsSwitchable,
		IsAddressable:     innerInfo.IsAddressable,
		IsPassedByAddress: isPassedByAddress,
		IsArbitraryType:   innerInfo.IsArbitraryType,
		IsBuiltin:         innerInfo.IsBuiltin,
	}

	insert(ti)

	return ti
}

func isTypeAddressable(fullName string) bool {
	// See: https://github.com/golang/go/issues/40701
	return fullName != "reflect.StringHeader" && fullName != "reflect.SliceHeader"
}

func Define(ts *TypeSpec, gf *godb.GoFile, parentDoc *CommentGroup) []*Info {
	localName := ts.Name.Name
	pkg := godb.GoPackageForTypeSpec(ts)
	fullName := computeFullName("%s", pkg, localName)
	ti := typesByFullName[fullName]

	if gf == nil {
		gf = godb.GoFileForPos(ts.Pos())
	}

	doc := ts.Doc // Try block comments for this specific decl
	if doc == nil {
		doc = ts.Comment // Use line comments if no preceding block comments are available
	}
	if doc == nil {
		doc = parentDoc // Use 'var'/'const' statement block comments as last resort
	}

	types := []*Info{}

	var specificity uint
	isArbitraryType := false
	if fullName == "unsafe.ArbitraryType" {
		isArbitraryType = true
		specificity = 0
	} else {
		specificity = calculateSpecificity(ts)
	}

	underlyingInfo := InfoForExpr(ts.Type)
	isAddressable := isTypeAddressable(fullName) && underlyingInfo.IsAddressable

	if ti == nil {
		ti = &Info{
			Expr:              ts.Name,
			FullName:          fullName,
			who:               "TypeDefine",
			Type:              ts.Type,
			TypeSpec:          ts,
			IsExported:        IsExported(localName),
			Doc:               genutils.CommentGroupAsString(doc),
			DefPos:            ts.Name.NamePos,
			File:              gf,
			Pattern:           "%s",
			Package:           pkg,
			LocalName:         localName,
			DocPattern:        "%s",
			Specificity:       specificity,
			IsSwitchable:      ts.Assign == token.NoPos, // Aliases exactly match their targets, so both can't be in the big switch statement
			IsArbitraryType:   isArbitraryType,
			IsAddressable:     isAddressable,
			NilPattern:        underlyingInfo.NilPattern,
			IsPassedByAddress: underlyingInfo.IsPassedByAddress,
			IsCtorable:        underlyingInfo.IsAddressable,
		}
		insert(ti)
	}
	types = append(types, ti)

	if ti.Specificity == Concrete {
		// Concrete types get a reference-to variant, allowing Clojure code to access them.
		newPattern := fmt.Sprintf(ti.Pattern, "*%s")
		fullName = computeFullName(newPattern, pkg, localName)
		tiPtrTo := typesByFullName[fullName]
		if tiPtrTo == nil {
			tiPtrTo = &Info{
				Expr:              &StarExpr{X: ti.Expr},
				FullName:          fullName,
				who:               "*TypeDefine*",
				Type:              &StarExpr{X: ti.Type},
				IsExported:        ti.IsExported,
				Doc:               ti.Doc,
				DefPos:            ti.DefPos,
				File:              gf,
				Pattern:           newPattern,
				Package:           ti.Package,
				LocalName:         ti.LocalName,
				DocPattern:        newPattern,
				UnderlyingType:    ti,
				Specificity:       Concrete,
				NilPattern:        "nil%.s",
				IsSwitchable:      ti.IsSwitchable,
				IsAddressable:     ti.IsAddressable,
				IsPassedByAddress: false,
				IsArbitraryType:   isArbitraryType,
			}
			insert(tiPtrTo)
		}
		types = append(types, tiPtrTo)
	}

	newPattern := fmt.Sprintf(ti.Pattern, "[]%s")
	fullName = computeFullName(newPattern, pkg, localName)
	tiArrayOf := typesByFullName[fullName]
	if tiArrayOf == nil {
		tiArrayOf = &Info{
			Expr:              &ArrayType{Elt: ti.Expr},
			FullName:          fullName,
			who:               "*TypeDefine*",
			Type:              &ArrayType{Elt: ti.Type},
			IsExported:        ti.IsExported,
			Doc:               ti.Doc,
			DefPos:            ti.DefPos,
			File:              gf,
			Pattern:           newPattern,
			Package:           ti.Package,
			LocalName:         ti.LocalName,
			DocPattern:        newPattern,
			UnderlyingType:    ti,
			Specificity:       Concrete,
			NilPattern:        "%s{}",
			IsSwitchable:      ti.IsSwitchable,
			IsAddressable:     false,
			IsPassedByAddress: false,
			IsArbitraryType:   isArbitraryType,
		}
		insert(tiArrayOf)
	}
	types = append(types, tiArrayOf)

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
	isNamed := false

	// If an explicit name, Define (if necessary) the name and return the resulting info
	switch v := e.(type) {
	case *Ident:
		isNamed = true
		localName = v.Name
		pkgName = godb.GoPackageForExpr(e)
		fullName = genutils.CombineGoName(pkgName, localName)

	case *SelectorExpr:
		localName = v.Sel.Name

		pkg := v.X.(*Ident).Name
		fullPathUnix := paths.Unix(godb.FileAt(v.Pos()))
		rf := godb.GoFileForExpr(v)
		if fullPkgName, found := (*rf.Spaces)[pkg]; found {
			if godb.IsAvailable(fullPkgName) {
				isNamed = true
			} else {
				localName = fmt.Sprintf("ABEND002(reference to unavailable package `%s' looking for type `%s')", fullPkgName, localName)
			}
			pkgName = fullPkgName.String()
			fullName = pkgName + "." + localName
		} else {
			panic(fmt.Sprintf("processing %s: could not find %s in %s",
				godb.WhereAt(v.Pos()), pkg, fullPathUnix))
		}
	}

	if isNamed {
		if ti, ok := typesByFullName[fullName]; ok {
			typesByExpr[e] = ti
			return ti
		}

		// Define the type here and now, return the resulting Info.
		di := godb.LookupDeclInfo(fullName)
		if di == nil {
			panic(fmt.Sprintf("gtypes.go/Define: unregistered type %q\n", fullName))
		}
		tsNode := di.Node()
		if tsNode == nil {
			panic(fmt.Sprintf("gtypes.go/Define: nil Node for %q\n", fullName))
		}
		ts, ok := tsNode.(*TypeSpec)
		if !ok {
			panic(fmt.Sprintf("ABEND008(gtypes.go/Define: non-Typespec %T for %q (%+v)", tsNode, fullName, tsNode))
		}

		return Define(ts, nil, di.Doc())[0] // TODO: parentDoc, maybe handle other elements as well?
	}

	var innerInfo *Info
	isPassedByAddress := false
	isNullable := false
	nilPattern := "nil%.s"
	pattern := "%s"
	docPattern := pattern
	isSwitchable := true
	isExported := false

	switch v := e.(type) {
	case *StarExpr:
		innerInfo = InfoForExpr(v.X)
		pattern = fmt.Sprintf("*%s", innerInfo.Pattern)
		docPattern = fmt.Sprintf("*%s", innerInfo.DocPattern)
		localName = innerInfo.LocalName
		pkgName = innerInfo.Package
		isExported = innerInfo.IsExported
		isNullable = true
		isSwitchable = innerInfo.IsSwitchable
	case *ArrayType:
		innerInfo = InfoForExpr(v.Elt)
		len, docLen := astutils.IntExprToString(v.Len)
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
		isExported = innerInfo.IsExported
		isSwitchable = innerInfo.IsSwitchable
	case *InterfaceType:
		pkgName = ""
		if !v.Incomplete && len(v.Methods.List) == 0 {
			localName = "interface{}"
			isExported = true
		} else {
			localName = fmt.Sprintf("ABEND320(gtypes.go: %s not supported)", astutils.ExprToString(v))
		}
		isNullable = true
	case *MapType:
		pkgName = ""
		key := InfoForExpr(v.Key)
		value := InfoForExpr(v.Value)
		localName = "map[" + key.RelativeName(e.Pos()) + "]" + value.RelativeName(e.Pos())
		isExported = key.IsExported && value.IsExported
	case *ChanType:
		pattern = "chan"
		switch v.Dir {
		case SEND:
			pattern = "chan<-"
		case RECV:
			pattern = "<-chan"
		case SEND | RECV:
		default:
			pattern = fmt.Sprintf("ABEND737(gtypes.go: %s Dir=0x%x not supported) %%s", astutils.ExprToString(v), v.Dir)
		}
		innerInfo = InfoForExpr(v.Value)
		pkgName = innerInfo.Package
		localName = innerInfo.LocalName
		pattern = fmt.Sprintf("%s %s", pattern, innerInfo.Pattern)
		docPattern = pattern
		isNullable = true
		isExported = innerInfo.IsExported
		//		fmt.Printf("gtypes.go/InfoForExpr(%s) => %s\n", astutils.ExprToString(v), fmt.Sprintf(pattern, localName))
	case *StructType:
		pkgName = ""
		if v.Fields == nil || len(v.Fields.List) == 0 {
			localName = "struct{}"
			fullName = localName
			isExported = true
		} else {
			localName = fmt.Sprintf("ABEND787(gtypes.go: %s not supported)", astutils.ExprToString(v))
		}
	case *FuncType:
		pkgName = ""
		localName = fmt.Sprintf("ABEND727(gtypes.go: %s not supported)", astutils.ExprToString(v))
	case *Ellipsis:
		pkgName = ""
		localName = fmt.Sprintf("ABEND747(jtypes.go: %s not supported)", astutils.ExprToString(v))
		isSwitchable = false
	}

	if innerInfo == nil {
		if localName == "" {
			localName = fmt.Sprintf("ABEND001(gtypes.go:NO GO NAME for %T aka %q)", e, fullName)
			fullName = ""
		}
		if fullName == "" {
			fullName = computeFullName(pattern, pkgName, localName)
		}
		if ti, ok := typesByFullName[fullName]; ok {
			return ti
		}
		if strings.Contains(localName, "ABEND") {
			pkgName = ""
			isPassedByAddress = false
		}
		ti := &Info{
			Expr:              e,
			who:               fmt.Sprintf("[InfoForExpr %T]", e),
			Pattern:           pattern,
			Package:           pkgName,
			FullName:          fullName,
			LocalName:         localName,
			DocPattern:        docPattern,
			DefPos:            e.Pos(),
			UnderlyingType:    innerInfo,
			NilPattern:        nilPattern,
			IsSwitchable:      isSwitchable,
			IsPassedByAddress: isPassedByAddress,
			IsExported:        isExported,
		}
		insert(ti)
		//		fmt.Printf("gtypes.go: %s inserted %s\n", ti.who, fullName)
		return ti
	}

	if fullName == "" {
		fullName = computeFullName(pattern, pkgName, innerInfo.LocalName)
	}
	if strings.Contains(fullName, "ABEND") {
		isSwitchable = false
		isExported = false
	}
	if variant, found := typesByFullName[fullName]; found {
		return variant
	}

	if !isNullable {
		nilPattern = "%s{}"
	}
	variant := finishVariant(fullName, pattern, docPattern, nilPattern, isSwitchable, isPassedByAddress, innerInfo, e)
	if isExported {
		variant.IsExported = true
		//		fmt.Printf("gtypes.go: %s (%p) is exportable\n", variant.FullName, variant)
	}

	return variant
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

func typeAsString(f *Field, pos token.Pos) string {
	ty := f.Type
	ti := InfoForExpr(ty)
	pkgPrefix := ti.Package
	if pkgPrefix == "" || pkgPrefix == godb.GoPackageForPos(pos) {
		pkgPrefix = ""
	}
	return fmt.Sprintf(ti.Pattern, genutils.CombineGoName(pkgPrefix, ti.LocalName))
}

func typeAsStringRelative(p token.Pos) func(*Field) string {
	return func(f *Field) string { return typeAsString(f, p) }
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
