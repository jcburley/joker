package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	Stdin          io.Reader = os.Stdin
	Stdout         io.Writer = os.Stdout
	Stderr         io.Writer = os.Stderr
	VerbosityLevel           = 0
)

type (
	Env struct {
		Namespaces    map[*string]*Namespace
		CoreNamespace *Namespace
		stdout        *Var
		stdin         *Var
		stderr        *Var
		printReadably *Var
		file          *Var
		MainFile      *Var
		args          *Var
		classPath     *Var
		ns            *Var
		NS_VAR        *Var
		IN_NS_VAR     *Var
		version       *Var
		libs          *Var
		Features      Set
	}
)

func versionMap() Map {
	res := EmptyArrayMap()
	parts := strings.Split(VERSION[1:], ".")
	i, _ := strconv.ParseInt(parts[0], 10, 64)
	res.Add(MakeKeyword("major"), Int{I: int(i)})
	i, _ = strconv.ParseInt(parts[1], 10, 64)
	res.Add(MakeKeyword("minor"), Int{I: int(i)})
	i, _ = strconv.ParseInt(parts[2], 10, 64)
	res.Add(MakeKeyword("incremental"), Int{I: int(i)})
	return res
}

func (env *Env) SetEnvArgs(newArgs []string) {
	args := EmptyArrayVector()
	for _, arg := range newArgs {
		args.Append(MakeString(arg))
	}
	if args.Count() > 0 {
		env.args.Value = args.Seq()
	} else {
		env.args.Value = NIL
	}
}

/*
This runs after invariant initialization, which includes calling

	NewEnv().  NOTE: Any changes to the list of run-time
	initializations must be reflected in gen_code/gen_code.go.
*/
func (env *Env) SetClassPath(cp string) {
	cpArray := filepath.SplitList(cp)
	cpVec := EmptyArrayVector()
	for _, cpelem := range cpArray {
		cpVec.Append(MakeString(cpelem))
	}
	if cpVec.Count() == 0 {
		cpVec.Append(MakeString(""))
	}
	env.classPath.Value = cpVec
}

/*
This runs after invariant initialization, which includes calling

	NewEnv().  NOTE: Any changes to the list of run-time
	initializations must be reflected in gen_code/gen_code.go.
*/
func (env *Env) InitEnv(stdin io.Reader, stdout, stderr io.Writer, args []string) {
	env.stdin.Value = MakeBufferedReader(stdin)
	env.stdout.Value = MakeIOWriter(stdout)
	env.stderr.Value = MakeIOWriter(stderr)
	env.SetEnvArgs(args)
}

func EnsureLoaded(name string) {
	np := MakeSymbol(name).name
	ns := GLOBAL_ENV.Namespaces[np]
	if ns == nil {
		if VerbosityLevel > 0 {
			fmt.Fprintf(Stderr, "EnsureLoaded: Cannot lazily load unknown namespace %s\n", *ns.Name.name)
		}
		return
	} else {
		ns.MaybeLazy("EnsureLoaded")
	}
}

func (env *Env) SetStdIO(stdin, stdout, stderr Object) {
	env.stdin.Value = stdin
	env.stdout.Value = stdout
	env.stderr.Value = stderr
}

func (env *Env) StdIO() (stdin, stdout, stderr Object) {
	return env.stdin.Value, env.stdout.Value, env.stderr.Value
}

/*
This runs after invariant initialization, which includes calling

	NewEnv().  NOTE: Any changes to the list of run-time
	initializations must be reflected in gen_code/gen_code.go.
*/
func (env *Env) SetMainFilename(filename string) {
	env.MainFile.Value = MakeString(filename)
}

/*
This runs after invariant initialization, which includes calling

	NewEnv().  NOTE: Any changes to the list of run-time
	initializations must be reflected in gen_code/gen_code.go.
*/
func (env *Env) SetFilename(obj Object) {
	env.file.Value = obj
}

func (env *Env) IsStdIn(obj Object) bool {
	return env.stdin.Value == obj
}

func (env *Env) CurrentNamespace() *Namespace {
	return EnsureObjectIsNamespace(env.ns.Value, "")
}

func (env *Env) SetCurrentNamespace(ns *Namespace) {
	env.ns.Value = ns
}

func (env *Env) EnsureSymbolIsNamespace(sym Symbol) *Namespace {
	if sym.ns != nil {
		panic(RT.NewError("Namespace's name cannot be qualified: " + sym.ToString(false)))
	}
	if env.Namespaces[sym.name] == nil {
		env.Namespaces[sym.name] = NewNamespace(sym)
	}
	return env.Namespaces[sym.name]
}

func (env *Env) EnsureSymbolIsLib(sym Symbol) *Namespace {
	ns := env.EnsureSymbolIsNamespace(sym)
	env.libs.Value.(*MapSet).Add(sym)
	return ns
}

func (env *Env) NamespaceFor(ns *Namespace, s Symbol) *Namespace {
	var res *Namespace
	if s.ns == nil {
		res = ns
	} else {
		res = ns.aliases[s.ns]
		if res == nil {
			res = env.Namespaces[s.ns]
		}
	}
	if res != nil {
		res.MaybeLazy("NamespaceFor")
	}
	return res
}

func (env *Env) ResolveIn(n *Namespace, s Symbol) (*Var, bool) {
	ns := env.NamespaceFor(n, s)
	if ns == nil {
		return nil, false
	}
	if v, ok := ns.mappings[s.name]; ok {
		return v, true
	}
	if s.Equals(env.IN_NS_VAR.name) {
		return env.IN_NS_VAR, true
	}
	if s.Equals(env.NS_VAR.name) {
		return env.NS_VAR, true
	}
	return nil, false
}

func (env *Env) Resolve(s Symbol) (*Var, bool) {
	return env.ResolveIn(env.CurrentNamespace(), s)
}

func (env *Env) FindNamespace(s Symbol) *Namespace {
	if s.ns != nil {
		return nil
	}
	ns := env.Namespaces[s.name]
	if ns != nil {
		ns.MaybeLazy("FindNameSpace")
	}
	return ns
}

func (env *Env) RemoveNamespace(s Symbol) *Namespace {
	if s.ns != nil {
		return nil
	}
	if s.Equals(SYMBOLS.joker_core) {
		panic(RT.NewError("Cannot remove core namespace"))
	}
	ns := env.Namespaces[s.name]
	delete(env.Namespaces, s.name)
	return ns
}

func (env *Env) ResolveSymbol(s Symbol) Symbol {
	if strings.ContainsRune(*s.name, '.') {
		return s
	}
	if s.ns == nil && TYPES[s.name] != nil {
		return s
	}
	currentNs := env.CurrentNamespace()
	if s.ns != nil {
		ns := env.NamespaceFor(currentNs, s)
		if ns == nil || ns.Name.name == s.ns {
			if ns != nil {
				ns.isUsed = true
				ns.isGloballyUsed = true
			}
			return s
		}
		ns.isUsed = true
		ns.isGloballyUsed = true
		return Symbol{
			name: s.name,
			ns:   ns.Name.name,
		}
	}
	vr, ok := currentNs.mappings[s.name]
	if !ok {
		return Symbol{
			name: s.name,
			ns:   currentNs.Name.name,
		}
	}
	vr.isUsed = true
	vr.isGloballyUsed = true
	vr.ns.isUsed = true
	vr.ns.isGloballyUsed = true
	return Symbol{
		name: vr.name.name,
		ns:   vr.ns.Name.name,
	}
}

func init() {
	GLOBAL_ENV.SetCurrentNamespace(GLOBAL_ENV.EnsureSymbolIsNamespace(MakeSymbol("user")))
}
