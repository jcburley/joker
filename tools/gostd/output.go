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
	"go/types"
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

func RegisterPackages(pkgs []string, clojureSourceDir paths.NativePath) {
	writeCustomLibsGo(pkgs, clojureSourceDir, paths.NewNativePath("g_custom.go"))
}

func RegisterClojureFiles(clojureFiles []string, clojureSourceDir paths.NativePath) {
	writeCustomLibsClojure(clojureFiles, clojureSourceDir, paths.NewNativePath(filepath.Join("core", "data", "g_customlibs.joke")))
}

func RegisterGoTypeSwitch(types []TypeInfo, clojureSourceDir paths.NativePath) {
	writeGoTypeSwitch(types, clojureSourceDir, paths.NewNativePath(filepath.Join("core", "g_goswitch.go")))
}

func writeCustomLibsGo(pkgs []string, dir, f paths.NativePath) {
	if Verbose {
		fmt.Printf("Adding %d custom imports to %s\n", len(pkgs), f)
	}

	newImports := ""
	importPrefix := "\t_ "
	for _, p := range pkgs {
		newImports += importPrefix + fmt.Sprintf("%q\n", generatedPkgPrefix+goStdPrefix.String()+p)
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "go-custom-libs.tmpl", newImports)

	if dir.String() != "" {
		err := ioutil.WriteFile(dir.Join(f.String()).ToNative().String(), buf.Bytes(), 0777)
		Check(err)
	}
}

func writeCustomLibsClojure(pkgs []string, dir, f paths.NativePath) {
	if Verbose {
		fmt.Printf("Adding %d custom loaded libraries to %s\n", len(pkgs), f)
	}

	m := ""
	var importPrefix = " '" + goNsPrefix
	for _, p := range pkgs {
		m += "    " + importPrefix + strings.ReplaceAll(p, "/", ".") + "\n"
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "clojure-custom-libs.tmpl", m)

	if dir.String() != "" {
		err := ioutil.WriteFile(dir.Join(f.String()).ToNative().String(), buf.Bytes(), 0777)
		Check(err)
	}
}

var Ordinal = map[TypeInfo]int{}
var SwitchableTypes []TypeInfo // Set by GenTypeInfo() to subset of AllTypesSorted() that will go into the Go Type Switch

func writeGoTypeSwitch(allTypes []TypeInfo, dir, f paths.NativePath) {
	types := SwitchableTypes

	if Verbose {
		fmt.Printf("Adding only %d types (out of %d) to %s\n", len(types), len(allTypes), f)
	}

	var cases []map[string]interface{}
	var importeds = &imports.Imports{For: "writeGoTypeSwitch"}
	for _, t := range types {
		specificity := t.Specificity()
		if specificity == 0 {
			// These are empty interface{} types, and so really can't be specifically matched to anything.
			continue
		}
		pkgPlusSeparator := ""
		if t.GoPackage() != "" {
			pkgPlusSeparator = importeds.AddPackage(t.GoPackage(), "", true, token.NoPos, "output.go/writeGoTypeSwitch") + "."
		}
		if Ordinal[t] == 0 {
			fmt.Fprintf(os.Stderr, "output.go/writeGoTypeSwitch: ERROR: No ordinal assigned to %s @%p\n", t, t)
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
			"ordinal": Ordinal[t] - 1,
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
			"Namespace": goNsPrefix + strings.ReplaceAll(pkgDirUnix, "/", "."),
		}

		buf := new(bytes.Buffer)
		Templates.ExecuteTemplate(buf, "clojure-code.tmpl", info)

		if n, err := out.Write(buf.Bytes()); err != nil {
			panic(fmt.Sprintf("n=%d err=%s", n, err))
		}
	}

	SortedConstantInfoMap(v.Constants,
		func(_ string, ci *ConstantInfo) {
			out.WriteString(ci.Def)
		})

	SortedVariableInfoMap(v.Variables,
		func(_ string, ci *VariableInfo) {
			out.WriteString(ci.Def)
		})

	SortedTypeInfoMap(v.Types,
		func(_ string, ti TypeInfo) {
			if !ti.IsCustom() {
				return
			}
			out.WriteString(ClojureCodeForType[ti])
		})

	SortedCodeMap(v,
		func(_ string, w *FnCodeInfo) {
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
			if ns == pi.Namespace {
				return // it me
			}

			pi.ImportsNative.InternPackage(ClojureCorePath, "", pos)

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
		func(_ string, w *FnCodeInfo) {
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
			if Ordinal[ti] == 0 {
				fmt.Fprintf(os.Stderr, "output.go/outputGoCode: ERROR: No ordinal assigned to %s @%p\n", ti, ti)
			} else {
				o := fmt.Sprintf("\tGoTypesVec[%d] = &%s\n", Ordinal[ti]-1, tmn)
				out.WriteString(o)
			}
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

func paramsAsSymbolVec(t *types.Tuple) string {
	genutils.GenSymReset()
	args := t.Len()
	var syms []string
	for argNum := 0; argNum < args; argNum++ {
		field := t.At(argNum)
		var p string
		if field.Name() == "" {
			p = genutils.GenSym("arg")
		} else {
			p = field.Name()
		}
		syms = append(syms, "MakeSymbol("+strconv.Quote(p)+")")
	}
	return strings.Join(syms, ", ")
}

func init() {
	TemplatesFuncMap["curtimeandversion"] = curTimeAndVersion
}
