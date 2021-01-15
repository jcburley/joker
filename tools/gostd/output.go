package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/candid82/joker/tools/gostd/genutils"
	. "github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/imports"
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

func RegisterPackages(pkgs []string, clojureSourceDir string) {
	writeCustomLibsGo(pkgs, clojureSourceDir, "g_custom.go")
}

func RegisterClojureFiles(clojureFiles []string, clojureSourceDir string) {
	writeCustomLibsClojure(clojureFiles, clojureSourceDir, filepath.Join("core", "data", "g_customlibs.joke"))
}

func RegisterGoTypeSwitch(types []TypeInfo, clojureSourceDir string) {
	writeGoTypeSwitch(types, clojureSourceDir, filepath.Join("core", "g_goswitch.go"))
}

func writeCustomLibsGo(pkgs []string, dir, f string) {
	if Verbose {
		fmt.Printf("Adding %d custom imports to %s\n", len(pkgs), filepath.ToSlash(f))
	}

	newImports := ""
	importPrefix := "\t_ \"github.com/candid82/joker/std/go/std/"
	for _, p := range pkgs {
		newImports += importPrefix + p + "\"\n"
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "custom-libs-go.tmpl", newImports)

	if dir != "" {
		err := ioutil.WriteFile(filepath.Join(dir, f), buf.Bytes(), 0777)
		Check(err)
	}
}

func writeCustomLibsClojure(pkgs []string, dir, f string) {
	if Verbose {
		fmt.Printf("Adding %d custom loaded libraries to %s\n", len(pkgs), filepath.ToSlash(f))
	}

	m := ""
	const importPrefix = " 'go.std."
	for _, p := range pkgs {
		m += "    " + importPrefix + strings.ReplaceAll(p, "/", ".") + "\n"
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "custom-libs-clojure.tmpl", m)

	if dir != "" {
		err := ioutil.WriteFile(filepath.Join(dir, f), buf.Bytes(), 0777)
		Check(err)
	}
}

var Ordinal = map[TypeInfo]uint{}
var SwitchableTypes []TypeInfo // Set by GenTypeInfo() to subset of AllTypesSorted() that will go into the Go Type Switch

func writeGoTypeSwitch(allTypes []TypeInfo, dir, f string) {
	types := SwitchableTypes

	if Verbose {
		fmt.Printf("Adding only %d types (out of %d) to %s\n", len(types), len(allTypes), filepath.ToSlash(f))
	}

	var cases string
	var importeds = &imports.Imports{}
	for _, t := range types {
		if t.Specificity() == 0 {
			// These are empty interface{} types, and so really can't be specifically matched to anything.
			continue
		}
		pkgPlusSeparator := ""
		if t.GoPackage() != "" {
			pkgPlusSeparator = importeds.AddPackage(t.GoPackage(), "", "", true, token.NoPos) + "."
		}
		specificity := ""
		if t.Specificity() != ConcreteType {
			specificity = fmt.Sprintf("  // Specificity=%d", t.Specificity())
		}
		cases += fmt.Sprintf("\tcase %s:%s\n\t\treturn %d\n", fmt.Sprintf(t.GoPattern(), pkgPlusSeparator+t.GoBaseName()), specificity, Ordinal[t])
	}

	info := map[string]string{}
	info["Imports"] = importeds.QuotedList("\n\t")
	info["NumberOfTypes"] = strconv.Itoa(len(types))
	info["Cases"] = cases

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "go-type-switch.tmpl", info)

	if dir != "" {
		err := ioutil.WriteFile(filepath.Join(dir, f), buf.Bytes(), 0777)
		Check(err)
	}
}

func outputClojureCode(pkgDirUnix string, v CodeInfo, clojureLibDir string, generateEmpty bool) {
	pi := PackagesInfo[pkgDirUnix]

	if !generateEmpty && !pi.NonEmpty {
		return
	}

	var out *bufio.Writer
	var unbuf_out *os.File

	jf := filepath.Join(clojureLibDir, filepath.FromSlash(pkgDirUnix)+".joke")
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
		myDoc := doc.New(pi.Pkg, importPath, doc.AllDecls)
		pkgDoc := fmt.Sprintf("Provides a low-level interface to the %s package.", pkgDirUnix)
		if myDoc.Doc != "" {
			pkgDoc += "\n\n" + myDoc.Doc
		}

		info := map[string]interface{}{
			"Imports":   pi.ImportsAutoGen.AsClojureMap(),
			"Doc":       strconv.Quote(pkgDoc),
			"Empty":     !pi.NonEmpty,
			"Namespace": "go.std." + strings.ReplaceAll(pkgDirUnix, "/", "."),
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
			if !ti.Custom() {
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
			if !ti.Custom() {
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

func outputGoCode(pkgDirUnix string, v CodeInfo, clojureLibDir string, generateEmpty bool) {
	pi := PackagesInfo[pkgDirUnix]

	if !generateEmpty && !pi.NonEmpty {
		return
	}

	pi.HasGoFiles = true
	pkgBaseName := path.Base(pkgDirUnix)
	pkgDirNative := filepath.FromSlash(pkgDirUnix)

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
	imports.SortedOriginalPackageImports(pi.Pkg,
		LegitimateImport,
		func(imp string, pos token.Pos) {
			ns := ClojureNamespaceForDirname(imp)
			if ns == pi.ClojureNameSpace {
				return // it me
			}

			pi.ImportsNative.InternPackage(ClojureCoreDir, "", "", pos)

			ensure += fmt.Sprintf("\tEnsureLoaded(\"%s\")  // E.g. from: %s\n", ns, WhereAt(pos))
		})

	if out != nil {
		info := map[string]string{
			"PackageName": pkgBaseName,
			"Imports":     pi.ImportsNative.QuotedList("\n\t"),
		}

		buf := new(bytes.Buffer)
		Templates.ExecuteTemplate(buf, "go-code.tmpl", info)

		if n, err := out.Write(buf.Bytes()); err != nil {
			panic(fmt.Sprintf("n=%d err=%s", n, err))
		}
	}

	SortedTypeInfoMap(v.Types,
		func(t string, ti TypeInfo) {
			if !ti.Custom() {
				return
			}
			ctor := ""
			if c, found := Ctors[ti]; found && c[0] != '/' {
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
			tmn = fmt.Sprintf("var %s GoTypeInfo\n", tmn)
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
			k1 := ti.ClojureName()
			ctor := ""
			if c, found := CtorNames[ti]; found {
				ctor = fmt.Sprintf(`
		Ctor: %s,
`[1:],
					c)
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
				"ClojureName": k1,
				"Ctor":        ctor,
				"Members":     mem,
				"Type":        "", /*"Type:"..., but probably not needed*/
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
			if tmn == "" || !ti.IsExported() || ti.IsArbitraryType() {
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

func OutputPackageCode(clojureLibDir string, generateEmpty bool) {
	SortedPackageMap(ClojureCode,
		func(pkgDirUnix string, v CodeInfo) {
			outputClojureCode(pkgDirUnix, v, clojureLibDir, generateEmpty)
		})

	SortedPackageMap(GoCode,
		func(pkgDirUnix string, v CodeInfo) {
			outputGoCode(pkgDirUnix, v, clojureLibDir, generateEmpty)
		})
}

func init() {
	TemplatesFuncMap["curtimeandversion"] = curTimeAndVersion
}
