package main

import (
	"fmt"
	. "go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type funcInfo struct {
	fd           *FuncDecl
	sourceFile   *goFile
	refersToSelf bool // whether :go-imports should list itself
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
func processFuncDecl(gf *goFile, pkgDirUnix, filename string, f *File, fd *FuncDecl) bool {
	if dump {
		fmt.Printf("Func in pkgDirUnix=%s filename=%s:\n", pkgDirUnix, filename)
		Print(fset, fd)
	}
	fname := pkgDirUnix + "." + fd.Name.Name
	if v, ok := qualifiedFunctions[fname]; ok {
		alreadySeen = append(alreadySeen,
			fmt.Sprintf("NOTE: Already seen function %s in %s, yet again in %s",
				fname, v.sourceFile.name, filename))
	}
	qualifiedFunctions[fname] = &funcInfo{fd, gf, false}
	return true
}

type typeInfo struct {
	sourceFile *goFile
	td         *TypeSpec
	where      token.Pos
	typeCode   string
}

type typeMap map[string]*typeInfo

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

// Maps qualified typename ("path/to/pkg.TypeName") to type info.
var types = map[string]*typeInfo{}

func processTypeSpec(gf *goFile, pkg string, pathUnix string, f *File, ts *TypeSpec) {
	typename := pkg + "." + ts.Name.Name
	if dump {
		fmt.Printf("Type %s at %s:\n", typename, whereAt(ts.Pos()))
		Print(fset, ts)
	}
	if c, ok := types[typename]; ok {
		fmt.Fprintf(os.Stderr, "WARNING: type %s found at %s and now again at %s\n",
			typename, whereAt(c.where), whereAt(ts.Pos()))
	}
	types[typename] = &typeInfo{gf, ts, ts.Pos(), ""}
}

func processTypeSpecs(gf *goFile, pkg string, pathUnix string, f *File, tss []Spec) {
	for _, spec := range tss {
		ts := spec.(*TypeSpec)
		if isPrivate(ts.Name.Name) {
			continue // Skipping non-exported functions
		}
		processTypeSpec(gf, pkg, pathUnix, f, ts)
	}
}

// Returns whether any public functions were actually processed.
func processDecls(gf *goFile, pkgDirUnix, pathUnix string, f *File) (found bool) {
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
			if processFuncDecl(gf, pkgDirUnix, pathUnix, f, v) {
				found = true
			}
		case *GenDecl:
			if v.Tok != token.TYPE {
				continue
			}
			processTypeSpecs(gf, pkgDirUnix, pathUnix, f, v.Specs)
		default:
			panic(fmt.Sprintf("unrecognized Decl type %T at: %s", v, whereAt(v.Pos())))
		}
	}
	return
}

type goFile struct {
	name       string
	rootUnix   string
	pkgDirUnix string
	pkgName    string
	spaces     *map[string]string // maps "foo" (in a reference such as "foo.Bar") to the pkgDirUnix in which it is defined
}

var goFiles = map[string]*goFile{}

func processPackageMeta(rootUnix, pkgDirUnix, goFilePathUnix string, f *File) (gf *goFile) {
	if egf, found := goFiles[goFilePathUnix]; found {
		panic(fmt.Sprintf("Found %s twice -- now in %s, previously in %s!", goFilePathUnix, pkgDirUnix, egf.pkgDirUnix))
	}
	importsMap := map[string]string{}
	gf = &goFile{goFilePathUnix, rootUnix, pkgDirUnix, f.Name.Name, &importsMap}
	goFiles[goFilePathUnix] = gf

	for _, imp := range f.Imports {
		if dump {
			fmt.Printf("Import for file %s:\n", goFilePathUnix)
			Print(fset, imp)
		}
		importPath, err := strconv.Unquote(imp.Path.Value)
		check(err)
		var as string
		if n := imp.Name; n != nil {
			switch n.Name {
			case "_":
				continue // Ignore these
			case ".":
				fmt.Fprintf(os.Stderr, "ERROR: `.' not supported in import directive at %v\n", whereAt(n.NamePos))
				continue
			default:
				as = n.Name
			}
		} else {
			as = filepath.Base(importPath)
		}
		importsMap[as] = importPath
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

/* Map (Unix-style) relative path to package info */
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

func processPackage(rootUnix, pkgDirUnix string, p *Package) {
	if verbose {
		fmt.Printf("Processing package=%s:\n", pkgDirUnix)
	}

	found := false
	for path, f := range p.Files {
		goFilePathUnix := strings.TrimPrefix(filepath.ToSlash(path), rootUnix+"/")
		gf := processPackageMeta(rootUnix, pkgDirUnix, goFilePathUnix, f)
		if processDecls(gf, pkgDirUnix, goFilePathUnix, f) {
			found = true
		}
	}
	if found {
		if _, ok := packagesInfo[pkgDirUnix]; !ok {
			packagesInfo[pkgDirUnix] = &packageInfo{packageImports{}, packageImports{}, false, false}
			goCode[pkgDirUnix] = codeInfo{fnCodeMap{}, typeMap{}}
			clojureCode[pkgDirUnix] = codeInfo{fnCodeMap{}, typeMap{}}
		}
	}
}

func processDir(root, rootUnix, path string, mode parser.Mode) error {
	pkgDir := strings.TrimPrefix(path, root+string(filepath.Separator))
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

	for pkgBaseName, v := range pkgs {
		if pkgBaseName != filepath.Base(path) {
			if verbose {
				fmt.Printf("NOTICE: Package %s is defined in %s -- ignored due to name mismatch\n",
					pkgBaseName, path)
			}
		} else {
			processPackage(rootUnix, pkgDirUnix, v)
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

func walkDirs(root string, mode parser.Mode) error {
	rootUnix := filepath.ToSlash(root)
	target, err := filepath.EvalSymlinks(root)
	check(err)
	err = filepath.Walk(target,
		func(path string, info os.FileInfo, err error) error {
			rel := strings.Replace(path, target, root, 1)
			relUnix := filepath.ToSlash(rel)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Skipping %s due to: %v\n", relUnix, err)
				return err
			}
			if rel == root {
				return nil // skip (implicit) "."
			}
			if excludeDirs[filepath.Base(rel)] {
				if verbose {
					fmt.Printf("Excluding %s\n", relUnix)
				}
				return filepath.SkipDir
			}
			if info.IsDir() {
				if verbose {
					fmt.Printf("Walking from %s to %s\n", rootUnix, relUnix)
				}
				return processDir(root, rootUnix, rel, mode)
			}
			return nil // not a directory
		})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while walking %s: %v\n", root, err)
		return err
	}

	return err
}

type fnCodeInfo struct {
	sourceFile *goFile
	fnCode     string
}

type fnCodeMap map[string]fnCodeInfo

type codeInfo struct {
	functions fnCodeMap
	types     typeMap
}

/* Map relative (Unix-style) package names to maps of function names to code info and strings. */
var clojureCode = map[string]codeInfo{}
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

func sortedCodeMap(m codeInfo, f func(k string, v fnCodeInfo)) {
	var keys []string
	for k, _ := range m.functions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, m.functions[k])
	}
}
