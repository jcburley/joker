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

/* Represents an 'import ( foo "bar/bletch/foo" )' line to be produced. */
type Import struct {
	Local         string // "foo", "_", or "."
	Full          string // "bar/bletch/foo"
	ClojurePrefix string // E.g. "go.std."
	Pos           token.Pos
}

/* Maps relative package (unix-style) names to their imports, non-emptiness, etc. */
type Imports struct {
	Me          string             // The package in which this belongs (e.g. "github.com/candid82/joker/std/gostd/go/std/foo/bar")
	MySourcePkg string             // E.g. "foo/bar", meaning don't emit the corresponding namespace (use empty string instead)
	For         string             // For diagnostics/reporting.
	LocalNames  map[string]string  // "foo" -> "bar/bletch/foo"; no "_" nor "." entries here
	FullNames   map[string]*Import // "bar/bletch/foo" -> ["foo", "bar/bletch/foo"]
}

// Given the full (though relative) name of the package to be
// referenced (imported), pick a local name (starting with the base
// component of the full name) that agrees with any existing entry and
// isn't already used, trying alternate local names as necessary, add
// the mapping if necessary, and return the local name.
func (imports *Imports) AddPackage(full, nsPrefix string, okToSubstitute bool, pos token.Pos) string {
	if imports == nil {
		panic(fmt.Sprintf("%q: is nil for %s at %s", imports.For, full, godb.WhereAt(pos)))
	}
	if full == imports.Me {
		return ""
	}

	more := false
	if Contains(full, "io/fs") || Contains(full, "zip") {
		more = true
		fmt.Fprintf(os.Stderr, "imports.go/(%q)AddPackage(%q, %q, %v) at %s\n", imports.For, full, nsPrefix, okToSubstitute, godb.WhereAt(pos))
	}

	local := path.Base(full)

	if e, found := imports.FullNames[full]; found {
		if e.Local == local {
			return e.Local
		}
		if okToSubstitute {
			return e.Local
		}
		panic(fmt.Sprintf("imports.go/(%q)AddPackage([%q %q]) at %s cannot supercede [%q %q] at %s", imports.For, local, full, godb.WhereAt(pos), e.Local, e.Full, godb.WhereAt(e.Pos)))
	}

	if more {
		fmt.Fprintf(os.Stderr, "imports.go/(%q)AddPackage() local=%q\n", imports.For, local)
	}

	components := Split(full, "/")
	origLocal := local
	prevComponentIndex := len(components) - 1
	for {
		curFull, found := imports.LocalNames[local]
		//		fmt.Printf("imports.go/AddPackage(%p): full=%s local=%s local=%s origLocal=%s curFull='%s' found=%v\n", imports, full, local, local, origLocal, curFull, found)
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
		panic(fmt.Sprintf("imports.go/(%q)AddPackage([%q %q]) cannot coexist with [%q %q]", imports.For, local, full, imports.FullNames[curFull].Local, curFull))
	}

	if imports.LocalNames == nil {
		imports.LocalNames = map[string]string{}
	}
	imports.LocalNames[local] = full

	if imports.FullNames == nil {
		imports.FullNames = map[string]*Import{}
	}
	imports.FullNames[full] = &Import{local, full, nsPrefix, pos}
	if more {
		fmt.Fprintf(os.Stderr, "imports.go/(%q)AddPackage(): full=%s nsPrefix=%s local=%s\n", imports.For, full, nsPrefix, local)
	}

	return local
}

// Given the full (though relative) name of the package, establish it
// as the (only) interned package (using the "."  name in Go's
// 'import' statement). This is presumably Joker's 'core' package.
func (imports *Imports) InternPackage(full string, nsPrefix string, pos token.Pos) {
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
		panic(fmt.Sprintf("imports.go/(%q)InternPackage(%q) at %s cannot replace %q: %+v", imports.For, full, godb.WhereAt(pos), curFull, imports.FullNames[curFull]))
	}
	if imports.LocalNames == nil {
		imports.LocalNames = map[string]string{}
	}
	imports.LocalNames["."] = full
	if imports.FullNames == nil {
		imports.FullNames = map[string]*Import{}
	}
	imports.FullNames[full] = &Import{".", full, nsPrefix, pos}
	// fmt.Fprintf(os.Stderr, "imports.go/InternPackage(): full=%s nsPrefix=%s\n", full, nsPrefix)
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
		imports = append(imports, fmt.Sprintf(`"%s" ["%s" "%s"]`, v.ClojurePrefix+ReplaceAll(k, "/", "."), v.Local, k))
	})
	return Join(imports, ", ")
}

func (to *Imports) Promote(from *Imports, pos token.Pos) {
	for _, imp := range from.FullNames {
		to.AddPackage(imp.Full, imp.ClojurePrefix, false, pos)
	}
}
