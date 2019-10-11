package imports

import (
	"fmt"
	. "strings"
)

/* Represents an 'import ( foo "bar/bletch/foo" )' line to be produced. */
type Import struct {
	Local    string // "foo", "_", ".", or empty
	LocalRef string // local unless empty, in which case final component of full (e.g. "foo")
	Full     string // "bar/bletch/foo"
}

/* Maps relative package (unix-style) names to their imports, non-emptiness, etc. */
type Imports struct {
	LocalNames map[string]string  // "foo" -> "bar/bletch/foo"; no "_" nor "." entries here
	FullNames  map[string]*Import // "bar/bletch/foo" -> ["foo", "bar/bletch/foo"]
}

/* Given desired local and the full (though relative) name of the
/* package, make sure the local name agrees with any existing entry
/* and isn't already used (picking an alternate local name if
/* necessary), add the mapping if necessary, and return the (possibly
/* alternate) local name. */
func AddImport(imports *Imports, local, full string, okToSubstitute bool) string {
	components := Split(full, "/")
	if e, found := imports.FullNames[full]; found {
		if e.Local == local {
			return e.LocalRef
		}
		if okToSubstitute {
			return e.LocalRef
		}
		panic(fmt.Sprintf("addImport(%s,%s) told to to replace (%s,%s)", local, full, e.Local, e.Full))
	}

	localRef := local
	if local == "" {
		localRef = components[len(components)-1]
	}
	if localRef != "." {
		prevComponentIndex := len(components) - 1
		for {
			curFull, found := imports.LocalNames[localRef]
			if !found {
				break
			}
			prevComponentIndex--
			if prevComponentIndex >= 0 {
				localRef = components[prevComponentIndex] + "_" + localRef
				continue
			} else if prevComponentIndex > -99 /* avoid infinite loop */ {
				localRef = fmt.Sprintf("%s_%d", localRef, -prevComponentIndex)
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
	imports.FullNames[full] = &Import{local, localRef, full}
	return localRef
}
