package types

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
var NumFullNameHits uint

// Info from the definition of the type (if any)
type Type struct {
	Type           Expr      // The actual type (if any)
	TypeSpec       *TypeSpec // The definition of the named type (if any)
	FullName       string    // Clojure name (e.g. "a.b.c/Typename")
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

var typesByFullName = map[string]*Type{}

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
	name := tdi.FullName
	if existingTdi, ok := typesByFullName[name]; ok {
		panic(fmt.Sprintf("already defined type %s at %s and again at %s", name, WhereAt(existingTdi.DefPos), WhereAt(tdi.DefPos)))
	}
	typesByFullName[name] = tdi

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
		FullName:    name,
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
			FullName:       "*" + tdi.FullName,
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

func defineVariant(pattern string, innerTdi *Type, te Expr) *Type {
	name := innerTdi.GoName
	isExported := innerTdi.IsExported

	tdi := &Type{
		Type:           te,
		FullName:       fmt.Sprintf(pattern, innerTdi.FullName),
		IsExported:     isExported,
		GoPattern:      pattern,
		GoName:         name,
		underlyingType: innerTdi,
	}

	define(tdi)

	//	fmt.Printf("defineVariant: %s\n", name)

	return tdi
}

func TypeDefineBuiltin(name string) *Type {
	tdi := &Type{
		Type:       &Ident{Name: name},
		FullName:   name,
		IsExported: true,
		GoPattern:  "%s",
		GoName:     name,
	}

	define(tdi)

	return tdi
}

func dynamicDefine(prefixes []string, e Expr) (ty *Type, fullName string) {
	switch v := e.(type) {
	case *Ident:
		break
	case *StarExpr:
		return dynamicDefine(append(prefixes, "*"), v.X)
	case *ArrayType:
		return dynamicDefine(append(prefixes, exprToString(v.Len)), v.Elt)
	default:
		return nil, strings.Join(prefixes, "")
	}
	// Try defining the type here.
	return dynamicDefine([]string{}, e)
}

func TypeLookup(e Expr) (ty *Type, fullName string) {
	if tdi, ok := typesByExpr[e]; ok {
		NumExprHits++
		return tdi, tdi.FullName
	}
	name := typeName(e)
	if tdi, ok := typesByFullName[name]; ok {
		NumFullNameHits++
		typesByExpr[e] = tdi
		return tdi, name
	}

	if _, yes := e.(*Ident); yes {
		return
	}

	var innerTdi *Type
	innerName := name
	pattern := "%s"

	switch v := e.(type) {
	case *StarExpr:
		innerTdi, innerName = TypeLookup(v.X)
		pattern = "*%s"
	case *ArrayType:
		innerTdi, innerName = TypeLookup(v.Elt)
		pattern = "[" + exprToString(v.Len) + "]%s"
	}

	newName := fmt.Sprintf(pattern, innerName)

	if innerTdi == nil {
		tdi := &Type{
			Type:      e,
			FullName:  newName,
			GoPattern: pattern,
			GoName:    newName,
		}
		define(tdi)
		return tdi, newName
	}

	return defineVariant(pattern, innerTdi, e), newName
}

var allTypesSorted = []*Type{}

// This establishes the order in which types are matched by 'case' statements in the "big switch" in goswitch.go. Once established,
// new types cannot be discovered/added.
func SortAll() {
	if len(allTypesSorted) > 0 {
		panic("Attempt to sort all types type after having already sorted all types!!")
	}
	for _, t := range typesByFullName {
		if t.IsExported {
			allTypesSorted = append(allTypesSorted, t)
		}
	}
	sort.SliceStable(allTypesSorted, func(i, j int) bool {
		if allTypesSorted[i].Specificity != allTypesSorted[j].Specificity {
			return allTypesSorted[i].Specificity > allTypesSorted[j].Specificity
		}
		return allTypesSorted[i].FullName < allTypesSorted[j].FullName
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
			keys = append(keys, k.FullName)
		}
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return typeKeyForSort(keys[i]) < typeKeyForSort(keys[j])
	})
	for _, k := range keys {
		f(typesByFullName[k])
	}
}

func fieldToString(f *Field) string {
	_, name := TypeLookup(f.Type)
	// Don't bother implementing this until it's actually needed:
	return "ABEND041(types.go/fieldToString found something: " + name + "!)"
}

func methodsToString(methods []*Field) string {
	mStrings := make([]string, len(methods))
	for i, m := range methods {
		mStrings[i] = fieldToString(m)
	}
	return strings.Join(mStrings, ", ")
}

func typeName(e Expr) (full string) {
	switch x := e.(type) {
	case *Ident:
		break
	case *ArrayType:
		elFull := typeName(x.Elt)
		len := exprToString(x.Len)
		full = "[" + len + "]" + elFull
		return
	case *StarExpr:
		elFull := typeName(x.X)
		full = "*" + elFull
		return
	case *InterfaceType:
		methods := methodsToString(x.Methods.List)
		full = "interface{" + methods
		if x.Incomplete {
			full += ", ..."
		}
		full += "}"
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
		return "map[" + keyName + "]" + valueName
	case *SelectorExpr:
		left := fmt.Sprintf("%s", x.X)
		return left + "." + x.Sel.Name
	case *ChanType:
		full = typeName(x.Value)
		switch x.Dir & (SEND | RECV) {
		case SEND:
			full = "chan<- " + full
		case RECV:
			full = "<-chan " + full
		default:
			full = "chan " + full
		}
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
	full = prefix + local

	o := x.Obj
	if o != nil && o.Kind == Typ {
		tdi := typesByFullName[full]
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
	// TODO: Support returning appropriate namespace prefix if Pos is from a different package
	return fmt.Sprintf(tdi.GoPattern, tdi.GoName)
}
