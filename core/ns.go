package core

import (
	"fmt"
	"io"
	"strings"
)

type (
	Namespace struct {
		MetaHolder
		Name           Symbol
		Lazy           func()
		mappings       map[*string]*Var
		aliases        map[*string]*Namespace
		isUsed         bool
		isGloballyUsed bool
		hash           uint32
	}
)

func (ns *Namespace) ToString(escape bool) string {
	return ns.Name.ToString(escape)
}

func (ns *Namespace) TypeToString(escape bool) string {
	return ns.GetType().ToString(escape)
}

func (ns *Namespace) Print(w io.Writer, printReadably bool) {
	fmt.Fprint(w, "#object[Namespace \""+ns.Name.ToString(true)+"\"]")
}

func (ns *Namespace) Equals(other interface{}) bool {
	return ns == other
}

func (ns *Namespace) GetInfo() *ObjectInfo {
	return nil
}

func (ns *Namespace) WithInfo(info *ObjectInfo) Object {
	return ns
}

func (ns *Namespace) GetType() *Type {
	return TYPE.Namespace
}

func (ns *Namespace) WithMeta(meta Map) Object {
	res := *ns
	res.meta = SafeMerge(res.meta, meta)
	return &res
}

func (ns *Namespace) ResetMeta(newMeta Map) Map {
	ns.meta = newMeta
	return ns.meta
}

func (ns *Namespace) AlterMeta(fn *Fn, args []Object) Map {
	return AlterMeta(&ns.MetaHolder, fn, args)
}

func (ns *Namespace) Hash() uint32 {
	return ns.hash
}

func (ns *Namespace) MaybeLazy(doc string) {
	if ns.Lazy != nil {
		lazyFn := ns.Lazy
		ns.Lazy = nil
		lazyFn()
		if VerbosityLevel > 0 {
			fmt.Fprintf(Stderr, "NamespaceFor: Lazily initialized %s for %s\n", *ns.Name.name, doc)
		}
	}
}

const nsHashMask uint32 = 0x90569f6f

func NewNamespace(sym Symbol) *Namespace {
	return &Namespace{
		Name:     sym,
		mappings: make(map[*string]*Var),
		aliases:  make(map[*string]*Namespace),
		hash:     sym.Hash() ^ nsHashMask,
	}
}

func (ns *Namespace) Refer(sym Symbol, vr *Var) *Var {
	if sym.ns != nil {
		panic(RT.NewError("Can't intern namespace-qualified symbol " + sym.ToString(false)))
	}
	ns.mappings[sym.name] = vr
	return vr
}

func (ns *Namespace) ReferAll(other *Namespace) {
	for name, vr := range other.mappings {
		if !vr.isPrivate {
			ns.mappings[name] = vr
		}
	}
}

func (ns *Namespace) InternFake(sym Symbol) *Var {
	vr := ns.Intern(sym)
	vr.isFake = true
	return vr
}

func (ns *Namespace) Intern(sym Symbol) *Var {
	if sym.ns != nil {
		panic(RT.NewError("Can't intern namespace-qualified symbol " + sym.ToString(false)))
	}
	if LINTER_MODE {
		if LINTER_TYPES[sym.name] {
			msg := fmt.Sprintf("Expecting var, but %s is a type", *sym.name)
			pos := sym.GetInfo().Pos()
			printParseWarning(pos, msg)
		}
	}
	sym.meta = nil
	existingVar, ok := ns.mappings[sym.name]
	if !ok {
		newVar := &Var{
			ns:   ns,
			name: sym,
		}
		ns.mappings[sym.name] = newVar
		return newVar
	}
	if existingVar.ns != ns {
		if existingVar.ns.Name.Equals(SYMBOLS.joker_core) {
			newVar := &Var{
				ns:   ns,
				name: sym,
			}
			ns.mappings[sym.name] = newVar
			if !strings.HasPrefix(ns.Name.Name(), "joker.") {
				printParseWarning(GetPosition(sym), fmt.Sprintf("WARNING: %s already refers to: %s in namespace %s, being replaced by: %s\n",
					sym.ToString(false), existingVar.ToString(false), ns.Name.ToString(false), newVar.ToString(false)))
			}
			return newVar
		}
		panic(RT.NewErrorWithPos(fmt.Sprintf("WARNING: %s already refers to: %s in namespace %s",
			sym.ToString(false), existingVar.ToString(false), ns.ToString(false)), sym.GetInfo().Pos()))
	}
	if LINTER_MODE && existingVar.expr != nil && !existingVar.ns.Name.Equals(SYMBOLS.joker_core) {
		if !isDeclaredInConfig(existingVar) {
			if sym.GetInfo() == nil {
				printParseWarning(existingVar.GetInfo().Pos(), "Subsequent duplicate def of "+existingVar.ToString(false))
			} else {
				printParseWarning(sym.GetInfo().Pos(), "Duplicate def of "+existingVar.ToString(false))
			}
		}
	}
	return existingVar
}

func isDeclaredInConfig(vr *Var) bool {
	m := vr.GetMeta()
	if m == nil {
		return false
	}
	ok, v := m.Get(KEYWORDS.file)
	if !ok {
		return false
	}
	s, ok := v.(String)
	if !ok {
		return false
	}
	return strings.Contains(s.S, ".jokerd")
}

func (ns *Namespace) InternVar(name string, val Object, meta *ArrayMap) *Var {
	vr := ns.Intern(MakeSymbol(name))
	vr.Value = val
	meta.Add(KEYWORDS.ns, ns)
	meta.Add(KEYWORDS.name, vr.name)
	vr.meta = meta
	return vr
}

func (ns *Namespace) InternVarAutoDeref(name string, val Object, meta *ArrayMap) *Var {
	vr := ns.InternVar(name, val, meta)
	vr.isAutoDeref = true
	return vr
}

func (ns *Namespace) AddAlias(alias Symbol, namespace *Namespace) {
	if alias.ns != nil {
		panic(RT.NewError("Alias can't be namespace-qualified"))
	}
	existing := ns.aliases[alias.name]
	if existing != nil && existing != namespace {
		msg := "Alias " + alias.ToString(false) + " already exists in namespace " + ns.Name.ToString(false) + ", aliasing " + existing.Name.ToString(false)
		if LINTER_MODE {
			printParseError(GetPosition(alias), msg)
			return
		}
		panic(RT.NewError(msg))
	}
	ns.aliases[alias.name] = namespace
}

func (ns *Namespace) Resolve(name string) *Var {
	return ns.mappings[STRINGS.Intern(name)]
}

func (ns *Namespace) Mappings() map[*string]*Var {
	return ns.mappings
}

func (ns *Namespace) Aliases() map[*string]*Namespace {
	return ns.aliases
}
