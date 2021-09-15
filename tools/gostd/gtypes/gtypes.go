package gtypes

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/astutils"
	"github.com/candid82/joker/tools/gostd/genutils"
	"github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/paths"
	. "go/ast"
	"go/token"
	"go/types"
	"os"
	"strings"
)

const Concrete = ^uint(0) /* MaxUint */

var NumExprHits uint

type Info struct {
	Expr              Expr   // [key] The canonical referencing expression (if any)
	FullName          string // [key] E.g. "bool", "*net.Listener", "[]net/url.Userinfo"
	GoType            types.Type
	TypeName          string    // The types.Type.String() corresponding to this type
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
var typesByTypeName = map[string]*Info{}

var typesByType = map[types.Type]*Info{}

func setBuiltinType(fullName, nilPattern string, nullable bool) *Info {
	if info, found := typesByFullName[fullName]; found {
		return info
	}

	info := &Info{
		who:           "setBuiltinType",
		FullName:      fullName,
		GoType:        nil,
		TypeName:      fullName,
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
	typesByTypeName[fullName] = info

	return info
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
	typeName := ti.TypeName

	if fullName != typeName && !strings.Contains(fullName, "ABEND") {
		//		fmt.Fprintf(os.Stderr, "gtypes.go/insert: %s vs %s\n", fullName, typeName)
		return // Don't insert an alias type.
	}

	if gt := ti.GoType; gt != nil {
		if cti, found := typesByType[gt]; found {
			if cti.FullName != ti.FullName || cti.TypeName != ti.TypeName {
				panic(fmt.Sprintf("already have %+v yet now have %+v", cti, ti))
			}
		} else {
			typesByType[gt] = ti
		}
	} else {
		//		panic(fmt.Sprintf("nil GoType for %s (%s)", fullName, typeName))
	}

	if existingTi, ok := typesByFullName[fullName]; ok {
		fmt.Fprintf(os.Stderr, "gtypes.insert(fullName): %s already seen/defined type %s (aka %s) at %s (%p) and again (aka %s) at %s (%p)\n", ti.who, fullName, existingTi.TypeName, godb.WhereAt(existingTi.DefPos), existingTi, typeName, godb.WhereAt(ti.DefPos), ti)
		return
	}
	typesByFullName[fullName] = ti

	if _, ok := typesByTypeName[typeName]; !ok {
		typesByTypeName[typeName] = ti
	} else {
		// Alias type; no need to re-enter this.
		//		fmt.Fprintf(os.Stderr, "gtypes.insert(typeName): %s already seen/defined type %s (aka %s) at %s (%p) and again (aka %s) at %s (%p)\n", ti.who, typeName, existingTi.FullName, godb.WhereAt(existingTi.DefPos), existingTi, fullName, godb.WhereAt(ti.DefPos), ti)
	}

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

func isTypeAddressable(fullName string) bool {
	// See: https://github.com/golang/go/issues/40701
	return fullName != "reflect.StringHeader" && fullName != "reflect.SliceHeader"
}

func isTypeNewable(ti *Info) bool {
	switch ti.Type.(type) {
	case *StructType:
	case *MapType:
	// case *SelectorExpr:  // Are these all just aliases? Might try to support these someday?
	// 	return isTypeNewable(InfoForExpr(uti))
	default:
		return false
	}
	return ti.Pattern == "%s" && isTypeAddressable(ti.FullName)
}

func Define(ts *TypeSpec, gf *godb.GoFile, parentDoc *CommentGroup) []*Info {
	localName := ts.Name.Name
	pkg := godb.GoPackageForTypeSpec(ts)
	fullName := computeFullName("%s", pkg, localName)
	ti := typesByFullName[fullName]

	var typeName string
	tav, found := astutils.TypeCheckerInfo.Defs[ts.Name]
	if found {
		typeName = tav.Type().String()
	} else {
		fmt.Fprintf(os.Stderr, "cannot find type info for %s", fullName)
		typeName = fullName
	}

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

	typs := []*Info{}

	var specificity uint
	isArbitraryType := false
	if fullName == "unsafe.ArbitraryType" || fullName == "unsafe.IntegerType" {
		isArbitraryType = true
		specificity = 0
	} else {
		specificity = calculateSpecificity(ts)
	}

	underlyingInfo := InfoForExpr(ts.Type)
	isAddressable := isTypeAddressable(fullName) && underlyingInfo.IsAddressable
	isCtorable := underlyingInfo.IsBuiltin && underlyingInfo.UnderlyingType == nil

	if ti == nil {
		ti = &Info{
			Expr:              ts.Name,
			FullName:          fullName,
			GoType:            tav.(*types.TypeName).Type(),
			TypeName:          typeName,
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
			IsCtorable:        isCtorable,
		}
		insert(ti)
	}
	typs = append(typs, ti)

	if ti.Specificity == Concrete {
		// Concrete types get a reference-to variant, allowing Clojure code to access them.
		newPattern := fmt.Sprintf(ti.Pattern, "*%s")
		fullName = computeFullName(newPattern, pkg, localName)
		tiPtrTo := typesByFullName[fullName]
		if tiPtrTo == nil {
			tiPtrTo = &Info{
				Expr:              &StarExpr{X: ti.Expr},
				FullName:          fullName,
				TypeName:          "*" + typeName,
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
				IsCtorable:        !isCtorable && isTypeNewable(ti),
			}
			insert(tiPtrTo)
		}
		typs = append(typs, tiPtrTo)
	}

	newPattern := fmt.Sprintf(ti.Pattern, "[]%s")
	fullName = computeFullName(newPattern, pkg, localName)
	tiArrayOf := typesByFullName[fullName]
	if tiArrayOf == nil {
		tiArrayOf = &Info{
			Expr:              &ArrayType{Elt: ti.Expr},
			FullName:          fullName,
			TypeName:          "[]" + typeName,
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
	typs = append(typs, tiArrayOf)

	return typs
}

func InfoForName(fullName string) *Info {
	if ti, ok := typesByFullName[fullName]; ok {
		return ti
	}
	return nil
}

func InfoForType(ty types.Type) *Info {
	if ti, ok := typesByType[ty]; ok {
		return ti
	}
	return InfoForTypeByName(ty) // TODO: something smarter here, e.g. if there's an ast.Expr available?
}

func tuple(t *types.Tuple, variadic bool) (res string) {
	res = "("
	if t != nil {
		len := t.Len()
		for i := 0; i < len; i++ {
			v := t.At(i)
			if i > 0 {
				res += ", "
			}
			typ := v.Type()
			if variadic && i == len-1 {
				if s, ok := typ.(*types.Slice); ok {
					res += "..."
					typ = s.Elem()
				} else {
					res += typ.String() + "..."
					continue
				}
			}
			res += typ.String()
		}
	}
	res += ")"
	return
}

// A definition like "func foo(arg func(x T) (b U))" gets, for arg, a
// Signature of "func(x T) (b U)", instead of "func(T) U", which it
// should match. So, compute the latter string to look it up.
// TODO: Theoretically (and maybe in practice, though not in current
// Go stdlib), recursively discovered types could be Signatures; so,
// might have to reproduce the (Type)String() method here for this and
// tuple() to call.
func justTheTypesMaam(ty types.Type) (res string) {
	s, yes := ty.(*types.Signature)
	if !yes {
		return ""
	}
	res = "func" + tuple(s.Params(), s.Variadic())
	switch s.Results().Len() {
	case 0:
	case 1:
		res += " " + s.Results().At(0).Type().String()
	default:
		res += " " + tuple(s.Results(), false)
	}
	return
}

func InfoForTypeByName(ty types.Type) *Info {
	typeName := ty.String()
	if ti, ok := typesByTypeName[typeName]; ok {
		return ti
	}

	tryThis := justTheTypesMaam(ty)
	if tryThis == "" || tryThis == typeName {
		return nil
	}

	if ti, ok := typesByTypeName[tryThis]; ok {
		return ti
	}

	fmt.Fprintf(os.Stderr, "gtypes.go/InfoForTypeName: tried %s, then %s, but both failed!\n", typeName, tryThis)
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
		// if pkgName == "unsafe" {
		// 	return nil
		// }

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

		return Define(ts, nil, di.Doc())[0] // Return the base type, not the * or [] variants.
	}

	var innerInfo *Info
	pattern := "%s"
	docPattern := pattern
	nilPattern := ""
	isNullable := false
	isExported := true
	isBuiltin := false
	isSwitchable := true
	isAddressable := true
	isPassedByAddress := false
	isArbitraryType := false
	isCtorable := false

	switch v := e.(type) {
	case *StarExpr:
		innerInfo = InfoForExpr(v.X)
		if innerInfo == nil {
			return nil
		}
		pattern = fmt.Sprintf("*%s", innerInfo.Pattern)
		docPattern = fmt.Sprintf("*%s", innerInfo.DocPattern)
		localName = innerInfo.LocalName
		pkgName = innerInfo.Package
		isNullable = true
		isBuiltin = innerInfo.IsBuiltin
		isAddressable = false
		isArbitraryType = true
		isCtorable = isTypeNewable(innerInfo)
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
		isBuiltin = innerInfo.IsBuiltin
		isAddressable = false
		isArbitraryType = true
	case *InterfaceType:
		pkgName = ""
		if !v.Incomplete && len(v.Methods.List) == 0 {
			localName = "interface{}"
			isExported = true
			isBuiltin = true
		} else {
			localName = fmt.Sprintf("ABEND320(gtypes.go: %s not supported)", astutils.ExprToString(v))
		}
		isNullable = true
	case *MapType:
		pkgName = ""
		key := InfoForExpr(v.Key)
		value := InfoForExpr(v.Value)
		localName = "map[" + key.RelativeName(e.Pos()) + "]" + value.RelativeName(e.Pos())
		if key.Package == "" {
			if value.Package == "" {
			} else {
				pkgName = value.Package
				fullName = "map[" + key.RelativeName(e.Pos()) + "]" + fmt.Sprintf(value.Pattern, value.Package+"."+value.LocalName)
			}
		} else {
			if value.Package == "" {
				pkgName = key.Package
				fullName = "map[" + fmt.Sprintf(key.Pattern, key.Package+"."+key.LocalName) + "]" + value.RelativeName(e.Pos())
			} else {
				fullName = fmt.Sprintf("ABEND444(both packages %s and %s unsupported for %s)", key.Package, value.Package, localName)
			}
		}
		isExported = key.IsExported && value.IsExported
		isBuiltin = key.IsBuiltin && value.IsBuiltin
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
		isBuiltin = innerInfo.IsBuiltin
		isArbitraryType = true
		//		fmt.Printf("gtypes.go/InfoForExpr(%s) => %s\n", astutils.ExprToString(v), fmt.Sprintf(pattern, localName))
	case *StructType:
		pkgName = ""
		if v.Fields == nil || len(v.Fields.List) == 0 {
			localName = "struct{}"
			fullName = localName
			isExported = true
			isBuiltin = true
		} else {
			localName = fmt.Sprintf("ABEND787(gtypes.go: %s not supported)", astutils.ExprToString(v))
			isPassedByAddress = true
		}
	case *FuncType:
		pkgName = ""
		localName = fmt.Sprintf("ABEND727(gtypes.go: %s not supported)", astutils.ExprToString(v))
		isNullable = true
	case *Ellipsis:
		pkgName = ""
		localName = fmt.Sprintf("ABEND747(jtypes.go: %s not supported)", astutils.ExprToString(v))
		isSwitchable = false
		isAddressable = false
	}

	if innerInfo == nil {
		if localName == "" {
			localName = fmt.Sprintf("ABEND001(gtypes.go:NO GO NAME for %T aka %q)", e, fullName)
			fullName = ""
		}
		isArbitraryType = false
	} else {
		if localName == "" {
			localName = innerInfo.LocalName
		}
		isExported = isExported && innerInfo.IsExported
		isSwitchable = isSwitchable && innerInfo.IsSwitchable
		isArbitraryType = isArbitraryType && innerInfo.IsArbitraryType
	}

	if fullName == "" {
		fullName = computeFullName(pattern, pkgName, localName)
	}

	if ti, ok := typesByFullName[fullName]; ok {
		return ti
	}

	var typeName string
	tav, found := astutils.TypeCheckerInfo.Types[e]
	if found {
		typeName = tav.Type.String()
		// if strings.Contains(typeName, "uint8") {
		// 	fmt.Fprintf(os.Stderr, "gtypes.go/InfoForExpr(): found type info for %s\n", typeName)
		// }
	} else {
		typeName = fullName
		fmt.Fprintf(os.Stderr, "gtypes.go/InfoForExpr(): cannot find type info for %s\n", fullName)
	}

	isUnsupported := false

	if strings.Contains(fullName, "ABEND") {
		isExported = false
		isBuiltin = false
		isSwitchable = false
		isAddressable = false
		isUnsupported = true
		isCtorable = false
	}
	if nilPattern == "" {
		if isNullable {
			nilPattern = "nil%.s"
		} else {
			nilPattern = "%s{}"
		}
	}

	ti := &Info{
		Expr:              e,
		FullName:          fullName,
		GoType:            tav.Type,
		TypeName:          typeName,
		who:               fmt.Sprintf("[InfoForExpr %T]", e),
		UnderlyingType:    innerInfo,
		Pattern:           pattern,
		Package:           pkgName,
		LocalName:         localName,
		DocPattern:        docPattern,
		DefPos:            e.Pos(),
		File:              nil, // TODO: need this? why?
		Specificity:       Concrete,
		NilPattern:        nilPattern,
		IsNullable:        isNullable,
		IsExported:        isExported,
		IsBuiltin:         isBuiltin,
		IsSwitchable:      isSwitchable,
		IsAddressable:     isAddressable,
		IsPassedByAddress: isPassedByAddress,
		IsArbitraryType:   isArbitraryType,
		IsUnsupported:     isUnsupported,
		IsCtorable:        isCtorable,
	}
	insert(ti)

	//		fmt.Printf("gtypes.go: %s inserted %s\n", ti.who, fullName)

	return ti
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

func (ti *Info) NameDoc(e Expr) string {
	if e != nil && godb.GoPackageForExpr(e) != ti.Package {
		return fmt.Sprintf(ti.DocPattern, genutils.CombineGoName(ti.Package, ti.LocalName))
	}
	return fmt.Sprintf(ti.DocPattern, ti.LocalName)
}

func (ti *Info) NameDocForType(pkg *types.Package) string {
	res := func() string {
		if pkg != nil && pkg.Path() != ti.Package {
			return fmt.Sprintf(ti.DocPattern, genutils.CombineGoName(ti.Package, ti.LocalName))
		}
		return fmt.Sprintf(ti.DocPattern, ti.LocalName)
	}()
	if ti.LocalName == "FileHeaderXXX" {
		fmt.Fprintf(os.Stderr, "gtypes.go/NameDocForType: relative to pkg=%+v, %s.%s => %s\n", pkg, ti.Package, ti.LocalName, res)
	}
	return res
}

func init() {
	setBuiltinType("error", "nil%.s", true)
	setBuiltinType("bool", "false%.s", false)
	setBuiltinType("byte", "0%.s", false)
	setBuiltinType("rune", "0%.s", false)
	setBuiltinType("string", "\"\"%.s", false)
	setBuiltinType("int", "0%.s", false)
	setBuiltinType("int8", "0%.s", false)
	setBuiltinType("int16", "0%.s", false)
	setBuiltinType("int32", "0%.s", false)
	setBuiltinType("int64", "0%.s", false)
	setBuiltinType("uint", "0%.s", false)
	setBuiltinType("uint8", "0%.s", false)
	setBuiltinType("uint16", "0%.s", false)
	setBuiltinType("uint32", "0%.s", false)
	setBuiltinType("uint64", "0%.s", false)
	setBuiltinType("uintptr", "0%.s", false)
	setBuiltinType("float32", "0%.s", false)
	setBuiltinType("float64", "0%.s", false)
	setBuiltinType("complex128", "0%.s", false)
}
