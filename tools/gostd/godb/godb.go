package godb

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/astutils"
	"github.com/candid82/joker/tools/gostd/paths"
	. "github.com/candid82/joker/tools/gostd/utils"
	. "go/ast"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strconv"
	. "strings"
)

// Root of Clojure source tree for generated import lines (so, using Unix path syntax).
var ClojureSourceDir = "github.com/candid82/joker"

// Clojure source tree's core directory for generated import lines (so, using Unix path syntax).
var ClojureCoreDir = path.Join(ClojureSourceDir, "core")

// Set the (Unix-syntax, i.e. slash-delimited) root for generated
// import lines to the given host-syntax path, with the local path
// prefix removed.  E.g. if p=="." and r=="/home/me/go/src", and
// Abs(p)=="/home/me/go/src/github.com/candid82/joker", then the
// resulting root for import lines would be
// "github.com/candid82/joker".
func SetClojureSourceDir(p string, r string) {
	abs, err := filepath.Abs(p)
	Check(err)
	imp := TrimPrefix(abs, r+string(filepath.Separator))
	ClojureSourceDir = filepath.ToSlash(imp)
	ClojureCoreDir = path.Join(ClojureSourceDir, "core")
}

var Fset *token.FileSet
var Dump bool
var Verbose bool

var NumMethods int
var NumGeneratedMethods int

func WhereAt(p token.Pos) string {
	return fmt.Sprintf("%s", Fset.Position(p).String())
}

func FileAt(p token.Pos) string {
	return token.Position{Filename: Fset.Position(p).Filename,
		Offset: 0, Line: 0, Column: 0}.String()
}

type mapping struct {
	prefix  paths.UnixPath // E.g. "/home/user/go/src"
	cljRoot string         // E.g. "go.std."
}

var mappings = []mapping{}

func AddMapping(dirNative paths.NativePath, root string) {
	dir := dirNative.ToUnix()
	dirString := dir.String()
	for _, m := range mappings {
		if HasPrefix(dirString, m.prefix.String()) {
			panic(fmt.Sprintf("duplicate mapping %s and %s", dirString, m.prefix))
		}
	}
	mappings = append(mappings, mapping{dir, root})
}

func goPackageForDirname(dirName string) (pkg, prefix string) {
	for _, m := range mappings {
		if HasPrefix(dirName, m.prefix.String()) {
			return dirName[len(m.prefix.String())+1:], m.cljRoot
		}
	}
	return "", mappings[0].cljRoot
}

func GoFilenameForPos(p token.Pos) string {
	fn := Fset.Position(p).Filename
	dirName := filepath.ToSlash(filepath.Dir(fn))
	pkg, _ := goPackageForDirname(dirName)
	if pkg == "" {
		panic(fmt.Sprintf("no mapping for %s", dirName))
	}
	return filepath.ToSlash(filepath.Join(pkg, filepath.Base(fn)))
}

func GoFilenameForExpr(e Expr) string {
	if id, yes := e.(*Ident); yes && astutils.IsBuiltin(id.Name) {
		return "" // A builtin, so not package-qualified.
	}
	return GoFilenameForPos(e.Pos())
}

func GoFilenameForTypeSpec(ts *TypeSpec) string {
	return GoFilenameForPos(ts.Pos())
}

func GoPackageForPos(p token.Pos) string {
	dirName := path.Dir(filepath.ToSlash(Fset.Position(p).Filename))
	pkg, _ := goPackageForDirname(dirName)
	if pkg == "" {
		panic(fmt.Sprintf("no mapping for %s", dirName))
	}
	return pkg
}

func GoPackageForExpr(e Expr) string {
	if id, yes := e.(*Ident); yes && astutils.IsBuiltin(id.Name) {
		return "" // A builtin, so not package-qualified.
	}
	return GoPackageForPos(e.Pos())
}

func GoPackageForTypeSpec(ts *TypeSpec) string {
	return GoPackageForPos(ts.Pos())
}

func ClojureNamespaceForPos(p token.Position) string {
	dirName := path.Dir(filepath.ToSlash(p.Filename))
	pkg, root := goPackageForDirname(dirName)
	if pkg == "" {
		panic(fmt.Sprintf("no mapping for %s given %s", dirName, filepath.ToSlash(p.Filename)))
	}
	return root + ReplaceAll(pkg, "/", ".")
}

func ClojureNamespaceForExpr(e Expr) string {
	if id, yes := e.(*Ident); yes && astutils.IsBuiltin(id.Name) {
		panic(fmt.Sprintf("no Clojure namespace for builtin `%s'", id.Name))
	}
	return ClojureNamespaceForPos(Fset.Position(e.Pos()))
}

func ClojureNamespaceForDirname(d string) string {
	pkg, root := goPackageForDirname(d)
	if pkg == "" {
		pkg = root + d
	}
	return ReplaceAll(pkg, "/", ".")
}

func ClojureNamespaceForGoFile(pkg string, g *GoFile) string {
	if fullPkgName, found := (*g.Spaces)[pkg]; found {
		f := fullPkgName.String()
		p, root := goPackageForDirname(f)
		if p == "" {
			p = root + f
		}
		return ReplaceAll(p, "/", ".")
	}
	panic(fmt.Sprintf("could not find %s in %s",
		pkg, g.Name))
}

func GoPackageBaseName(e Expr) string {
	return path.Base(path.Dir(filepath.ToSlash(Fset.Position(e.Pos()).Filename)))
}

type PackageDb struct {
	Pkg      *Package // nil means Universal scope
	Root     paths.UnixPath
	Dir      paths.UnixPath
	BaseName string
	NsRoot   string // "go.std." or whatever is desired as the root namespace
	decls    map[string]DeclInfo
}

var packagesByUnixPath = map[string]*PackageDb{}

var PackagesAsDiscovered = []*PackageDb{}

type DeclInfo struct {
	name string
	node Node
	pos  token.Pos
}

type GoFile struct {
	Package *PackageDb
	Name    paths.UnixPath
	Spaces  *map[string]paths.UnixPath // maps "foo" (in a reference such as "foo.Bar") to the package in which it is defined
}

// Map relative (Unix-style) filenames to objects with info on them.
var GoFilesRelative = map[string]*GoFile{}

// Map absolute (Unix-style) filenames to objects with info on them.
var GoFilesAbsolute = map[string]*GoFile{}

func IsAvailable(p paths.UnixPath) (available bool) {
	_, available = packagesByUnixPath[p.String()]
	return
}

func GoFileForPos(p token.Pos) *GoFile {
	fullPathUnix := paths.Unix(Fset.Position(p).Filename)

	gf, ok := GoFilesAbsolute[fullPathUnix]

	if !ok {
		panic(fmt.Sprintf("could not find referring file %s at %s",
			fullPathUnix, WhereAt(p)))
	}

	return gf
}

func GoFileForExpr(e Expr) *GoFile {
	return GoFileForPos(e.Pos())
}

func GoFileForTypeSpec(ts *TypeSpec) *GoFile {
	return GoFileForPos(ts.Pos())
}

func newDecl(decls *map[string]DeclInfo, pkg paths.UnixPath, name *Ident, node Node) {
	if !IsExported(name.Name) /* || (pkg.String() == "unsafe" && name.Name == "ArbitraryType") */ {
		if IsExported(name.Name) {
			if Verbose {
				fmt.Printf("Excluding mythical type %s.%s\n", pkg.String(), name.Name)
			}
		}
		return
	}
	if e, found := (*decls)[name.Name]; found {
		panic(fmt.Sprintf("already seen decl %s.%s at %s, now: %v at %s", pkg, e.name, WhereAt(e.pos), node, WhereAt(name.NamePos)))
	}
	(*decls)[name.Name] = DeclInfo{name.Name, node, name.NamePos}
}

func RegisterPackage(rootUnix, pkgDirUnix paths.UnixPath, nsRoot string, pkg *Package) {
	if _, found := packagesByUnixPath[pkgDirUnix.String()]; found {
		panic(fmt.Sprintf("already seen package %s", pkgDirUnix))
	}

	decls := map[string]DeclInfo{}
	pkgDb := &PackageDb{pkg, rootUnix, pkgDirUnix, pkgDirUnix.Base(), nsRoot, decls}

	for p, f := range pkg.Files {
		absFilePathUnix := paths.NewNativePath(p).ToUnix()
		goFilePathUnix, _ := absFilePathUnix.RelativeTo(rootUnix)
		if egf, found := GoFilesRelative[goFilePathUnix.String()]; found {
			panic(fmt.Sprintf("Found %s twice -- now in %s, previously in %s!", goFilePathUnix, pkgDirUnix, egf.Package.Dir))
		}
		importsMap := map[string]paths.UnixPath{}

		for _, imp := range f.Imports {
			if Dump {
				fmt.Printf("Import for file %s:\n", goFilePathUnix)
				Print(Fset, imp)
			}
			importPath, err := strconv.Unquote(imp.Path.Value)
			Check(err)
			var as string
			if n := imp.Name; n != nil {
				switch n.Name {
				case "_":
					continue // Ignore these
				case ".":
					fmt.Fprintf(os.Stderr, "ERROR: `.' not supported in import directive at %v\n", WhereAt(n.NamePos))
					continue
				default:
					as = n.Name
				}
			} else {
				as = path.Base(importPath)
			}
			importsMap[as] = paths.NewUnixPath(importPath)
		}

		gf := &GoFile{
			Package: pkgDb,
			Name:    goFilePathUnix,
			Spaces:  &importsMap,
		}
		GoFilesRelative[goFilePathUnix.String()] = gf
		GoFilesAbsolute[absFilePathUnix.String()] = gf

		for _, d := range f.Decls {
			switch o := d.(type) {
			case *FuncDecl:
				if o.Recv == nil {
					newDecl(&decls, pkgDirUnix, o.Name, o)
				}
			case *GenDecl:
				for _, s := range o.Specs {
					switch o.Tok {
					case token.IMPORT: // Ignore these
					case token.CONST:
					case token.TYPE:
						newDecl(&decls, pkgDirUnix, s.(*TypeSpec).Name, s)
					case token.VAR:
					default:
						panic(fmt.Sprintf("unrecognized GenDecl type %d for %v", o.Tok, o))
					}
				}
			}
		}
	}

	pkgDb.decls = decls
	packagesByUnixPath[pkgDirUnix.String()] = pkgDb
	PackagesAsDiscovered = append(PackagesAsDiscovered, pkgDb)
}

func ResolveInPackage(pkg, name string) Node {
	p := packagesByUnixPath[pkg]
	if p == nil {
		return nil
	}
	res := p.decls[name].node
	return res
}

func ResolveSelector(n Node) string {
	return n.(*Ident).Name
}

func Resolve(n Node) Node {
	pkg := GoPackageForExpr(n.(Expr))
	switch o := n.(type) {
	case *Ident:
		p := ""
		if IsExported(o.Name) {
			p = pkg
		}
		return ResolveInPackage(p, o.Name)
	case *SelectorExpr:
		return ResolveInPackage(ResolveSelector(o.X.(Node)), o.Sel.Name)
	default:
		panic(fmt.Sprintf("don't know how to resolve node %v", o))
	}
}

func init() {
	eid := &Ident{Name: "error"}
	enames := &Ident{Name: "Error"}
	emethodft := &FuncType{Params: &FieldList{List: []*Field{}}, Results: &FieldList{List: []*Field{}}}
	emethod := &Field{Names: []*Ident{enames}, Type: emethodft}
	emethods := &FieldList{List: []*Field{emethod}}
	etype := &InterfaceType{Methods: emethods}
	ets := &TypeSpec{Name: eid, Type: etype}
	decl := DeclInfo{"error", ets, 0}

	decls := map[string]DeclInfo{}
	decls["error"] = decl

	pkgDb := &PackageDb{nil, paths.NewUnixPath(""), paths.NewUnixPath(""), "", "", decls}
	packagesByUnixPath[""] = pkgDb
}
