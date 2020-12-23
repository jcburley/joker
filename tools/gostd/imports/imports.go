package imports

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/godb"
	. "go/ast"
	"go/token"
	"path"
	"sort"
	"strconv"
	. "strings"
)

/* Represents an 'import ( foo "bar/bletch/foo" )' line to be produced. */
type Import struct {
	Local         string // "foo", "_", ".", or empty
	LocalRef      string // local unless empty, in which case final component of full (e.g. "foo")
	Full          string // "bar/bletch/foo"
	ClojurePrefix string // E.g. "go.std."
	PathPrefix    string // E.g. "" (for Go std) or "github.com/candid82/" (for other namespaces)
	substituted   bool   // Had to substitute a different local name
	Pos           token.Pos
}

/* Maps relative package (unix-style) names to their imports, non-emptiness, etc. */
type Imports struct {
	PackageName string             // 'package %s' reserves that name
	LocalNames  map[string]string  // "foo" -> "bar/bletch/foo"; no "_" nor "." entries here
	FullNames   map[string]*Import // "bar/bletch/foo" -> ["foo", "bar/bletch/foo"]
}

/* Given desired local and the full (though relative) name of the
/* package, make sure the local name agrees with any existing entry
/* and isn't already used (picking an alternate local name if
/* necessary), add the mapping if necessary, and return the (possibly
/* alternate) local name. */
func (imports *Imports) Add(local, full, nsPrefix, pathPrefix string, okToSubstitute bool, pos token.Pos) string {
	if imports == nil {
		panic(fmt.Sprintf("imports is nil for %s at %s", full, godb.WhereAt(pos)))
	}
	if e, found := imports.FullNames[full]; found {
		if e.Local == local {
			return e.LocalRef
		}
		if okToSubstitute {
			return e.LocalRef
		}
		panic(fmt.Sprintf("addImport(%s,%s) at %s told to to replace (%s,%s) at %s", local, full, godb.WhereAt(pos), e.Local, e.Full, godb.WhereAt(e.Pos)))
	}

	substituted := false
	components := Split(full, "/")
	localRef := local

	if local == "" {
		localRef = components[len(components)-1]
	}

	if localRef != "." {
		prevComponentIndex := len(components) - 1
		for {
			origLocalRef := localRef
			curFull, found := imports.LocalNames[localRef]
			if !found && localRef != imports.PackageName {
				break
			}
			substituted = true
			prevComponentIndex--
			if prevComponentIndex >= 0 {
				localRef = components[prevComponentIndex] + "_" + localRef
				continue
			} else if prevComponentIndex > -99 /* avoid infinite loop */ {
				localRef = fmt.Sprintf("%s_%d", origLocalRef, -prevComponentIndex)
				continue
			}
			panic(fmt.Sprintf("addImport(%s,%s) trying to replace (%s,%s)", localRef, full, imports.FullNames[curFull].LocalRef, curFull))
		}
		if imports.LocalNames == nil {
			imports.LocalNames = map[string]string{}
		}
		imports.LocalNames[localRef] = full
	}
	if imports.FullNames == nil {
		imports.FullNames = map[string]*Import{}
	}
	imports.FullNames[full] = &Import{local, localRef, full, nsPrefix, pathPrefix, substituted, pos}
	return localRef
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
		if (v.Local == "" && !v.substituted) || v.Local == path.Base(k) {
			imports += prefix + `"` + k + `"`
		} else {
			imports += prefix + v.LocalRef + ` "` + k + `"`
		}
	})
	return imports
}

func (pi *Imports) AsClojureMap() string {
	imports := []string{}
	pi.sort(func(k string, v *Import) {
		imports = append(imports, fmt.Sprintf(`"%s" ["%s" "%s"]`, v.ClojurePrefix+ReplaceAll(k, "/", "."), v.LocalRef, v.PathPrefix+k))
	})
	return Join(imports, ", ")
}

func (to *Imports) Promote(from *Imports, pos token.Pos) {
	for _, imp := range from.FullNames {
		local := imp.Local
		if local == "" {
			local = path.Base(imp.Full)
		}
		to.Add(local, imp.Full, imp.ClojurePrefix, imp.PathPrefix, false, pos)
	}
}
