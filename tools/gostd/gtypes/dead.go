// +build NO

package gtypes

var typesByClojureName = map[string]*GoType{}

func define(tdi *GoType) {
	name := tdi.ClojureName
	if existingTdi, ok := typesByClojureName[name]; ok {
		panic(fmt.Sprintf("already defined type %s at %s and again at %s", name, godb.WhereAt(existingTdi.DefPos), godb.WhereAt(tdi.DefPos)))
	}
	typesByClojureName[name] = tdi

	if tdi.Type != nil {
		tdiByExpr, found := typesByExpr[tdi.Type]
		if found && tdiByExpr != tdi {
			panic(fmt.Sprintf("different expr for type %s", name))
		}
		typesByExpr[tdi.Type] = tdi
	}
}

// clojureName = clojureTypeName(e)
// if tdi, ok := typesByClojureName[clojureName]; ok {
// 	NumClojureNameHits++
// 	typesByExpr[e] = tdi
// 	return tdi, clojureName
// }

func clojureTypeName(e Expr) (clj string) {
	switch x := e.(type) {
	case *Ident:
		break
	case *ArrayType:
		elClj := clojureTypeName(x.Elt)
		len := exprToString(x.Len)
		if len != "" {
			len = ":length " + len + " "
		}
		clj = "(vector-of " + len + elClj + ")"
		return
	case *StarExpr:
		elClj := clojureTypeName(x.X)
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
		prefix = godb.ClojureNamespaceForExpr(e) + "/"
	}
	clj = prefix + local

	o := x.Obj
	if o != nil && o.Kind == Typ {
		tdi := typesByClojureName[clj]
		if o.Name != local || (tdi != nil && o.Decl.(*TypeSpec) != tdi.TypeSpec) {
			Print(godb.Fset, x)
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
