package imports

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/godb"
	. "go/ast"
	"go/token"
	"os"
	"path"
	"sort"
	"strconv"
	. "strings"
)

/* TODO: Remove all namespace support from here. Namespaces matter
   only for AutoGen'ed files anyway (they should be empty strings for
   Native files), and coupling them to planning the generation of Go
   'import' makes little sense. Note that namespaces also shouldn't be
   emitted (in :go-imports) for source packages; they are intended
   solely to map a Clojure namespace to the generated package,
   e.g. joker/std/gostd/go/std/os/; but, for now, since they must be
   emitted in those cases, they're parenthesized as commentary to
   avoid potential collisions when multiple such namespaces go into a
   single :go-imports. */

/* Represents an 'import ( foo "bar/bletch/foo" )' line to be produced. */
type Import struct {
	Local     string   // "foo", "_", or "."
	Full      string   // "bar/bletch/foo"
	Namespace string   // E.g. "go.std.io", or "" if no namespace
	who       string   // For trace
	file      *Imports // For trace
	Pos       token.Pos
	Confirmed bool
}

/* Maps relative package (unix-style) names to their imports, non-emptiness, etc. */
type Imports struct {
	Me          string             // The package in which this belongs (e.g. "github.com/candid82/joker/std/gostd/go/std/foo/bar")
	MySourcePkg string             // E.g. "foo/bar", meaning don't emit the corresponding namespace (use empty string instead)
	For         string             // For diagnostics/reporting.
	LocalNames  map[string]string  // "foo" -> "bar/bletch/foo"; no "_" nor "." entries here
	FullNames   map[string]*Import // "bar/bletch/foo" -> ["foo", "bar/bletch/foo"]
}

type Reservations struct {
	fileImports *Imports
	_for        string
	imports     map[*Import]struct{}
}

// Given the full (though relative) name of the package to be
// referenced (imported), pick a local name (starting with the base
// component of the full name) that agrees with any existing entry and
// isn't already used, trying alternate local names as necessary, add
// the mapping if necessary, and return the local name.
func (imports *Imports) AddPackage(full, ns string, okToSubstitute bool, pos token.Pos, who string) string {
	if imp := imports.addPackage(full, ns, okToSubstitute, pos, who); imp != nil {
		if Contains(full, "/time") {
			fmt.Fprintf(os.Stderr, "imports.go/(%q %q)AddPackage(full=%q ns=%q okToSubstitute=%v who=%s) at %s\n", imports.Me, imports.For, full, ns, okToSubstitute, who, godb.WhereAt(pos))
		}
		imp.Confirmed = true
		return imp.Local
	}
	return ""
}

func (imports *Imports) NewReservations(f string) *Reservations {
	return &Reservations{
		fileImports: imports,
		_for:        f,
		imports:     map[*Import]struct{}{},
	}
}

func (r *Reservations) ReservePackage(full, ns string, okToSubstitute bool, pos token.Pos, who string) string {
	if imp := r.fileImports.addPackage(full, ns, okToSubstitute, pos, "ReservePackage for "+who); imp != nil {
		if Contains(full, "/time") {
			fmt.Fprintf(os.Stderr, "imports.go/ReservePackage(%q): [%s %q] %v\n", r._for, imp.Local, imp.Full, imp.Confirmed)
		}
		r.imports[imp] = struct{}{}
		return imp.Local
	}
	return ""
}

func (r *Reservations) Reset(f string) {
	r._for = f
	r.imports = map[*Import]struct{}{}
}

// Given the full (though relative) name of the package to be
// referenced (imported), pick a local name (starting with the base
// component of the full name) that agrees with any existing entry and
// isn't already used, trying alternate local names as necessary, add
// the mapping if necessary, and return the local name.
func (imports *Imports) addPackage(full, ns string, okToSubstitute bool, pos token.Pos, who string) *Import {
	if imports == nil {
		panic(fmt.Sprintf("%q: is nil for %s at %s", imports.For, full, godb.WhereAt(pos)))
	}
	if full == "" || full == imports.Me {
		return nil
	}

	more := false
	if Contains(full, "/time") {
		more = true
		fmt.Fprintf(os.Stderr, "imports.go/(%q %q)addPackage(full=%q ns=%q okToSubstitute=%v who=%s) at %s\n", imports.Me, imports.For, full, ns, okToSubstitute, who, godb.WhereAt(pos))
	}

	local := path.Base(full)

	if e, found := imports.FullNames[full]; found {
		if e.Local == local || okToSubstitute {
			if more {
				fmt.Fprintf(os.Stderr, "imports.go/(%q)addPackage() e.Local=%q\n", imports.For, e.Local)
			}
			return e
		}
		panic(fmt.Sprintf("imports.go/(%q)addPackage([%q %q]) at %s cannot supercede [%q %q] at %s", imports.For, local, full, godb.WhereAt(pos), e.Local, e.Full, godb.WhereAt(e.Pos)))
	}

	if more {
		fmt.Fprintf(os.Stderr, "imports.go/(%q)addPackage() local=%q\n", imports.For, local)
	}

	components := Split(full, "/")
	origLocal := local
	prevComponentIndex := len(components) - 1
	for {
		curFull, found := imports.LocalNames[local]
		//		fmt.Printf("imports.go/addPackage(%p): full=%s local=%s local=%s origLocal=%s curFull='%s' found=%v\n", imports, full, local, local, origLocal, curFull, found)
		if !found {
			break
		}
		prevComponentIndex--
		if prevComponentIndex >= 0 {
			local = components[prevComponentIndex] + "_" + local
			continue
		} else if prevComponentIndex > -99 { // avoid infinite loop
			local = fmt.Sprintf("%s_%d", origLocal, -prevComponentIndex)
			continue
		}
		panic(fmt.Sprintf("imports.go/(%q)addPackage([%q %q]) cannot coexist with [%q %q]", imports.For, local, full, imports.FullNames[curFull].Local, curFull))
	}

	if imports.LocalNames == nil {
		imports.LocalNames = map[string]string{}
	}
	imports.LocalNames[local] = full

	if imports.FullNames == nil {
		imports.FullNames = map[string]*Import{}
	}
	imp := &Import{
		Local:     local,
		Full:      full,
		Namespace: ns,
		who:       who,
		file:      imports,
		Pos:       pos,
		Confirmed: false,
	}

	imports.FullNames[full] = imp
	if more {
		fmt.Fprintf(os.Stderr, "imports.go/(%q)addPackage(): full=%s ns=%s local=%s\n", imports.For, full, ns, local)
	}

	return imp
}

// Given the full (though relative) name of the package, establish it
// as the (only) interned package (using the "."  name in Go's
// 'import' statement). This is presumably Joker's 'core' package.
func (imports *Imports) InternPackage(full, ns string, pos token.Pos) {
	if imports == nil {
		panic(fmt.Sprintf("nil imports for %s at %s", full, godb.WhereAt(pos)))
	}
	if full == imports.Me {
		return
	}

	if e, found := imports.FullNames[full]; found {
		if e.Local != "." {
			panic(fmt.Sprintf("imports.go/(%q)InternPackage(%q) at %s cannot coexist with %q at %s", imports.For, full, godb.WhereAt(pos), e.Full, godb.WhereAt(e.Pos)))
		}
		return
	}

	curFull, found := imports.LocalNames["."]
	if found {
		panic(fmt.Sprintf("imports.go/(%q)InternPackage(%q) at %s cannot replace %q: %+v at %s", imports.For, full, godb.WhereAt(pos), curFull, imports.FullNames[curFull], godb.WhereAt(imports.FullNames[curFull].Pos)))
	}
	if imports.LocalNames == nil {
		imports.LocalNames = map[string]string{}
	}
	imports.LocalNames["."] = full
	if imports.FullNames == nil {
		imports.FullNames = map[string]*Import{}
	}
	imports.FullNames[full] = &Import{
		Local:     ".",
		Full:      full,
		who:       "InternPackage",
		file:      imports,
		Namespace: ns,
		Pos:       pos,
		Confirmed: true,
	}
}

func SortedOriginalPackageImports(p *Package, filter func(string) bool, f func(string, token.Pos)) {
	imports := map[string]token.Pos{}
	for _, f := range p.Files {
		for _, impSpec := range f.Imports {
			newPos := impSpec.Path.ValuePos
			newPosStr := godb.WhereAt(newPos)
			if oldPos, found := imports[impSpec.Path.Value]; found {
				if godb.WhereAt(oldPos) < newPosStr {
					continue
				}
			}
			imports[impSpec.Path.Value] = newPos
		}
	}
	var sortedImports []string
	for k, _ := range imports {
		path, err := strconv.Unquote(k)
		if err != nil {
			panic(fmt.Sprintf("error %s unquoting %s", err, k))
		}
		if filter(path) {
			sortedImports = append(sortedImports, path)
		}
	}
	sort.Strings(sortedImports)
	for _, imp := range sortedImports {
		// Note: re-quoting is a bit of a kludge, in that this
		// code depends on it reconstituting the original
		// quoted string, which it presumably always will.
		f(imp, imports[strconv.Quote(imp)])
	}
}

func (pi *Imports) sort(f func(string, *Import)) {
	var keys []string
	for k, _ := range pi.FullNames {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := pi.FullNames[k]
		f(k, v)
	}
}

func (pi *Imports) QuotedList(prefix string) string {
	imports := ""
	pi.sort(func(k string, v *Import) {
		if !v.Confirmed {
			return
		}
		if v.Local == path.Base(k) {
			imports += prefix + `"` + k + `"`
		} else {
			imports += prefix + v.Local + ` "` + k + `"`
		}
	})
	return imports
}

func (pi *Imports) AsClojureMap() string {
	imports := []string{}
	pi.sort(func(k string, v *Import) {
		if v.Confirmed {
			imports = append(imports, fmt.Sprintf(`"%s" ["%s" "%s"]`, v.Namespace, v.Local, k))
		} else {
		}
	})
	return Join(imports, ", ")
}

func (r *Reservations) Confirm() {
	for imp, _ := range r.imports {
		if Contains(imp.Full, "/time") {
			fmt.Fprintf(os.Stderr, "imports.go/Confirm(%q): %s [%s %q]\n", imp.file.For, imp.who, imp.Local, imp.Full)
		}
		imp.Confirmed = true
	}
}
