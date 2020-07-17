package gtypes

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/godb"
	. "github.com/candid82/joker/tools/gostd/godb"
	. "github.com/candid82/joker/tools/gostd/utils"
	. "go/ast"
	"go/token"
	"go/types"
	"path"
	"sort"
	"strings"
)

const Concrete = ^uint(0) /* MaxUint */

var NumExprHits uint
var NumClojureNameHits uint

// Info from the definition of the type (if any)
type Type struct {
	Type           Expr      // The actual type (if any)
	TypeSpec       *TypeSpec // The definition of the named type (if any)
	ClojureName    string    // Clojure name (e.g. "a.b.c/Typename", "(vector-of int)")
	IsExported     bool
	Doc            string
	DefPos         token.Pos
	GoFile         *GoFile
	GoPackage      string // E.g. a/b/c (always Unix style)
	GoPattern      string // E.g. "%s", "*%s" (for reference types), "[]%s" (for array types)
	GoName         string // Base name of type (without any prefix/pattern applied)
	underlyingType *Type
	Ord            uint // Slot in []*GoTypeInfo and position of case statement in big switch in goswitch.go
	Specificity    uint // Concrete means concrete type; else # of methods defined for interface{} (abstract) type
}

var typesByClojureName = map[string]*Type{}

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

func define(tdi *Type) {
	name := tdi.ClojureName
	if existingTdi, ok := typesByClojureName[name]; ok {
		panic(fmt.Sprintf("already defined type %s at %s and again at %s", name, WhereAt(existingTdi.DefPos), WhereAt(tdi.DefPos)))
	}
	typesByClojureName[name] = tdi

	if tdi.Type != nil {
		tdiByExpr, found := typesByExpr[tdi.Type]
		if found && tdiByExpr != tdi {
			panic(fmt.Sprintf("different expr for type %s", name))
		}
		typesByExpr[tdi.Type] = tdi
	}

	//	fmt.Printf("define: %s\n", name)
}

func TypeDefine(ts *TypeSpec, gf *godb.GoFile, parentDoc *CommentGroup) []*Type {
	if len(allTypesSorted) > 0 {
		panic("Attempt to define new type after having sorted all types!!")
	}

	prefix := ClojureNamespaceForPos(Fset.Position(ts.Name.NamePos)) + "/"
	localName := ts.Name.Name
	name := prefix + localName

	doc := ts.Doc // Try block comments for this specific decl
	if doc == nil {
		doc = ts.Comment // Use line comments if no preceding block comments are available
	}
	if doc == nil {
		doc = parentDoc // Use 'var'/'const' statement block comments as last resort
	}

	types := []*Type{}

	tdi := &Type{
		Type:        ts.Type,
		TypeSpec:    ts,
		ClojureName: name,
		IsExported:  IsExported(localName),
		Doc:         CommentGroupAsString(doc),
		DefPos:      ts.Name.NamePos,
		GoFile:      gf,
		GoPattern:   "%s",
		GoPackage:   GoPackageForTypeSpec(ts),
		GoName:      localName,
		Specificity: specificity(ts),
	}
	define(tdi)
	types = append(types, tdi)

	if tdi.Specificity == Concrete {
		// Concrete types get reference-to variants, allowing Joker code to access them.
		tdiPtrTo := &Type{
			Type:           &StarExpr{X: tdi.Type},
			ClojureName:    "*" + tdi.ClojureName,
			IsExported:     tdi.IsExported,
			Doc:            "",
			GoPattern:      fmt.Sprintf(tdi.GoPattern, "*%s"),
			GoPackage:      tdi.GoPackage,
			GoName:         tdi.GoName,
			underlyingType: tdi,
			Specificity:    Concrete,
		}
		define(tdiPtrTo)
		types = append(types, tdiPtrTo)
	}

	return types
}

// Maps type-defining Expr to exactly one struct describing that type
var typesByExpr = map[Expr]*Type{}

func defineVariant(clojureName, pattern string, innerTdi *Type, te Expr) *Type {
	tdi := &Type{
		Type:           te,
		ClojureName:    clojureName,
		IsExported:     innerTdi.IsExported,
		GoPattern:      pattern,
		GoName:         innerTdi.GoName,
		underlyingType: innerTdi,
	}

	define(tdi)

	//	fmt.Printf("defineVariant: %s\n", name)

	return tdi
}

func TypeDefineBuiltin(name string) *Type {
	tdi := &Type{
		Type:        &Ident{Name: name},
		ClojureName: name,
		IsExported:  true,
		GoPattern:   "%s",
		GoName:      name,
	}

	define(tdi)

	return tdi
}

func TypeLookup(e Expr) (ty *Type, clojureName string) {
	if tdi, ok := typesByExpr[e]; ok {
		NumExprHits++
		return tdi, tdi.ClojureName
	}
	clojureName = typeName(e)
	if tdi, ok := typesByClojureName[clojureName]; ok {
		NumClojureNameHits++
		typesByExpr[e] = tdi
		return tdi, clojureName
	}

	if _, yes := e.(*Ident); yes {
		// No more information to be gleaned.
		return nil, e.(*Ident).Name
	}

	var innerTdi *Type
	pattern := "%s"
	goName := ""

	switch v := e.(type) {
	case *StarExpr:
		innerTdi, _ = TypeLookup(v.X)
		pattern = "*%s"
		goName = innerTdi.GoName
	case *ArrayType:
		innerTdi, _ = TypeLookup(v.Elt)
		len := exprToString(v.Len)
		pattern = "[" + len + "]%s"
		goName = innerTdi.GoName
	case *InterfaceType:
		goName = "interface{"
		methods := methodsToString(v.Methods.List)
		if v.Incomplete {
			methods = strings.Join([]string{methods, "..."}, ", ")
		}
		goName += methods + "}"
	case *MapType:
		key, _ := TypeLookup(v.Key)
		value, _ := TypeLookup(v.Value)
		goName = "map[" + key.RelativeGoName(e.Pos()) + "]" + value.RelativeGoName(e.Pos())
	case *SelectorExpr:
		left := fmt.Sprintf("%s", v.X)
		goName = left + "." + v.Sel.Name
	case *ChanType:
		ty, _ := TypeLookup(v.Value)
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
			goName = fmt.Sprintf("ABEND001(NO GO NAME for %s??!!)", clojureName)
		}
		tdi := &Type{
			Type:        e,
			ClojureName: clojureName,
			GoPattern:   pattern,
			GoName:      goName,
		}
		define(tdi)
		return tdi, clojureName
	}

	return defineVariant(clojureName, pattern, innerTdi, e), clojureName
}

var allTypesSorted = []*Type{}

// This establishes the order in which types are matched by 'case' statements in the "big switch" in goswitch.go. Once established,
// new types cannot be discovered/added.
func SortAll() {
	if len(allTypesSorted) > 0 {
		panic("Attempt to sort all types type after having already sorted all types!!")
	}
	for _, t := range typesByClojureName {
		if t.IsExported {
			allTypesSorted = append(allTypesSorted, t)
		}
	}
	sort.SliceStable(allTypesSorted, func(i, j int) bool {
		if allTypesSorted[i].Specificity != allTypesSorted[j].Specificity {
			return allTypesSorted[i].Specificity > allTypesSorted[j].Specificity
		}
		return allTypesSorted[i].ClojureName < allTypesSorted[j].ClojureName
	})
	for ord, t := range allTypesSorted {
		t.Ord = (uint)(ord)
		ord++
	}
}

func AllSorted() []*Type {
	return allTypesSorted
}

func typeKeyForSort(k string) string {
	if strings.HasPrefix(k, "*") {
		return k[1:] + "*"
	}
	if strings.HasPrefix(k, "[]") {
		return k[2:] + "[]"
	}
	return k
}

func SortedTypeDefinitions(m map[*Type]struct{}, f func(ti *Type)) {
	var keys []string
	for k, _ := range m {
		if k != nil {
			keys = append(keys, k.ClojureName)
		}
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return typeKeyForSort(keys[i]) < typeKeyForSort(keys[j])
	})
	for _, k := range keys {
		f(typesByClojureName[k])
	}
}

func fieldToString(f *Field) string {
	_, name := TypeLookup(f.Type)
	// Don't bother implementing this until it's actually needed:
	return "ABEND041(gtypes.go/fieldToString found something: " + name + "!)"
}

func methodsToString(methods []*Field) string {
	mStrings := make([]string, len(methods))
	for i, m := range methods {
		mStrings[i] = fieldToString(m)
	}
	return strings.Join(mStrings, ", ")
}

func typeName(e Expr) (clj string) {
	switch x := e.(type) {
	case *Ident:
		break
	case *ArrayType:
		elClj := typeName(x.Elt)
		len := exprToString(x.Len)
		if len != "" {
			len = ":length " + len + " "
		}
		clj = "(vector-of " + len + elClj + ")"
		return
	case *StarExpr:
		elClj := typeName(x.X)
		clj = "*" + elClj
		return
	case *InterfaceType:
		clj = "(interface-of "
		methods := methodsToString(x.Methods.List)
		if x.Incomplete {
			methods = strings.Join([]string{methods, "..."}, ", ")
		}
		if methods == "" {
			methods = "nil"
		}
		clj += methods + ")"
		return
	case *MapType:
		key, keyName := TypeLookup(x.Key)
		value, valueName := TypeLookup(x.Value)
		if key != nil {
			keyName = key.RelativeGoName(e.Pos())
		}
		if value != nil {
			valueName = value.RelativeGoName(e.Pos())
		}
		return "(hash-map-of " + keyName + " " + valueName + ")"
	case *SelectorExpr:
		left := fmt.Sprintf("%s", x.X)
		return left + "/" + x.Sel.Name
	case *ChanType:
		ty, tyName := TypeLookup(x.Value)
		if ty != nil {
			tyName = ty.RelativeGoName(e.Pos())
		}
		clj = "(channel-of "
		switch x.Dir & (SEND | RECV) {
		case SEND:
			clj += ":<- "
		case RECV:
			clj += ":-> "
		default:
			clj += ":<> "
		}
		clj += tyName + ")"
		return
	case *StructType:
		clj = "(struct-of ...)"
		return
	default:
		return
	}

	x := e.(*Ident)
	local := x.Name
	prefix := ""
	if types.Universe.Lookup(local) == nil {
		prefix = ClojureNamespaceForExpr(e) + "/"
	}
	clj = prefix + local

	o := x.Obj
	if o != nil && o.Kind == Typ {
		tdi := typesByClojureName[clj]
		if o.Name != local || (tdi != nil && o.Decl.(*TypeSpec) != tdi.TypeSpec) {
			Print(Fset, x)
			var ts *TypeSpec
			if tdi != nil {
				ts = tdi.TypeSpec
			}
			panic(fmt.Sprintf("mismatch name=%s != %s or ts %p != %p!", o.Name, local, o.Decl.(*TypeSpec), ts))
		}
	} else {
		// Strangely, not all *Ident's referring to defined types have x.Obj populated! Can't figure out what's
		// different about them, though maybe it's just that they're for only those receivers currently being
		// code-generated?
	}

	return
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

func (tdi *Type) TypeReflected() (packageImport, pattern string) {
	t := ""
	suffix := ".Elem()"
	if tdiu := tdi.underlyingType; tdiu != nil {
		t = "_" + path.Base(tdiu.GoPackage) + "." + fmt.Sprintf(tdi.GoPattern, tdi.GoName)
		suffix = ""
	} else {
		t = "_" + path.Base(tdi.GoPackage) + "." + fmt.Sprintf(tdi.GoPattern, tdi.GoName)
	}
	return "reflect", fmt.Sprintf("%%s.TypeOf((*%s)(nil))%s", t, suffix)
}

func (tdi *Type) TypeMappingsName() string {
	if !tdi.IsExported {
		return ""
	}
	if tdi.underlyingType != nil {
		return "info_PtrTo_" + fmt.Sprintf(tdi.underlyingType.GoPattern, tdi.underlyingType.GoName)
	}
	return "info_" + fmt.Sprintf(tdi.GoPattern, tdi.GoName)
}

func (tdi *Type) RelativeGoName(pos token.Pos) string {
	pkgPrefix := tdi.GoPackage
	if pkgPrefix == GoPackageForPos(pos) {
		pkgPrefix = ""
	} else if pkgPrefix != "" {
		pkgPrefix += "."
	}
	return fmt.Sprintf(tdi.GoPattern, pkgPrefix+tdi.GoName)
}