package imports

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/paths"
	. "go/ast"
	"go/token"
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
	PathPrefix    string // E.g. "" (for Go std) or "github.com/candid82/joker/std/gostd/go/std" (for other namespaces)
	Suffix        string // E.g. "_gostd" (for APIs generated in *_native.go files)
	Pos           token.Pos
}

/* Maps relative package (unix-style) names to their imports, non-emptiness, etc. */
type Imports struct {
	LocalNames map[string]string  // "foo" -> "bar/bletch/foo"; no "_" nor "." entries here
	FullNames  map[string]*Import // "bar/bletch/foo" -> ["foo", "bar/bletch/foo"]
}

// Given the full (though relative) name of the package, pick a local
// name (starting with the base component of the full name) that
// agrees with any existing entry and isn't already used, trying
// alternate local names as necessary, add the mapping if necessary,
// and return the local name.
func (imports *Imports) AddPackage(full, nsPrefix, pathPrefix, suffix string, okToSubstitute bool, pos token.Pos) string {
	if imports == nil {
		panic(fmt.Sprintf("imports is nil for %s at %s", full, godb.WhereAt(pos)))
	}

	local := path.Base(full) + suffix

	if e, found := imports.FullNames[full]; found {
		if e.Local == local {
			return e.Local
		}
		if okToSubstitute {
			return e.Local
		}
		panic(fmt.Sprintf("AddPackage('%s') at %s cannot supercede '%s' aka \"%s\" at %s", full, godb.WhereAt(pos), e.Full, e.Local, godb.WhereAt(e.Pos)))
	}

	// if suffix != "" {
	// 	fmt.Fprintf(os.Stderr, "imports.go/AddPackage(): full=%s nsPrefix=%s pathPrefix=%s suffix=%s local=%s\n", full, nsPrefix, pathPrefix, suffix, local)
	// }

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
		panic(fmt.Sprintf("AddPackage('%s') aka \"%s\" cannot coexist with '%s' aka \"%s\"", full, local, curFull, imports.FullNames[curFull].Local))
	}

	if imports.LocalNames == nil {
		imports.LocalNames = map[string]string{}
	}
	imports.LocalNames[local] = full

	if imports.FullNames == nil {
		imports.FullNames = map[string]*Import{}
	}
	imports.FullNames[full] = &Import{local, full, nsPrefix, pathPrefix, suffix, pos}
	// fmt.Fprintf(os.Stderr, "imports.go/AddPackage(): full=%s nsPrefix=%s pathPrefix=%s suffix=%s\n", full, nsPrefix, pathPrefix, suffix)

	return local
}

// Given the full (though relative) name of the package, establish it
// as the (only) interned package (using the "."  name in Go's
// 'import' statement).
func (imports *Imports) InternPackage(fullPath paths.UnixPath, nsPrefix, pathPrefix string, pos token.Pos) {
	full := fullPath.String()
	if imports == nil {
		panic(fmt.Sprintf("imports is nil for %s at %s", full, godb.WhereAt(pos)))
	}

	if e, found := imports.FullNames[full]; found {
		if e.Local != "." {
			panic(fmt.Sprintf("InternPackage('%s') at %s cannot coexist with '%s' at %s", full, godb.WhereAt(pos), e.Full, godb.WhereAt(e.Pos)))
		}
		return
	}

	curFull, found := imports.LocalNames["."]
	if found {
		panic(fmt.Sprintf("InternPackage('%s') at %s cannot replace '%s': %+v", full, godb.WhereAt(pos), curFull, imports.FullNames[curFull]))
	}
	if imports.LocalNames == nil {
		imports.LocalNames = map[string]string{}
	}
	imports.LocalNames["."] = full
	if imports.FullNames == nil {
		imports.FullNames = map[string]*Import{}
	}
	imports.FullNames[full] = &Import{".", full, nsPrefix, pathPrefix, "", pos}
	// fmt.Fprintf(os.Stderr, "imports.go/InternPackage(): full=%s nsPrefix=%s pathPrefix=%s\n", full, nsPrefix, pathPrefix)
}

func SortedOriginalPackageImports(p *Package, filter func(p string) bool, f func(k string, p token.Pos)) {
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

func (pi *Imports) sort(f func(k string, v *Import)) {
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
		imports = append(imports, fmt.Sprintf(`"%s" ["%s" "%s"]`, v.ClojurePrefix+ReplaceAll(k, "/", "."), v.Local, path.Join(v.PathPrefix, k)))
	})
	return Join(imports, ", ")
}

func (to *Imports) Promote(from *Imports, pos token.Pos) {
	for _, imp := range from.FullNames {
		to.AddPackage(imp.Full, imp.ClojurePrefix, imp.PathPrefix, imp.Suffix, false, pos)
	}
}
