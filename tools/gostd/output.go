package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/candid82/joker/tools/gostd/genutils"
	. "github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/imports"
	"github.com/candid82/joker/tools/gostd/paths"
	"go/doc"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

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

func RegisterNamespaces(namespaces []string, clojureSourceDir paths.NativePath) {
	writeCustomLibsGo(namespaces, clojureSourceDir, paths.NewNativePath("g_custom.go"))
}

func RegisterClojureFiles(clojureFiles []string, clojureSourceDir paths.NativePath) {
	writeCustomLibsClojure(clojureFiles, clojureSourceDir, paths.NewNativePath(filepath.Join("core", "data", "g_customlibs.joke")))
}

func RegisterGoTypeSwitch(types []TypeInfo, clojureSourceDir paths.NativePath) {
	writeGoTypeSwitch(types, clojureSourceDir, paths.NewNativePath(filepath.Join("core", "g_goswitch.go")))
}

func writeCustomLibsGo(namespaces []string, dir, f paths.NativePath) {
	if Verbose {
		fmt.Printf("Adding %d custom imports to %s\n", len(namespaces), f)
	}

	newImports := ""
	importPrefix := "\t_ "
	gostd := generatedPkgPrefix + goStdPrefix.String()
	for _, ns := range namespaces {
		newImports += importPrefix + fmt.Sprintf("%q\n", gostd+NamespacesInfo[ns].Package) // TODO: Precompute this %q arg in NamespacesInfo
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "go-custom-libs.tmpl", newImports)

	if dir.String() != "" {
		err := ioutil.WriteFile(dir.Join(f.String()).ToNative().String(), buf.Bytes(), 0777)
		Check(err)
	}
}

func writeCustomLibsClojure(namespaces []string, dir, f paths.NativePath) {
	if Verbose {
		fmt.Printf("Adding %d custom loaded libraries to %s\n", len(namespaces), f)
	}

	m := ""
	for _, ns := range namespaces {
		m += "     '" + ns + "\n"
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "clojure-custom-libs.tmpl", m)

	if dir.String() != "" {
		err := ioutil.WriteFile(dir.Join(f.String()).ToNative().String(), buf.Bytes(), 0777)
		Check(err)
	}
}

var Ordinal = map[TypeInfo]uint{}
var SwitchableTypes []TypeInfo // Set by GenTypeInfo() to subset of AllTypesSorted() that will go into the Go Type Switch

func writeGoTypeSwitch(allTypes []TypeInfo, dir, f paths.NativePath) {
	types := SwitchableTypes

	if Verbose {
		fmt.Printf("Adding only %d types (out of %d) to %s\n", len(types), len(allTypes), f)
	}

	var cases []map[string]interface{}
	var importeds = &imports.Imports{}
	for _, t := range types {
		specificity := t.Specificity()
		if specificity == 0 {
			// These are empty interface{} types, and so really can't be specifically matched to anything.
			continue
		}
		pkgPlusSeparator := ""
		if t.GoPackage() != "" {
			pkgPlusSeparator = importeds.AddPackage(t.GoPackage(), "", "", "", true, token.NoPos) + "."
		}
		cases = append(cases, map[string]interface{}{
			"match": fmt.Sprintf(t.GoPattern(), pkgPlusSeparator+t.GoBaseName()),
			"specificity": func() uint {
				if specificity != ConcreteType {
					return specificity
				} else {
					return 0 // These won't occur here (if they did, no comment would be emitted).
				}
			}(),
			"ordinal": Ordinal[t],
		})
	}

	info := map[string]interface{}{}
	info["Imports"] = importeds.QuotedList("\n\t")
	info["NumberOfTypes"] = len(types)
	info["Cases"] = cases

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "go-type-switch.tmpl", info)

	if dir.String() != "" {
		err := ioutil.WriteFile(dir.Join(f.String()).ToNative().String(), buf.Bytes(), 0777)
		Check(err)
	}
}

func outputClojureCode(ns string, v CodeInfo, clojureLibDir string, generateEmpty bool) {
	nsi := NamespacesInfo[ns]

	if !generateEmpty && !nsi.NonEmpty {
		return
	}

	var out *bufio.Writer
	var unbuf_out *os.File

	jf := filepath.Join(clojureLibDir, filepath.FromSlash(nsi.Package)+".joke")
	var e error
	e = os.MkdirAll(filepath.Dir(jf), 0777)
	Check(e)
	unbuf_out, e = os.Create(jf)
	Check(e)
	out = bufio.NewWriterSize(unbuf_out, 16384)

	defer func() {
		out.Flush()
		unbuf_out.Close()
	}()

	if out != nil {
		importPath, _ := filepath.Abs("/")
		myDoc := doc.New(PackagesInfo[nsi.Package].Pkg, importPath, doc.AllDecls)
		pkgDoc := fmt.Sprintf("Provides a low-level interface to the %s package.", nsi.Package)
		if myDoc.Doc != "" {
			pkgDoc += "\n\n" + myDoc.Doc
		}

		info := map[string]interface{}{
			"Imports":   nsi.ImportsAutoGen.AsClojureMap(),
			"Doc":       strconv.Quote(pkgDoc),
			"Empty":     !nsi.NonEmpty,
			"Namespace": goNsPrefix + strings.ReplaceAll(nsi.Package, "/", "."),
		}

		buf := new(bytes.Buffer)
		Templates.ExecuteTemplate(buf, "clojure-code.tmpl", info)

		if n, err := out.Write(buf.Bytes()); err != nil {
			panic(fmt.Sprintf("n=%d err=%s", n, err))
		}
	}

	SortedConstantInfoMap(v.Constants,
		func(c string, ci *ConstantInfo) {
			out.WriteString(ci.Def)
		})

	SortedVariableInfoMap(v.Variables,
		func(c string, ci *VariableInfo) {
			out.WriteString(ci.Def)
		})

	SortedTypeInfoMap(v.Types,
		func(t string, ti TypeInfo) {
			if !ti.IsCustom() {
				return
			}
			out.WriteString(ClojureCodeForType[ti])
		})

	SortedCodeMap(v,
		func(f string, w *FnCodeInfo) {
			out.WriteString(w.FnCode)
		})

	SortedTypeDefinitions(v.InitTypes,
		func(ti TypeInfo) {
			if !ti.IsCustom() {
				return
			}
			tmn := ti.TypeMappingsName()
			if tmn == "" || ti.GoBaseName() == "" || !ti.IsExported() || ti.IsArbitraryType() {
				return
			}
			typeDoc := ti.Doc()
			specificity := ""
			if ti.Specificity() != ConcreteType {
				specificity = fmt.Sprintf("    :specificity %d\n", ti.Specificity())
			}

			info := map[string]string{
				"Doc":         strconv.Quote(typeDoc),
				"Specificity": specificity,
				"GoName":      tmn,
				"ClojureName": fmt.Sprintf(ti.ClojurePattern(), ti.ClojureBaseName()),
			}

			buf := new(bytes.Buffer)
			Templates.ExecuteTemplate(buf, "clojure-typedef.tmpl", info)

			if n, err := out.Write(buf.Bytes()); err != nil {
				panic(fmt.Sprintf("n=%d err=%s", n, err))
			}
		})
}

func outputGoCode(ns string, v CodeInfo, clojureLibDir string, generateEmpty bool) {
	nsi := NamespacesInfo[ns]

	if !generateEmpty && !nsi.NonEmpty {
		return
	}

	nsi.HasGoFiles = true
	pkgBaseName := path.Base(nsi.Package)
	pkgDirNative := filepath.FromSlash(nsi.Package)

	var out *bufio.Writer
	var unbuf_out *os.File

	gf := filepath.Join(clojureLibDir, pkgDirNative,
		pkgBaseName+"_native.go")
	var e error
	e = os.MkdirAll(filepath.Dir(gf), 0777)
	Check(e)
	unbuf_out, e = os.Create(gf)
	Check(e)
	out = bufio.NewWriterSize(unbuf_out, 16384)

	defer func() {
		out.Flush()
		unbuf_out.Close()
	}()

	// First, figure out what other packages need to be imported,
	// before the import statement is generated.
	ensure := ""
	imports.SortedOriginalPackageImports(PackagesInfo[nsi.Package].Pkg,
		LegitimateImport,
		func(imp string, pos token.Pos) {
			thisNs := ClojureNamespaceForDirname(imp)
			if ns == thisNs {
				return // it me
			}

			nsi.ImportsNative.InternPackage(ClojureCoreDir, "", "", pos)

			ensure += fmt.Sprintf("\tEnsureLoaded(\"%s\")  // E.g. from: %s\n", thisNs, WhereAt(pos))
		})

	if out != nil {
		info := map[string]string{
			"PackageName": pkgBaseName,
			"Imports":     nsi.ImportsNative.QuotedList("\n\t"),
		}

		buf := new(bytes.Buffer)
		Templates.ExecuteTemplate(buf, "go-code.tmpl", info)

		if n, err := out.Write(buf.Bytes()); err != nil {
			panic(fmt.Sprintf("n=%d err=%s", n, err))
		}
	}

	SortedTypeInfoMap(v.Types,
		func(t string, ti TypeInfo) {
			if !ti.IsCustom() {
				return
			}
			ctor := ""
			if c, found := Ctors[ti]; found {
				ctor = c
			}
			if t == "crypto.Hash" {
				// fmt.Fprintf(os.Stdout, "output.go: %s aka %s @%p: %+v\n", t, ti.ClojureName(), ti, ti)
			}
			out.WriteString(GoCodeForType[ti])
			out.WriteString(ctor)
		})

	SortedCodeMap(v,
		func(f string, w *FnCodeInfo) {
			out.WriteString(w.FnCode)
		})

	SortedTypeDefinitions(v.InitTypes,
		func(ti TypeInfo) {
			tmn := ti.TypeMappingsName()
			if tmn == "" || !ti.IsExported() || ti.IsArbitraryType() {
				return
			}
			tmn = fmt.Sprintf("var %s Type\n", tmn)
			out.WriteString(tmn)
		})

	if out != nil {
		out.WriteString("\nfunc initNative() {\n")
	}

	SortedTypeDefinitions(v.InitTypes,
		func(ti TypeInfo) {
			tmn := ti.TypeMappingsName()
			if tmn == "" || !ti.IsExported() || ti.IsArbitraryType() {
				return
			}
			ctor := "nil"
			if c, found := CtorNames[ti]; found {
				ctor = c
			}
			mem := ""
			SortedFnCodeInfo(v.InitVars[ti], // Will always be populated
				func(c string, r *FnCodeInfo) {
					doc := r.FnDoc
					g := r.FnCode
					mem += fmt.Sprintf(`
			"%s": MakeGoReceiver("%s", %s, %s, %s, NewVectorFrom(%s)),
`[1:],
						c, c, g, strconv.Quote(genutils.CommentGroupAsString(doc)), strconv.Quote("1.0"), paramsAsSymbolVec(r.Params))
				})

			info := map[string]string{
				"GoName":      tmn,
				"ClojureName": ti.ClojureName(),
				"Ctor":        ctor,
				"Members":     mem,
			}

			buf := new(bytes.Buffer)
			Templates.ExecuteTemplate(buf, "go-func-init.tmpl", info)

			if n, err := out.Write(buf.Bytes()); err != nil {
				panic(fmt.Sprintf("n=%d err=%s", n, err))
			}
		})

	SortedTypeDefinitions(v.InitTypes,
		func(ti TypeInfo) {
			tmn := ti.TypeMappingsName()
			if tmn == "" || !ti.IsExported() || ti.IsArbitraryType() || !ti.IsSwitchable() {
				return
			}
			o := fmt.Sprintf("\tGoTypesVec[%d] = &%s\n", Ordinal[ti], tmn)
			out.WriteString(o)
		})

	if ensure != "" {
		out.WriteString(ensure)
	}

	out.WriteString("}\n")
}

func OutputNamespaces(clojureLibDir string, generateEmpty bool) {
	SortedNamespaceMap(ClojureCode,
		func(ns string, v CodeInfo) {
			outputClojureCode(ns, v, clojureLibDir, generateEmpty)
		})

	SortedNamespaceMap(GoCode,
		func(ns string, v CodeInfo) {
			outputGoCode(ns, v, clojureLibDir, generateEmpty)
		})
}

func init() {
	TemplatesFuncMap["curtimeandversion"] = curTimeAndVersion
}
