package godb

import (
	"fmt"
	. "github.com/candid82/joker/tools/gostd/utils"
	. "go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	. "strings"
)

var Fset *token.FileSet
var Dump bool

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
	prefix  string // E.g. "/home/user/go/src"
	cljRoot string // E.g. "go.std."
}

var mappings = []mapping{}

func AddMapping(dir string, root string) {
	for _, m := range mappings {
		if HasPrefix(dir, m.prefix) {
			panic(fmt.Sprintf("duplicate mapping %s and %s", dir, m.prefix))
		}
	}
	mappings = append(mappings, mapping{dir, root})
}

func goPackageForDirname(dirName string) (pkg, prefix string) {
	for _, m := range mappings {
		if HasPrefix(dirName, m.prefix) {
			return dirName[len(m.prefix)+1:], m.cljRoot
		}
	}
	return "", mappings[0].cljRoot
}

func GoPackageForExpr(e Expr) string {
	dirName := filepath.Dir(Fset.Position(e.Pos()).Filename)
	pkg, _ := goPackageForDirname(dirName)
	if pkg == "" {
		panic(fmt.Sprintf("no mapping for %s", dirName))
	}
	return pkg
}

func GoPackageForTypeSpec(ts *TypeSpec) string {
	dirName := filepath.Dir(Fset.Position(ts.Pos()).Filename)
	pkg, _ := goPackageForDirname(dirName)
	if pkg == "" {
		panic(fmt.Sprintf("no mapping for %s", dirName))
	}
	return pkg
}

func ClojureNamespaceForPos(p token.Position) string {
	dirName := filepath.Dir(p.Filename)
	pkg, root := goPackageForDirname(dirName)
	if pkg == "" {
		panic(fmt.Sprintf("no mapping for %s", dirName))
	}
	return root + ReplaceAll(pkg, (string)(filepath.Separator), ".")
}

func ClojureNamespaceForExpr(e Expr) string {
	return ClojureNamespaceForPos(Fset.Position(e.Pos()))
}

func ClojureNamespaceForDirname(d string) string {
	pkg, root := goPackageForDirname(d)
	if pkg == "" {
		pkg = root + d
	}
	return ReplaceAll(pkg, (string)(filepath.Separator), ".")
}

func GoPackageBaseName(e Expr) string {
	return filepath.Base(filepath.Dir(Fset.Position(e.Pos()).Filename))
}

type PackageDb struct {
	Pkg      *Package // nil means Universal scope
	RootUnix string
	DirUnix  string
	BaseName string
	NsRoot   string
	decls    map[string]DeclInfo
}

var packagesByName = map[string]*PackageDb{}

var PackagesAsDiscovered = []*PackageDb{}

type DeclInfo struct {
	name string
	node Node
	pos  token.Pos
}

type GoFile struct {
	Package *PackageDb
	Name    string
	Spaces  *map[string]string // maps "foo" (in a reference such as "foo.Bar") to the pkgDirUnix in which it is defined
	NsRoot  string             // "go.std." or whatever is desired as the root namespace
}

var GoFiles = map[string]*GoFile{}

func newDecl(decls *map[string]DeclInfo, pkg string, name *Ident, node Node) {
	if !IsExported(name.Name) {
		return
	}
	if e, found := (*decls)[name.Name]; found {
		panic(fmt.Sprintf("already seen decl %s.%s at %s, now: %v at %s", pkg, e.name, WhereAt(e.pos), node, WhereAt(name.NamePos)))
	}
	(*decls)[name.Name] = DeclInfo{name.Name, node, name.NamePos}
}

func RegisterPackage(rootUnix, pkgDirUnix, nsRoot string, pkg *Package) {
	if _, found := packagesByName[pkgDirUnix]; found {
		panic(fmt.Sprintf("already seen package %s", pkgDirUnix))
	}

	decls := map[string]DeclInfo{}
	pkgDb := &PackageDb{pkg, rootUnix, pkgDirUnix, filepath.Base(pkgDirUnix), nsRoot, decls}

	for path, f := range pkg.Files {
		goFilePathUnix := TrimPrefix(filepath.ToSlash(path), rootUnix+"/")
		if egf, found := GoFiles[goFilePathUnix]; found {
			panic(fmt.Sprintf("Found %s twice -- now in %s, previously in %s!", goFilePathUnix, pkgDirUnix, egf.Package.DirUnix))
		}
		importsMap := map[string]string{}

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
				as = filepath.Base(importPath)
			}
			importsMap[as] = importPath
		}

		gf := &GoFile{
			Package: pkgDb,
			Name:    goFilePathUnix,
			Spaces:  &importsMap,
			NsRoot:  nsRoot,
		}
		GoFiles[goFilePathUnix] = gf

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
	packagesByName[pkgDirUnix] = pkgDb
	PackagesAsDiscovered = append(PackagesAsDiscovered, pkgDb)
}

func ResolveInPackage(pkg, name string) Node {
	p := packagesByName[pkg]
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
	return nil
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

	pkgDb := &PackageDb{nil, "", "", "", "", decls}
	packagesByName[""] = pkgDb
}
