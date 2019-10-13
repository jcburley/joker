package godb

import (
	"fmt"
	. "go/ast"
	"go/token"
	"path/filepath"
	. "strings"
)

var Fset *token.FileSet

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
	Pkg        *Package
	RootUnix   string
	PkgDirUnix string
	NsRoot     string
	decls      map[string]DeclInfo
}

var packagesByName = map[string]*PackageDb{}

var PackagesAsDiscovered = []*PackageDb{}

type DeclInfo struct {
	name string
	node Node
	pos  token.Pos
}

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
	for _, f := range pkg.Files {
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

	pkgDb := &PackageDb{pkg, rootUnix, pkgDirUnix, nsRoot, decls}
	packagesByName[pkgDirUnix] = pkgDb
	PackagesAsDiscovered = append(PackagesAsDiscovered, pkgDb)
}

func ResolveInPackage(pkg, name string) Node {
	p := packagesByName[pkg]
	if p == nil {
		return nil
	}
	return p.decls[name].node
}

func ResolveSelector(n Node) string {
	return ""
}

func Resolve(n Node) Node {
	pkg := GoPackageForExpr(n.(Expr))
	switch o := n.(type) {
	case *Ident:
		return ResolveInPackage(pkg, o.Name)
	case *SelectorExpr:
		return ResolveInPackage(ResolveSelector(o.X.(Node)), o.Sel.Name)
	default:
		panic(fmt.Sprintf("don't know how to resolve node %v", o))
	}
	return nil
}
