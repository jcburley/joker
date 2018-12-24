package main

import (
	"bufio"
	"fmt"
	. "go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const VERSION = "0.1"

func check(e error) {
	if e != nil {
		panic(e)
	}
}

/* Want to support e.g.:

     net/dial.go:DialTimeout(network, address string, timeout time.Duration) => _ Conn, _ error

   I.e. a function defined in one package refers to a type defined in
   another (a different directory, even).

   Sample routines include (from 'net' package):
     - lookupMX
     - queryEscape
   E.g.:
     ./gostd --dir $PWD/tests/small --output-code 2>&1 | grep -C20 lookupMX

*/

var fset *token.FileSet
var dump bool
var verbose bool
var methods int
var generatedFunctions int

func whereAt(p token.Pos) string {
	return fmt.Sprintf("%s", fset.Position(p).String())
}

func unix(p string) string {
	return filepath.ToSlash(p)
}

func commentGroupInQuotes(doc *CommentGroup, jokIn, jokOut, goIn, goOut string) string {
	var d string
	if doc != nil {
		d = doc.Text()
	}
	if goIn != "" {
		if d != "" {
			d = strings.Trim(d, " \t\n") + "\n\n"
		}
		d += "Go input arguments: " + goIn
	}
	if goOut != "" {
		if d != "" {
			d = strings.Trim(d, " \t\n") + "\n\n"
		}
		d += "Go return type: " + goOut
	}
	if jokIn != "" {
		if d != "" {
			d = strings.Trim(d, " \t\n") + "\n\n"
		}
		d += "Joker input arguments: " + jokIn
	}
	if jokOut != "" {
		if d != "" {
			d = strings.Trim(d, " \t\n") + "\n\n"
		}
		d += "Joker return type: " + jokOut
	}
	return `  ` + strings.Trim(strconv.Quote(d), " \t\n") + "\n"
}

type funcInfo struct {
	fd         *FuncDecl
	pkg        string // base package name
	pkgDirUnix string // relative (Unix-style) path to package
	filename   string // relative (Unix-style) filename within package
}

/* Go apparently doesn't support/allow 'interface{}' as the value (or
/* key) of a map such that any arbitrary type can be substituted at
/* run time, so there are several of these nearly-identical functions
/* sprinkled through this code. Still get some reuse out of some of
/* them, and it's still easier to maintain these copies than if the
/* body of these were to be included at each call point.... */
func sortedFuncInfoMap(m map[string]*funcInfo, f func(k string, v *funcInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

// Map qualified function names to info on each.
var qualifiedFunctions = map[string]*funcInfo{}

var alreadySeen = []string{}

// Returns whether any public functions were actually processed.
func processFuncDecl(pkg, pkgDirUnix, filename string, f *File, fn *FuncDecl) bool {
	if dump {
		Print(fset, fn)
	}
	fname := pkgDirUnix + "." + fn.Name.Name
	if v, ok := qualifiedFunctions[fname]; ok {
		alreadySeen = append(alreadySeen,
			fmt.Sprintf("NOTE: Already seen function %s in %s, yet again in %s",
				fname, v.filename, filename))
	}
	qualifiedFunctions[fname] = &funcInfo{fn, pkg, pkgDirUnix, filename}
	return true
}

type typeInfo struct {
	td       *TypeSpec
	file     string // Relative (Unix-style) path to defining file
	building bool
}

func sortedTypeInfoMap(m map[string]*typeInfo, f func(k string, v *typeInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

var types = map[string]*typeInfo{}

func processTypeSpec(pkg string, filename string, f *File, ts *TypeSpec) {
	if dump {
		Print(fset, ts)
	}
	typename := pkg + "." + ts.Name.Name
	if c, ok := types[typename]; ok {
		if c.file == filename {
			panic(fmt.Sprintf("type %s defined twice in file %s", typename, filename))
		}
	}
	types[typename] = &typeInfo{ts, filename, false}
}

func processTypeSpecs(pkg string, filename string, f *File, tss []Spec) {
	for _, spec := range tss {
		ts := spec.(*TypeSpec)
		if isPrivate(ts.Name.Name) {
			continue // Skipping non-exported functions
		}
		processTypeSpec(pkg, filename, f, ts)
	}
}

// Returns whether any public functions were actually processed.
func processDecls(pkg, pkgDirUnix, filename string, f *File) (found bool) {
	for _, s := range f.Decls {
		switch v := s.(type) {
		case *FuncDecl:
			rcv := v.Recv // *FieldList of methods or nil (functions)
			if rcv != nil {
				methods += 1
				continue // Skipping these for now
			}
			if isPrivate(v.Name.Name) {
				continue // Skipping non-exported functions
			}
			if processFuncDecl(pkg, pkgDirUnix, filename, f, v) {
				found = true
			}
		case *GenDecl:
			if v.Tok != token.TYPE {
				continue
			}
			processTypeSpecs(pkgDirUnix, filename, f, v.Specs)
		default:
			panic(fmt.Sprintf("unrecognized Decl type %T at: %s", v, whereAt(v.Pos())))
		}
	}
	return
}

var exists = struct{}{}

/* Maps relative package (unix-style) names to their imports, non-emptiness, etc. */
type packageImports map[string]struct{}
type packageInfo struct {
	importsNative  packageImports
	importsAutoGen packageImports
	nonEmpty       bool // Whether any non-comment code has been generated
	hasGoFiles     bool // Whether any .go files (would) have been generated
}

var packagesInfo = map[string]*packageInfo{}

/* Sort the packages -- currently appears to not actually be
/* necessary, probably because of how walkDirs() works. */
func sortedPackagesInfo(m map[string]*packageInfo, f func(k string, i *packageInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

func sortedPackageImports(pi packageImports, f func(k string)) {
	var keys []string
	for k, _ := range pi {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k)
	}
}

func processPackage(pkgDir, pkgDirUnix, pkg string, p *Package) {
	if verbose {
		fmt.Printf("Processing package=%s in %s:\n", pkg, pkgDirUnix)
	}
	found := false
	for filename, f := range p.Files {
		if processDecls(pkg, pkgDirUnix, filepath.ToSlash(filename), f) {
			found = true
		}
	}
	if found {
		if _, ok := packagesInfo[pkgDirUnix]; !ok {
			packagesInfo[pkgDirUnix] = &packageInfo{packageImports{}, packageImports{}, false, false}
		}
	}
}

func processDir(d string, path string, mode parser.Mode) error {
	pkgDir := strings.TrimPrefix(path, d+string(filepath.Separator))
	pkgDirUnix := filepath.ToSlash(pkgDir)
	if verbose {
		fmt.Printf("Processing %s:\n", pkgDirUnix)
	}

	pkgs, err := parser.ParseDir(fset, path,
		// Walk only *.go files that meet default (target) build constraints, e.g. per "// build ..."
		func(info os.FileInfo) bool {
			if strings.HasSuffix(info.Name(), "_test.go") {
				if verbose {
					fmt.Printf("Ignoring test code in %s\n", info.Name())
				}
				return false
			}
			b, e := build.Default.MatchFile(path, info.Name())
			if verbose {
				fmt.Printf("Matchfile(%s) => %v %v\n",
					filepath.ToSlash(filepath.Join(path, info.Name())),
					b, e)
			}
			return b && e == nil
		},
		mode)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	basename := filepath.Base(path)
	for k, v := range pkgs {
		if k != basename && k != basename+"_test" {
			if verbose {
				fmt.Printf("NOTICE: Package %s is defined in %s -- ignored\n", k, path)
			}
		} else {
			if verbose {
				fmt.Printf("Package %s:\n", k)
			}
			processPackage(pkgDir, pkgDirUnix, k, v) // processPackage(strings.Replace(path, d + "/", "", 1) + "/" + k, v)
		}
	}

	return nil
}

var excludeDirs = map[string]bool{
	"builtin":  true,
	"cmd":      true,
	"internal": true, // look into this later?
	"testdata": true,
	"vendor":   true,
}

func walkDirs(d string, mode parser.Mode) error {
	target, err := filepath.EvalSymlinks(d)
	check(err)
	err = filepath.Walk(target,
		func(path string, info os.FileInfo, err error) error {
			rel := strings.Replace(path, target, d, 1)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Skipping %s due to: %v\n",
					filepath.ToSlash(rel), err)
				return err
			}
			if rel == d {
				return nil // skip (implicit) "."
			}
			if excludeDirs[filepath.Base(rel)] {
				if verbose {
					fmt.Printf("Excluding %s\n",
						filepath.ToSlash(rel))
				}
				return filepath.SkipDir
			}
			if info.IsDir() {
				if verbose {
					fmt.Printf("Walking from %s to %s\n",
						filepath.ToSlash(d), filepath.ToSlash(rel))
				}
				return processDir(d, rel, mode)
			}
			return nil // not a directory
		})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while walking %s: %v\n", d, err)
		return err
	}

	return err
}

func paramNameAsClojure(n string) string {
	return n
}

func funcNameAsGoPrivate(f string) string {
	return strings.ToLower(f[0:1]) + f[1:]
}

func paramNameAsGo(p string) string {
	return p
}

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

var genSymIndex = map[string]int{}

func genSym(pre string) string {
	var idx int
	if i, ok := genSymIndex[pre]; ok {
		idx = i + 1
	} else {
		idx = 1
	}
	genSymIndex[pre] = idx
	return fmt.Sprintf("%s%d", pre, idx)
}

func genSymReset() {
	genSymIndex = map[string]int{}
}

func exprIsUseful(rtn string) bool {
	return rtn != "NIL"
}

func genGoPostNamed(indent, pkg, in, t, onlyIf string) (jok, gol, goc, out string) {
	qt := pkg + "." + t
	if v, ok := types[qt]; ok {
		if v.building { // Mutually-referring types currently not supported
			jok = fmt.Sprintf("ABEND947(recursive type reference involving %s)",
				qt) // TODO: handle these, e.g. http Request/Response
			gol = jok
			goc = ""
		} else {
			v.building = true
			jok, gol, goc, out = genGoPostExpr(indent, pkg, in, v.td.Type, onlyIf)
			v.building = false
		}
	} else {
		jok = fmt.Sprintf("ABEND042(cannot find typename %s)", qt)
	}
	return
}

func isPrivate(p string) bool {
	return !unicode.IsUpper(rune(p[0]))
}

// func tryThis(s string) struct { a int; b string } {
//	return struct { a int; b string }{ 5, "hey" }
// }

// Joker: { :a ^Int, :b ^String }
// Go: struct { a int; b string }
func genGoPostStruct(indent, pkg, in string, fl *FieldList, onlyIf string) (jok, gol, goc, out string) {
	tmpmap := "_map" + genSym("")
	useful := false
	for _, f := range fl.List {
		for _, p := range f.Names {
			if isPrivate(p.Name) {
				continue // Skipping non-exported fields
			}
			var joktype, goltype, more_goc string
			joktype, goltype, more_goc, out =
				genGoPostExpr(indent, pkg, in+"."+p.Name, f.Type, "")
			if useful || exprIsUseful(out) {
				useful = true
			}
			goc += more_goc
			goc += indent + tmpmap +
				".Add(MakeKeyword(\"" + p.Name + "\"), " + out + ")\n"
			if jok != "" {
				jok += ", "
			}
			if gol != "" {
				gol += "; "
			}
			if p == nil {
				jok += "_ "
			} else {
				jok += ":" + p.Name + " "
				gol += p.Name + " "
			}
			if joktype != "" {
				jok += "^" + joktype
			}
			if goltype != "" {
				gol += goltype
			}
		}
	}
	jok = "{" + jok + "}"
	gol = "struct {" + gol + "}"
	if useful {
		goc = wrapStmtOnlyIfs(indent, tmpmap, "ArrayMap", "EmptyArrayMap()", onlyIf, goc, &out)
	} else {
		goc = ""
		out = "NIL"
	}
	return
}

func genGoPostArray(indent, pkg, in string, el Expr, onlyIf string) (jok, gol, goc, out string) {
	tmp := genSym("")
	tmpvec := "_vec" + tmp
	tmpelem := "_elem" + tmp

	var goc_pre string
	jok, gol, goc_pre, out = genGoPostExpr(indent+"\t", pkg, tmpelem, el, "")
	useful := exprIsUseful(out)
	jok = "(vector-of " + jok + ")"
	gol = "[]" + gol

	if useful {
		goc = indent + "for _, " + tmpelem + " := range " + in + " {\n"
		goc += goc_pre
		goc += indent + "\t" + tmpvec + " = " + tmpvec + ".Conjoin(" + out + ")\n"
		goc += indent + "}\n"
		goc = wrapStmtOnlyIfs(indent, tmpvec, "Vector", "EmptyVector", onlyIf, goc, &out)
	} else {
		goc = ""
	}
	return
}

// TODO: Maybe return a ref or something Joker (someday) supports? flag.String() is useful only as it returns a ref;
// whereas net.LookupMX() returns []*MX, and these are not only populated, it's unclear there's any utility in
// modifying them (it could just as well return []MX AFAICT).
func genGoPostStar(indent, pkg, in string, e Expr, onlyIf string) (jok, gol, goc, out string) {
	if onlyIf == "" {
		onlyIf = in + " != nil"
	} else {
		onlyIf = in + " != nil && " + onlyIf
	}
	jok, gol, goc, out = genGoPostExpr(indent, pkg, "(*"+in+")", e, onlyIf)
	gol = "*" + gol
	return
}

func maybeNil(expr, in string) string {
	return "func () Object { if (" + expr + ") == nil { return NIL } else { return " + in + " } }()"
}

func genGoPostExpr(indent, pkg, in string, e Expr, onlyIf string) (jok, gol, goc, out string) {
	switch v := e.(type) {
	case *Ident:
		switch v.Name {
		case "string":
			jok = "String"
			gol = "string"
			out = "MakeString(" + in + ")"
		case "int", "int16", "uint", "uint16", "int32", "uint32", "int64", "byte": // TODO: Does Joker always have 64-bit signed ints?
			jok = "Int"
			gol = "int"
			out = "MakeInt(int(" + in + "))"
		case "bool":
			jok = "Bool"
			gol = "bool"
			out = "MakeBool(" + in + ")"
		case "error":
			jok = "Error"
			gol = "error"
			out = maybeNil(in, "MakeError("+in+")") // TODO: Test this against the MakeError() added to joker/core/object.go
		default:
			jok, _, goc, out = genGoPostNamed(indent, pkg, in, v.Name, onlyIf)
			gol = v.Name // This is as far as Go needs to go for a type signature
		}
	case *ArrayType:
		jok, gol, goc, out = genGoPostArray(indent, pkg, in, v.Elt, onlyIf)
	case *StarExpr:
		jok, gol, goc, out = genGoPostStar(indent, pkg, in, v.X, onlyIf)
	case *StructType:
		jok, gol, goc, out = genGoPostStruct(indent, pkg, in, v.Fields, onlyIf)
	default:
		jok = fmt.Sprintf("ABEND883(unrecognized Expr type %T at: %s)", e, unix(whereAt(e.Pos())))
		gol = "..."
		out = in
	}
	return
}

const resultName = "_res"

func genGoPostItem(indent, pkg, in string, f *Field, onlyIf string) (captureVar, jok, gol, goc, out string, useful bool) {
	captureVar = in
	if in == "" {
		captureVar = genSym(resultName)
	}
	jok, gol, goc, out = genGoPostExpr(indent, pkg, captureVar, f.Type, onlyIf)
	if in != "" && in != resultName {
		gol = paramNameAsGo(in) + " " + gol
	}
	useful = exprIsUseful(out)
	if !useful {
		captureVar = "_"
	}
	return
}

func reverseJoin(a []string, infix string) string {
	j := ""
	for idx := len(a) - 1; idx >= 0; idx-- {
		if idx != len(a)-1 {
			j += infix
		}
		j += a[idx]
	}
	return j
}

// Generates code that, at run time, tests each of the onlyIf's and, if all true, returns the expr; else returns NIL.
func wrapOnlyIfs(onlyIf string, e string) string {
	if len(onlyIf) == 0 {
		return e
	}
	return "func() Object { if " + onlyIf + " { return " + e + " } else { return NIL } }()"
}

// Add one level of indent to each line
func indentedCode(c string) string {
	return "\t" + strings.Replace(c, "\n", "\n\t", -1)
}

func wrapStmtOnlyIfs(indent, v, t, e string, onlyIf string, c string, out *string) string {
	if len(onlyIf) == 0 {
		*out = v
		return indent + v + " := " + e + "\n" + c
	}
	*out = "_obj" + v
	return indent + "var " + *out + " Object\n" +
		indent + "if " + onlyIf + " {\n" +
		indent + "\t" + v + " := " + e + "\n" +
		strings.TrimRight(indentedCode(c), "\t") +
		indent + "\t" + *out + " = Object(" + v + ")\n" +
		indent + "} else {\n" +
		indent + "\t" + *out + " = NIL\n" +
		indent + "}\n"
}

// Caller generates "outGOCALL;goc" while saving jok and gol for type info (they go into .joke as metadata and docstrings)
func genGoPostList(indent string, pkg string, fl FieldList) (jok, gol, goc, out string) {
	useful := false
	captureVars := []string{}
	jokType := []string{}
	golType := []string{}
	goCode := []string{}

	result := resultName
	multipleCaptures := len(fl.List) > 1 || (fl.List[0].Names != nil && len(fl.List[0].Names) > 1)
	for _, f := range fl.List {
		names := []string{}
		if f.Names == nil {
			names = append(names, "")
		} else {
			for _, n := range f.Names {
				names = append(names, n.Name)
			}
		}
		for _, n := range names {
			captureName := result
			if multipleCaptures {
				captureName = n
			}
			captureVar, jokNew, golNew, gocNew, outNew, usefulItem := genGoPostItem(indent, pkg, captureName, f, "")
			useful = useful || usefulItem
			if multipleCaptures {
				gocNew += indent + result + " = " + result + ".Conjoin(" + outNew + ")\n"
			} else {
				result = outNew
			}
			captureVars = append(captureVars, captureVar)
			jokType = append(jokType, jokNew)
			golType = append(golType, golNew)
			goCode = append(goCode, gocNew)
		}
	}

	out = strings.Join(captureVars, ", ")
	if out != "" {
		out += " := "
	}

	jok = strings.Join(jokType, " ")
	if len(jokType) > 1 && jok != "" {
		jok = "[" + jok + "]"
	}

	gol = strings.Join(golType, ", ")
	if len(golType) > 1 && gol != "" {
		gol = "(" + gol + ")"
	}

	goc = strings.Join(goCode, "")

	if multipleCaptures {
		if useful {
			goc = indent + result + " := EmptyVector\n" + goc + indent + "return " + result + "\n"
		} else {
			goc = indent + "ABEND123(no public information returned)\n"
		}
	} else {
		if goc == "" {
			out = "return " // No code generated, so no need to use intermediary
		} else {
			goc += indent + "return " + result + "\n"
		}
		if !useful {
			goc += indent + "ABEND124(no public information returned)\n"
		}
	}

	return
}

// Return a form of the return type as supported by generate-std.joke,
// or empty string if not supported (which will trigger attempting to
// generate appropriate code for *_native.go). gol either passes
// through or "Object" is returned for it if jok is returned as empty.
func jokerReturnTypeForGenerateCustom(in_jok, in_gol string) (jok, gol string) {
	switch in_jok {
	case "String", "Int", "Byte", "Double", "Bool", "Time", "Error": // TODO: Have tested only String so far
		jok = `^"` + in_jok + `"`
	default:
		jok = ""
		gol = "Object"
	}
	return
}

type codeInfo map[string]string

/* Map relative (Unix-style) package names to maps of filenames to code strings. */
var jokerCode = map[string]codeInfo{}
var goCode = map[string]codeInfo{}

func sortedPackageMap(m map[string]codeInfo, f func(k string, v codeInfo)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

func sortedCodeMap(m codeInfo, f func(k string, v string)) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m[k])
	}
}

var nonEmptyLineRegexp *regexp.Regexp

/*
   (defn <jokerReturnType> <godecl.Name>
     <docstring>
     {:added "1.0"
      :go "<jok2golcall>"}                ; jok2golcall := <conversion?>(<jok2gol>+<jokerGoParams>)
     [jokerParamList])                    ; jokerParamList

   func <goFname>(<goParamList>) <goReturnType> {  // goParamList
           <goCode>                                // goCode := <goPreCode>+"\t"+<goResultAssign>+"_"+pkg+"."+<godecl.Name>+"("+<goParams>+")\n"+<goPostCode>
   }

*/

type funcCode struct {
	jokerParamList        string
	jokerParamListDoc     string
	goParamList           string
	goParamListDoc        string
	jokerGoParams         string
	goCode                string
	jokerReturnTypeForDoc string
	goReturnTypeForDoc    string
}

/* IMPORTANT: The public functions listed herein should be only those
   defined in in joker/core/custom-runtime.go.

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

func genGoPreArray(indent string, e *ArrayType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	el := e.Elt
	len := e.Len
	clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genTypePre(indent, el, paramName)
	runtime := "ConvertToArrayOf" + clType
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if len != nil {
		jok2golParam = "ABEND901(specific-length arrays not supported: " + jok2golParam + ")"
	} else if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND902(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	} else if _, ok := el.(*Ident); !ok {
		jok2golParam = "ABEND910(arrays of things other than identifiers not supported: " + jok2golParam + ")"
	}
	clType = "Object"
	clTypeDoc = "(vector-of " + clTypeDoc + ")"
	goType = "[]" + goType
	goTypeDoc = goType
	return
}

func genGoPreStar(indent string, e *StarExpr, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	el := e.X
	clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genTypePre(indent, el, paramName)
	runtime := "ConvertToIndirectOf" + clType
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND903(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	}
	clType = "Object"
	clTypeDoc = "(atom-of " + clTypeDoc + ")"
	goType = "*" + goType
	goTypeDoc = goType
	return
}

func genGoPreSelector(indent string, e *SelectorExpr, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	X := e.X // a package identifier
	Sel := e.Sel
	clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genTypePre(indent, Sel, paramName)
	runtime := X.(*Ident).Name + "." + Sel.Name // wrong, but documents what is needed here
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND904(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	}
	goType = "*" + goType
	goTypeDoc = goType
	return
}

func genGoPreEllipsis(indent string, e *Ellipsis, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	el := e.Elt
	clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genTypePre(indent, el, paramName)
	runtime := "ConvertToEllipsisHaHa" + clType
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND905(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	}
	clTypeDoc = "(ellipsis-somehow " + clType + ")"
	goType = "..." + goType
	goTypeDoc = goType
	return
}

func genGoPreFunc(indent string, e *FuncType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	clType = "fn"
	goType = "func"
	runtime := "ConvertToFuncTypeHaHa"
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND906(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	}
	clTypeDoc = clType
	goTypeDoc = goType
	return
}

func genGoPreInterface(indent string, e *InterfaceType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	clType = "<protocol-or-something>"
	goType = "interface {}"
	runtime := "ConvertToInterfaceTypeHaHa"
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND907(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	}
	clTypeDoc = clType
	goTypeDoc = goType
	return
}

func genGoPreMap(indent string, e *MapType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	clType = "{}"
	goType = "map[]"
	runtime := "ConvertToMapTypeHaHa"
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND908(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	}
	clTypeDoc = clType
	goTypeDoc = goType
	return
}

func genGoPreChan(indent string, e *ChanType, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	clType = "<no-idea-about-chan-yet>"
	goType = "<-chan"
	runtime := "ConvertToChanTypeHaHa"
	jok2golParam = runtime + "(" + jok2golParam + ")"
	if _, ok := customRuntimeImplemented[runtime]; !ok {
		if !strings.Contains(jok2golParam, "ABEND") {
			jok2golParam = "ABEND909(custom-runtime routine not implemented: " + jok2golParam + ")"
		}
	}
	clTypeDoc = clType
	goTypeDoc = goType
	return
}

func genTypePre(indent string, e Expr, paramName string) (clType, clTypeDoc, goType, goTypeDoc, jok2golParam string) {
	clType = fmt.Sprintf("ABEND881(unrecognized Expr type %T at: %s)", e, whereAt(e.Pos()))
	goType = fmt.Sprintf("ABEND882(unrecognized Expr type %T at: %s)", e, whereAt(e.Pos()))
	jok2golParam = paramName
	switch v := e.(type) {
	case *Ident:
		goType = v.Name
		clType = fmt.Sprintf("ABEND885(unrecognized type %s at: %s)", v.Name, whereAt(e.Pos()))
		switch v.Name {
		case "string":
			clType = "String"
		case "int":
			clType = "Int"
		case "byte":
			clType = "Byte"
		case "bool":
			clType = "Bool"
		case "int16":
			clType = "Int16"
		case "uint":
			clType = "UInt"
		case "uint16":
			clType = "UInt16"
		case "int32":
			clType = "Int32"
		case "uint32":
			clType = "UInt32"
		case "int64":
			clType = "Int64"
		case "error":
		default:
			clType = v.Name                                                                         // The important thing here is that we don't have a Go conversion
			goType = fmt.Sprintf("ABEND884(unrecognized type %s at: %s)", v.Name, whereAt(e.Pos())) // only user-defined types left now
		}
		clTypeDoc = clType
		goTypeDoc = goType
	case *ArrayType:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreArray(indent, v, paramName)
	case *StarExpr:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreStar(indent, v, paramName)
	case *SelectorExpr:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreSelector(indent, v, paramName)
	case *Ellipsis:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreEllipsis(indent, v, paramName)
	case *FuncType:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreFunc(indent, v, paramName)
	case *InterfaceType:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreInterface(indent, v, paramName)
	case *MapType:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreMap(indent, v, paramName)
	case *ChanType:
		clType, clTypeDoc, goType, goTypeDoc, jok2golParam = genGoPreChan(indent, v, paramName)
	}
	return
}

func genGoPre(indent string, fl *FieldList, goFname string) (jokerParamList, jokerParamListDoc,
	jokerGoParams, goParamList, goParamListDoc, goPreCode, goParams string) {
	if fl == nil {
		return
	}
	for _, f := range fl.List {
		for _, p := range f.Names {
			clType, clTypeDoc, goType, goTypeDoc, jok2golParam := genTypePre(indent, f.Type, "_"+p.Name)

			if jokerParamList != "" {
				jokerParamList += ", "
			}
			if clType != "" {
				jokerParamList += "^" + clType + " "
			}
			jokerParamList += "_" + paramNameAsClojure(p.Name)

			if jokerParamListDoc != "" {
				jokerParamListDoc += ", "
			}
			if clTypeDoc != "" {
				jokerParamListDoc += "^" + clTypeDoc + " "
			}
			jokerParamListDoc += paramNameAsClojure(p.Name)

			if jokerGoParams != "" {
				jokerGoParams += ", "
			}
			jokerGoParams += jok2golParam

			if goParamList != "" {
				goParamList += ", "
			}
			goParamList += paramNameAsGo(p.Name)
			if goType != "" {
				goParamList += " " + goType
			}

			if goParamListDoc != "" {
				goParamListDoc += ", "
			}
			goParamListDoc += paramNameAsGo(p.Name)
			if goTypeDoc != "" {
				goParamListDoc += " " + goTypeDoc
			}

			if goParams != "" {
				goParams += ", "
			}
			goParams += paramNameAsGo(p.Name)
		}
	}
	jokerGoParams = "(" + jokerGoParams + ")"
	jokerParamListDoc = "[" + jokerParamListDoc + "]"
	if strings.Contains(goParamListDoc, " ") || strings.Contains(goParamListDoc, ",") {
		goParamListDoc = "(" + goParamListDoc + ")"
	}
	return
}

func genGoCall(pkg, goFname string, goParams string) string {
	return "_" + pkg + "." + goFname + "(" + goParams + ")\n"
}

func genGoPost(indent string, pkg string, d *FuncDecl) (goResultAssign, jokerReturnTypeForDoc, goReturnTypeForDoc string, goReturnCode string) {
	fl := d.Type.Results
	if fl == nil || fl.List == nil {
		return
	}
	jokerReturnTypeForDoc, goReturnTypeForDoc, goReturnCode, goResultAssign = genGoPostList(indent, pkg, *fl)
	return
}

func genFuncCode(pkgBaseName, pkgDirUnix string, d *FuncDecl, goFname string) (fc funcCode) {
	var goPreCode, goParams, goResultAssign, goPostCode string

	fc.jokerParamList, fc.jokerParamListDoc, fc.jokerGoParams, fc.goParamList, fc.goParamListDoc, goPreCode, goParams =
		genGoPre("\t", d.Type.Params, goFname)
	goCall := genGoCall(pkgBaseName, d.Name.Name, goParams)
	goResultAssign, fc.jokerReturnTypeForDoc, fc.goReturnTypeForDoc, goPostCode =
		genGoPost("\t", pkgDirUnix, d)

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
func maybeConvertGoResult(pkg, call string, fl *FieldList) string {
	if fl == nil || len(fl.List) != 1 || (fl.List[0].Names != nil && len(fl.List[0].Names) > 1) {
		return call
	}
	named := false
	t := fl.List[0].Type
	for {
		stop := false
		switch v := t.(type) {
		case *Ident:
			qt := pkg + "." + v.Name
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

func genFunction(f string, fn *funcInfo) {
	genSymReset()
	d := fn.fd
	pkgDirUnix := fn.pkgDirUnix
	pkgBaseName := filepath.Base(pkgDirUnix)
	jfmt := `
(defn %s%s
%s  {:added "1.0"
   :go "%s"}
  [%s])
`
	goFname := funcNameAsGoPrivate(d.Name.Name)
	fc := genFuncCode(pkgBaseName, pkgDirUnix, d, goFname)
	jokerReturnType, goReturnType := jokerReturnTypeForGenerateCustom(fc.jokerReturnTypeForDoc, fc.goReturnTypeForDoc)

	var jok2gol string
	if jokerReturnType == "" {
		jok2gol = goFname
	} else {
		jokerReturnType += " "
		jok2gol = pkgBaseName + "." + d.Name.Name
		if _, found := packagesInfo[pkgDirUnix]; !found {
			panic(fmt.Sprintf("Cannot find package %s", pkgDirUnix))
		}
	}
	jok2golCall := maybeConvertGoResult(pkgDirUnix, jok2gol+fc.jokerGoParams, fn.fd.Type.Results)

	jokerFn := fmt.Sprintf(jfmt, jokerReturnType, d.Name.Name,
		commentGroupInQuotes(d.Doc, fc.jokerParamListDoc, fc.jokerReturnTypeForDoc,
			fc.goParamListDoc, fc.goReturnTypeForDoc),
		jok2golCall, fc.jokerParamList)

	gfmt := `
func %s(%s) %s {
%s}
`

	goFn := ""
	if jokerReturnType == "" { // TODO: Generate this anyway if it contains ABEND, so we can see what's needed.
		goFn = fmt.Sprintf(gfmt, goFname, fc.goParamList, goReturnType, fc.goCode)
	}

	if strings.Contains(jokerFn, "ABEND") || strings.Contains(goFn, "ABEND") {
		jokerFn = nonEmptyLineRegexp.ReplaceAllString(jokerFn, `;; $1`)
		goFn = nonEmptyLineRegexp.ReplaceAllString(goFn, `// $1`)
		trackAbends(jokerFn)
		trackAbends(goFn)
	} else {
		generatedFunctions++
		packagesInfo[pkgDirUnix].nonEmpty = true
		if jokerReturnType == "" {
			packagesInfo[pkgDirUnix].importsNative[pkgDirUnix] = exists
		} else {
			packagesInfo[pkgDirUnix].importsAutoGen[pkgDirUnix] = exists
		}
	}

	if _, ok := jokerCode[pkgDirUnix]; !ok {
		jokerCode[pkgDirUnix] = codeInfo{}
	}
	jokerCode[pkgDirUnix][d.Name.Name] = jokerFn

	if _, ok := goCode[pkgDirUnix]; !ok {
		goCode[pkgDirUnix] = codeInfo{} // There'll at least be a .joke file
	}
	if goFn != "" {
		goCode[pkgDirUnix][d.Name.Name] = goFn
	}
}

func notOption(arg string) bool {
	return arg == "-" || !strings.HasPrefix(arg, "-")
}

func usage() {
	fmt.Print(`
Usage: gostd options...

Options:
  --go <go-source-dir-name>      # Location of Go source tree's src/ subdirectory
  --overwrite                    # Overwrite any existing <joker-std-subdir> files, leaving existing files intact
  --replace                      # 'rm -fr <joker-std-subdir>' before creating <joker-std-subdir>
  --fresh                        # (Default) Refuse to overwrite existing <joker-std-subdir> directory
  --joker <joker-source-dir-name>  # Modify pertinent source files to reflect packages being created
  --verbose, -v                  # Print info on what's going on
  --summary                      # Print summary of #s of types, functions, etc.
  --output-code                  # Print generated code to stdout (used by test.sh)
  --empty                        # Generate empty packages (those with no Joker code)
  --dump                         # Use go's AST dump API on pertinent elements (functions, types, etc.)
  --no-timestamp                 # Don't put the time (and version) info in generated/modified files
  --help, -h                     # Print this information

If <joker-std-subdir> is not specified, no Go nor Clojure source files
(nor any other files nor directories) are created, effecting a sort of
"dry run".
`)
	os.Exit(0)
}

var currentTimeAndVersion = ""
var noTimeAndVersion = false

func curTimeAndVersion() string {
	if noTimeAndVersion {
		return "(omitted for testing)"
	}
	if currentTimeAndVersion == "" {
		by, _ := time.Now().MarshalText()
		currentTimeAndVersion = string(by) + " by version " + VERSION
	}
	return currentTimeAndVersion
}

// E.g.: \t_ "github.com/candid82/joker/std/go/std/net"
func updateJokerMain(pkgs []string, f string) {
	if verbose {
		fmt.Printf("Adding custom imports to %s\n", filepath.ToSlash(f))
	}

	m := "// Auto-modified by gostd at " + curTimeAndVersion() + "\n"

	newImports := `
package main

import (
`
	importPrefix := "\t_ \"github.com/candid82/joker/std/go/std/"
	for _, p := range pkgs {
		newImports += importPrefix + p + "\"\n"
	}
	newImports += `)
`

	m += newImports

	err := ioutil.WriteFile(f, []byte(m), 0777)
	check(err)
}

func updateCoreDotJoke(pkgs []string, f string) {
	if verbose {
		fmt.Printf("Adding custom loaded libraries to %s\n", filepath.ToSlash(f))
	}

	by, err := ioutil.ReadFile(f)
	check(err)
	m := string(by)
	flag := "Loaded-libraries added by gostd"
	endflag := "End gostd-added loaded-libraries"

	if !strings.Contains(m, flag) {
		m = strings.Replace(m, "\n  *loaded-libs* #{",
			"\n  *loaded-libs* #{\n   ;; "+flag+"\n   ;; "+endflag+"\n", 1)
		m = ";;;; Auto-modified by gostd at " + curTimeAndVersion() + "\n\n" + m
	}

	reImport := regexp.MustCompile("(?msU)" + flag + ".*" + endflag + "\n *?")
	newImports := "\n  "
	importPrefix := " 'go.std."
	curLine := ""
	for _, p := range pkgs {
		more := importPrefix + strings.Replace(p, "/", ".", -1)
		if curLine != "" && len(curLine)+len(more) > 77 {
			newImports += curLine + "\n  "
			curLine = more
		} else {
			curLine += more
		}
	}
	newImports += curLine
	m = reImport.ReplaceAllString(m, flag+newImports+"\n   ;; "+endflag+"\n   ")

	err = ioutil.WriteFile(f, []byte(m), 0777)
	check(err)
}

func updateGenerateCustom(pkgs []string, f string) {
	if verbose {
		fmt.Printf("Adding custom loaded libraries to %s\n", filepath.ToSlash(f))
	}

	m := ";;;; Auto-modified by gostd at " + curTimeAndVersion() + "\n\n"

	newImports := ""
	importPrefix := "'go.std."
	curLine := "(def custom-namespaces ["
	for _, p := range pkgs {
		more := importPrefix + strings.Replace(p, "/", ".", -1)
		if curLine != "" && len(curLine)+len(more) > 77 {
			newImports += curLine + "\n "
			curLine = more
		} else {
			curLine += more
		}
		importPrefix = " 'go.std."
	}
	newImports += curLine + `])

(apply require :reload custom-namespaces)

(doseq [ns-sym custom-namespaces]
  (let [ns-name (str ns-sym)
        dir (rpl ns-name "." "/")
        ns-name-final (rpl ns-name #".*[.]" "")]
    (debug "Processing custom namespace" ns-name "in" dir "final name" ns-name-final)
    (spit (ns-file-name dir ns-name-final)
          (remove-blanky-lines (generate-ns ns-sym ns-name ns-name-final)))))
`

	m += newImports

	err := ioutil.WriteFile(f, []byte(m), 0777)
	check(err)
}

func packageQuotedImportList(pi packageImports, prefix string, rename bool) string {
	imports := ""
	sortedPackageImports(pi,
		func(k string) {
			if rename {
				imports += prefix + "_" + path.Base(k) + ` "` + k + `"`
			} else {
				imports += prefix + `"` + k + `"`
			}
		})
	return imports
}

func main() {
	fset = token.NewFileSet() // positions are relative to fset
	dump = false

	length := len(os.Args)
	sourceDir := ""
	jokerSourceDir := ""
	replace := false
	overwrite := false
	summary := false
	generateEmpty := false
	outputCode := false

	var mode parser.Mode = parser.ParseComments

	for i := 1; i < length; i++ { // shift
		a := os.Args[i]
		if a[0] == "-"[0] {
			switch a {
			case "--help", "-h":
				usage()
			case "--version", "-V":
				fmt.Printf("%s version %s\n", os.Args[0], VERSION)
			case "--no-timestamp":
				noTimeAndVersion = true
			case "--dump":
				dump = true
			case "--overwrite":
				overwrite = true
				replace = false
			case "--replace":
				replace = true
				overwrite = false
			case "--fresh":
				replace = false
				overwrite = false
			case "--verbose", "-v":
				verbose = true
			case "--summary":
				summary = true
			case "--output-code":
				outputCode = true
			case "--empty":
				generateEmpty = true
			case "--go":
				if sourceDir != "" {
					panic("cannot specify --go <go-source-dir-name> more than once")
				}
				if i < length-1 && notOption(os.Args[i+1]) {
					i += 1 // shift
					sourceDir = os.Args[i]
				} else {
					panic("missing path after --go option")
				}
			case "--joker":
				if jokerSourceDir != "" {
					panic("cannot specify --joker <joker-source-dir-name> more than once")
				}
				if i < length-1 && notOption(os.Args[i+1]) {
					i += 1 // shift
					jokerSourceDir = os.Args[i]
				} else {
					panic("missing path after --joker option")
				}
			default:
				panic("unrecognized option " + a)
			}
		} else {
			panic("extraneous argument(s) starting with: " + a)
		}
	}

	if verbose {
		fmt.Printf("Default context: %v\n", build.Default)
	}

	if sourceDir == "" {
		goLink := "GO.link"
		si, e := os.Stat(goLink)
		if e == nil && !si.IsDir() {
			var by []byte
			by, e = ioutil.ReadFile(goLink)
			if e != nil {
				panic("Must specify --go <go-source-dir-name> option, or put <go-source-dir-name> as the first line of a file named ./GO.link")
			}
			m := string(by)
			if idx := strings.IndexAny(m, "\r\n"); idx == -1 {
				goLink = m
			} else {
				goLink = m[0:idx]
			}
			si, e = os.Stat(goLink)
		}
		if e != nil || !si.IsDir() {
			panic(fmt.Sprintf("Must specify --go <go-source-dir-name> option, or make %s a symlink (or text file containing the native path) pointing to the golang/go/ source directory", goLink))
		}
		sourceDir = goLink
	}

	sourceDir = filepath.Join(sourceDir, "src")

	if fi, e := os.Stat(filepath.Join(sourceDir, "go")); e != nil || !fi.IsDir() {
		if m, e := filepath.Glob(filepath.Join(sourceDir, "*.go")); e != nil || m == nil || len(m) == 0 {
			panic(fmt.Sprintf("Does not exist or is not a Go source directory: %s;\n%v", sourceDir, m))
		}
	}

	jokerLibDir := ""
	if jokerSourceDir != "" && jokerSourceDir != "-" {
		jokerLibDir = filepath.Join(jokerSourceDir, "std", "go", "std")
		if replace {
			if e := os.RemoveAll(jokerLibDir); e != nil {
				panic(fmt.Sprintf("Unable to effectively 'rm -fr %s'", jokerLibDir))
			}
		}

		if !overwrite {
			if _, e := os.Stat(jokerLibDir); e == nil ||
				(!strings.Contains(e.Error(), "no such file or directory") &&
					!strings.Contains(e.Error(), "The system cannot find the path specified.")) {
				msg := "already exists"
				if e != nil {
					msg = e.Error()
				}
				panic(fmt.Sprintf("Refusing to populate existing directory %s; please 'rm -fr' first, or specify --overwrite or --replace: %s",
					jokerLibDir, msg))
			}
			if e := os.MkdirAll(jokerLibDir, 0777); e != nil {
				panic(fmt.Sprintf("Cannot 'mkdir -p %s': %s", jokerLibDir, e.Error()))
			}
		}
	}

	err := walkDirs(filepath.Join(sourceDir, "."), mode)
	if err != nil {
		panic("Error walking directory " + sourceDir + ": " + fmt.Sprintf("%v", err))
	}

	sort.Strings(alreadySeen)
	for _, a := range alreadySeen {
		fmt.Fprintln(os.Stderr, a)
	}

	if verbose {
		/* Output map in sorted order to stabilize for testing. */
		sortedTypeInfoMap(types,
			func(t string, ti *typeInfo) {
				fmt.Printf("TYPE %s:\n", t)
				fmt.Printf("  %s\n", ti.file)
			})
	}

	/* Generate function code snippets in alphabetical order, to stabilize test output in re unsupported types. */
	sortedFuncInfoMap(qualifiedFunctions,
		func(f string, v *funcInfo) {
			genFunction(f, v)
		})

	var out *bufio.Writer
	var unbuf_out *os.File

	sortedPackageMap(jokerCode,
		func(pkgDirUnix string, v codeInfo) {
			pkgBaseName := path.Base(pkgDirUnix)
			if jokerLibDir != "" && jokerLibDir != "-" &&
				(generateEmpty || packagesInfo[pkgDirUnix].nonEmpty) {
				jf := filepath.Join(jokerLibDir, filepath.FromSlash(pkgDirUnix)+".joke")
				var e error
				e = os.MkdirAll(filepath.Dir(jf), 0777)
				unbuf_out, e = os.Create(jf)
				check(e)
				out = bufio.NewWriterSize(unbuf_out, 16384)

				pi := packagesInfo[pkgDirUnix]

				fmt.Fprintf(out,
					`;;;; Auto-generated by gostd at `+curTimeAndVersion()+`, do not edit!!

(ns
  ^{:go-imports [%s]
    :doc "Provides a low-level interface to the %s package."
    :empty %s}
  go.std.%s)
`,
					strings.TrimPrefix(packageQuotedImportList(pi.importsAutoGen, " ", false), " "),
					pkgDirUnix,
					func() string {
						if pi.nonEmpty {
							return "false"
						} else {
							return "true"
						}
					}(),
					strings.Replace(pkgDirUnix, "/", ".", -1))
			}
			sortedCodeMap(v,
				func(f string, w string) {
					if outputCode {
						fmt.Printf("JOKER FUNC %s.%s has:%v\n",
							pkgBaseName, f, w)
					}
					if out != nil {
						out.WriteString(w)
					}
				})
			if out != nil {
				out.Flush()
				unbuf_out.Close()
				out = nil
			}
		})
	sortedPackageMap(goCode,
		func(pkgDirUnix string, v codeInfo) {
			pkgBaseName := path.Base(pkgDirUnix)
			pi := packagesInfo[pkgDirUnix]
			packagesInfo[pkgDirUnix].hasGoFiles = true
			pkgDirNative := filepath.FromSlash(pkgDirUnix)

			if jokerLibDir != "" && jokerLibDir != "-" &&
				(generateEmpty || packagesInfo[pkgDirUnix].nonEmpty) {
				gf := filepath.Join(jokerLibDir, pkgDirNative,
					pkgBaseName+"_native.go")
				var e error
				e = os.MkdirAll(filepath.Dir(gf), 0777)
				check(e)
				unbuf_out, e = os.Create(gf)
				check(e)
				out = bufio.NewWriterSize(unbuf_out, 16384)

				importCore := ""
				if _, f := pi.importsNative[pkgDirUnix]; f {
					importCore = `
	. "github.com/candid82/joker/core"`
				}

				fmt.Fprintf(out,
					`// Auto-generated by gostd at `+curTimeAndVersion()+`, do not edit!!

package %s

import (%s%s
)
`,
					pkgBaseName,
					packageQuotedImportList(pi.importsNative, "\n\t", true),
					importCore)
			}
			sortedCodeMap(v,
				func(f string, w string) {
					if outputCode {
						fmt.Printf("GO FUNC %s.%s has:%v\n",
							pkgBaseName, f, w)
					}
					if out != nil {
						out.WriteString(w)
					}
				})
			if out != nil {
				out.Flush()
				unbuf_out.Close()
				out = nil
			}
		})

	if jokerSourceDir != "" && jokerSourceDir != "-" {
		var packagesArray = []string{} // Relative package pathnames in alphabetical order
		var dotJokeArray = []string{}  // Relative package pathnames in alphabetical order

		sortedPackagesInfo(packagesInfo,
			func(p string, i *packageInfo) {
				if !generateEmpty && !i.nonEmpty {
					return
				}
				if i.hasGoFiles {
					packagesArray = append(packagesArray, p)
				}
				dotJokeArray = append(dotJokeArray, p)
			})
		updateJokerMain(packagesArray, filepath.Join(jokerSourceDir, "custom.go"))
		updateCoreDotJoke(dotJokeArray, filepath.Join(jokerSourceDir, "core", "data", "core.joke"))
		updateGenerateCustom(packagesArray, filepath.Join(jokerSourceDir, "std", "generate-custom.joke"))
	}

	if verbose || summary {
		fmt.Printf("ABENDs:")
		printAbends(abends)
		fmt.Printf("\nTotals: types=%d functions=%d methods=%d (%s%%) standalone=%d (%s%%) generated=%d (%s%%)\n",
			len(types), len(qualifiedFunctions)+methods, methods,
			pct(methods, len(qualifiedFunctions)+methods),
			len(qualifiedFunctions), pct(len(qualifiedFunctions), len(qualifiedFunctions)+methods),
			generatedFunctions, pct(generatedFunctions, len(qualifiedFunctions)))
	}

	os.Exit(0)
}

func pct(i, j int) string {
	if j == 0 {
		return "--"
	}
	return fmt.Sprintf("%0.2f", (float64(i)/float64(j))*100.0)
}

func init() {
	nonEmptyLineRegexp = regexp.MustCompile(`(?m)^(.)`)
	abendRegexp = regexp.MustCompile(`ABEND([0-9]+)`)
}
