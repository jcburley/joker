package main

import (
	"fmt"
	. "go/ast"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

/* The transformation code, below, takes an approach that is new for me.

   Instead of each transformation point having its own transform
   routine(s), as is customary, I'm trying an approach in which the
   transform is driven by the input and multiple outputs are
   generated, where appropriate, for further processing and/or
   insertion into the ultimate transformation points.

   The primary reason for this is that the input is complicated and
   (generally) being supported to a greater extent as enhancements are
   made. I want to maintain coherence among the various transformation
   insertions, so it's less likely that a change made for one
   insertion point (to support a new input form, or modify an existing
   one) won't have corresponding changes made to other forms relying
   on the same essential input, which could lead to coding errors.

   This approach also should make it easier to see how the different
   snippets of code, relating to one particular aspect of the input,
   relate to each other, because the code will be in the same place.

   Further, it should be easier to make and track decisions (such as
   what will be the names of temporary variables) by doing it in one
   place, rather than having to make a "decision pass" first, memoize
   the results, and pass them around to the various transformation
   routines.

   However, I'm concerned that the resulting code will be too
   complicated for that to be sufficiently helpful. If I was
   proficient in a constraint/unification-based transformation
   language, I'd look at that instead, because it would allow me to
   express that e.g. "func foo(args) (returns) { ...do things with
   args...; call foo in some fashion; ...do things with returns... }"
   not only have specific transformations for each of the variables
   involved, but that they are also constrained in some fashion
   (e.g. whatever names are picked for unnamed 'returns' values are
   the same in both "returns" and "do things with returns"; whatever
   types are involved in both "args" and "returns" are properly
   processed in "do things with args" and "do things with returns",
   respectively; and so on).

   Now that I've refactored the code to achieve this, I'll start
   adding transformations and see how it goes. Might revert to
   old-fashioned use of custom transformation code per point (sharing
   code where appropriate, of course) if it gets too hairy.

*/

var nonEmptyLineRegexp *regexp.Regexp

/*
   (defn <clojureReturnType> <godecl.Name>
     <docstring>
     {:added "1.0"
      :go "<cl2golcall>"}  ; cl2golcall := <conversion?>(<cl2gol>+<clojureGoParams>)
     [clojureParamList])   ; clojureParamList

   func <goFname>(<goParamList>) <goReturnType> {  // goParamList
           <goCode>                                // goCode := <goPreCode>+"\t"+<goResultAssign>+"_"+pkg+"."+<godecl.Name>+"("+<goParams>+")\n"+<goPostCode>
   }

*/

type funcCode struct {
	clojureParamList        string
	clojureParamListDoc     string
	goParamList             string
	goParamListDoc          string
	clojureGoParams         string
	goCode                  string
	clojureReturnTypeForDoc string
	goReturnTypeForDoc      string
}

/* IMPORTANT: The public functions listed herein should be only those
   defined in joker/core/custom-runtime.go.

   That's how gostd knows to not actually generate calls to
   as-yet-unimplemented (or stubbed-out) functions, saving the
   developer the hassle of getting most of the way through a build
   before hitting undefined-func errors.
*/
var customRuntimeImplemented = map[string]struct{}{
	"ConvertToArrayOfByte":   {},
	"ConvertToArrayOfInt":    {},
	"ConvertToArrayOfString": {},
}

func genGoCall(pkgBaseName, goFname, goParams string) string {
	return "_" + pkgBaseName + "." + goFname + "(" + goParams + ")\n"
}

func genFuncCode(fn *funcInfo, pkgBaseName, pkgDirUnix string, d *FuncDecl, goFname string) (fc funcCode) {
	var goPreCode, goParams, goResultAssign, goPostCode string

	fc.clojureParamList, fc.clojureParamListDoc, fc.clojureGoParams, fc.goParamList, fc.goParamListDoc, goPreCode, goParams =
		genGoPre(fn, "\t", d.Type.Params, goFname)
	goCall := genGoCall(pkgBaseName, d.Name.Name, goParams)
	goResultAssign, fc.clojureReturnTypeForDoc, fc.goReturnTypeForDoc, goPostCode = genGoPost(fn, "\t", d)

	if goPostCode == "" && goResultAssign == "" {
		goPostCode = "\t...ABEND675: TODO...\n"
	}

	fc.goCode = goPreCode + // Optional block of pre-code
		"\t" + goResultAssign + goCall + // [results := ]fn-to-call([args...])
		goPostCode // Optional block of post-code
	return
}

// If the Go API returns a single result, and it's an Int, wrap the call in "int()". If a StarExpr is found, ABEND for now
// TODO: Return ref's for StarExpr?
func maybeConvertGoResult(pkgDirUnix, call string, fl *FieldList) string {
	if fl == nil || len(fl.List) != 1 || (fl.List[0].Names != nil && len(fl.List[0].Names) > 1) {
		return call
	}
	named := false
	t := fl.List[0].Type
	for {
		stop := false
		switch v := t.(type) {
		case *Ident:
			qt := pkgDirUnix + "." + v.Name
			if v, ok := types[qt]; ok {
				named = true
				t = v.td.Type
			} else {
				stop = true
			}
		default:
			stop = true
		}
		if stop {
			break
		}
	}
	switch v := t.(type) {
	case *Ident:
		switch v.Name {
		case "int16", "uint", "uint16", "int32", "uint32", "int64", "byte": // TODO: Does Joker always have 64-bit signed ints?
			return "int(" + call + ")"
		case "int":
			if named {
				return "int(" + call + ")"
			} // Else it's already an int, so don't bother wrapping it.
		}
	case *StarExpr:
		return fmt.Sprintf("ABEND401(StarExpr not supported -- no refs returned just yet: %s)", call)
	}
	return call
}

var abendRegexp *regexp.Regexp

var abends = map[string]int{}

func trackAbends(a string) {
	subMatches := abendRegexp.FindAllStringSubmatch(a, -1)
	//	fmt.Printf("trackAbends: %v %s => %#v\n", abendRegexp, a, subMatches)
	for _, m := range subMatches {
		if len(m) != 2 {
			panic(fmt.Sprintf("len(%v) != 2", m))
		}
		n := m[1]
		if _, ok := abends[n]; !ok {
			abends[n] = 0
		}
		abends[n] += 1
	}
}

func printAbends(m map[string]int) {
	type ac struct {
		abendCode  string
		abendCount int
	}
	a := []ac{}
	for k, v := range m {
		a = append(a, ac{abendCode: k, abendCount: v})
	}
	sort.Slice(a,
		func(i, j int) bool {
			if a[i].abendCount == a[j].abendCount {
				return a[i].abendCode < a[j].abendCode
			}
			return a[i].abendCount > a[j].abendCount
		})
	for _, v := range a {
		fmt.Printf(" %s(%d)", v.abendCode, v.abendCount)
	}
}

func genFunction(fn *funcInfo) {
	genSymReset()
	d := fn.fd
	pkgDirUnix := fn.sourceFile.pkgDirUnix
	pkgBaseName := filepath.Base(pkgDirUnix)

	jfmt := `
(defn %s%s
%s  {:added "1.0"
   :go "%s"}
  [%s])
`
	goFname := funcNameAsGoPrivate(d.Name.Name)
	fc := genFuncCode(fn, pkgBaseName, pkgDirUnix, d, goFname)
	clojureReturnType, goReturnType := clojureReturnTypeForGenerateCustom(fc.clojureReturnTypeForDoc, fc.goReturnTypeForDoc)

	var cl2gol string
	if clojureReturnType == "" {
		cl2gol = goFname
	} else {
		clojureReturnType += " "
		cl2gol = pkgBaseName + "." + d.Name.Name
		if _, found := packagesInfo[pkgDirUnix]; !found {
			panic(fmt.Sprintf("Cannot find package %s", pkgDirUnix))
		}
	}
	cl2golCall := maybeConvertGoResult(pkgDirUnix, cl2gol+fc.clojureGoParams, fn.fd.Type.Results)

	clojureFn := fmt.Sprintf(jfmt, clojureReturnType, d.Name.Name,
		commentGroupInQuotes(d.Doc, fc.clojureParamListDoc, fc.clojureReturnTypeForDoc,
			fc.goParamListDoc, fc.goReturnTypeForDoc),
		cl2golCall, fc.clojureParamList)

	gfmt := `
func %s(%s) %s {
%s}
`

	goFn := ""
	if clojureReturnType == "" { // TODO: Generate this anyway if it contains ABEND, so we can see what's needed.
		goFn = fmt.Sprintf(gfmt, goFname, fc.goParamList, goReturnType, fc.goCode)
	}

	if strings.Contains(clojureFn, "ABEND") || strings.Contains(goFn, "ABEND") {
		clojureFn = nonEmptyLineRegexp.ReplaceAllString(clojureFn, `;; $1`)
		goFn = nonEmptyLineRegexp.ReplaceAllString(goFn, `// $1`)
		trackAbends(clojureFn)
		trackAbends(goFn)
	} else {
		generatedFunctions++
		packagesInfo[pkgDirUnix].nonEmpty = true
		if clojureReturnType == "" {
			packagesInfo[pkgDirUnix].importsNative[pkgDirUnix] = exists
		} else {
			packagesInfo[pkgDirUnix].importsAutoGen[pkgDirUnix] = exists
		}
	}

	if _, ok := clojureCode[pkgDirUnix]; !ok {
		clojureCode[pkgDirUnix] = codeInfo{}
	}
	clojureCode[pkgDirUnix][d.Name.Name] = fnCodeInfo{fn.sourceFile, clojureFn}

	if _, ok := goCode[pkgDirUnix]; !ok {
		goCode[pkgDirUnix] = codeInfo{} // There'll at least be a .joke file
	}
	if goFn != "" {
		goCode[pkgDirUnix][d.Name.Name] = fnCodeInfo{fn.sourceFile, goFn}
	}
}
