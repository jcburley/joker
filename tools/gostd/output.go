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

func RegisterPackages(pkgs []string, clojureSourceDir string, outputCode bool) {
	writeCustomLibsGo(pkgs, clojureSourceDir, "g_custom.go", outputCode)
}

func RegisterClojureFiles(clojureFiles []string, clojureSourceDir string, outputCode bool) {
	writeCustomLibsClojure(clojureFiles, clojureSourceDir, filepath.Join("core", "data", "g_customlibs.joke"), outputCode)
}

func RegisterGoTypeSwitch(types []TypeInfo, clojureSourceDir string, outputCode bool) {
	writeGoTypeSwitch(types, clojureSourceDir, filepath.Join("core", "g_goswitch.go"), outputCode)
}

func writeCustomLibsGo(pkgs []string, dir, f string, outputCode bool) {
	if Verbose {
		fmt.Printf("Adding %d custom imports to %s\n", len(pkgs), filepath.ToSlash(f))
	}

	newImports := ""
	importPrefix := "\t_ \"github.com/candid82/joker/std/go/std/"
	for _, p := range pkgs {
		newImports += importPrefix + p + "\"\n"
	}

	buf := new(bytes.Buffer)
	Templates.ExecuteTemplate(buf, "customlibs.tmpl", newImports)

	if dir != "" {
		err := ioutil.WriteFile(filepath.Join(dir, f), buf.Bytes(), 0777)
		Check(err)
	}

	if outputCode {
		fmt.Printf("\n-------- BEGIN Generated file %s:\n", f)
		fmt.Print(buf.String())
		fmt.Printf("-------- END generated file %s.\n\n", f)
	}
}

func writeCustomLibsClojure(pkgs []string, dir, f string, outputCode bool) {
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

	if outputCode {
		fmt.Printf("\n-------- BEGIN Generated file %s:\n", f)
		fmt.Print(buf.String())
		fmt.Printf("-------- END generated file %s.\n\n", f)
	}
}

var Ordinal = map[TypeInfo]uint{}
var SwitchableTypes []TypeInfo // Set by GenTypeInfo() to subset of AllTypesSorted() that will go into the Go Type Switch

func writeGoTypeSwitch(allTypes []TypeInfo, dir, f string, outputCode bool) {
	types := SwitchableTypes

	if Verbose {
		fmt.Printf("Adding only %d types (out of %d) to %s\n", len(types), len(allTypes), filepath.ToSlash(f))
	}

	pattern := "// Auto-modified by gostd at " + curTimeAndVersion() + `

package core

import (%s
)

var GoTypesVec [%d]*GoTypeInfo

func SwitchGoType(g interface{}) int {
	switch g.(type) {
%s	}
	return -1
}
`

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

	m := fmt.Sprintf(pattern, importeds.QuotedList("\n\t"), len(types), cases)

	if dir != "" {
		err := ioutil.WriteFile(filepath.Join(dir, f), []byte(m), 0777)
		Check(err)
	}

	if outputCode {
		fmt.Printf("\n-------- BEGIN generated file %s:\n", f)
		fmt.Print(m)
		fmt.Printf("-------- END generated file %s.\n\n", f)
	}
}

func outputClojureCode(pkgDirUnix string, v CodeInfo, clojureLibDir string, outputCode, generateEmpty bool) {
	var out *bufio.Writer
	var unbuf_out *os.File

	if clojureLibDir != "" && clojureLibDir != "-" &&
		(generateEmpty || PackagesInfo[pkgDirUnix].NonEmpty) {
		jf := filepath.Join(clojureLibDir, filepath.FromSlash(pkgDirUnix)+".joke")
		var e error
		e = os.MkdirAll(filepath.Dir(jf), 0777)
		unbuf_out, e = os.Create(jf)
		Check(e)
	} else if generateEmpty || PackagesInfo[pkgDirUnix].NonEmpty {
		unbuf_out = os.Stdout
	}
	if unbuf_out != nil {
		out = bufio.NewWriterSize(unbuf_out, 16384)
	}

	pi := PackagesInfo[pkgDirUnix]

	if out != nil {
		importPath, _ := filepath.Abs("/")
		myDoc := doc.New(pi.Pkg, importPath, doc.AllDecls)
		pkgDoc := fmt.Sprintf("Provides a low-level interface to the %s package.", pkgDirUnix)
		if myDoc.Doc != "" {
			pkgDoc += "\n\n" + myDoc.Doc
		}

		fmt.Fprintf(out,
			`;;;; Auto-generated by gostd at `+curTimeAndVersion()+`, do not edit!!

(ns
  ^{:go-imports {%s}
    :doc %s
    :empty %s}
  %s)
`,
			pi.ImportsAutoGen.AsClojureMap(),
			strconv.Quote(pkgDoc),
			func() string {
				if pi.NonEmpty {
					return "false"
				} else {
					return "true"
				}
			}(),
			"go.std."+strings.ReplaceAll(pkgDirUnix, "/", "."))
	}

	SortedConstantInfoMap(v.Constants,
		func(c string, ci *ConstantInfo) {
			if outputCode {
				fmt.Printf("CLOJURE CONSTANT %s from %s:%s\n", c, ci.SourceFile.Name, ci.Def)
			}
			if out != nil && unbuf_out != os.Stdout {
				out.WriteString(ci.Def)
			}
		})

	SortedVariableInfoMap(v.Variables,
		func(c string, ci *VariableInfo) {
			if outputCode {
				fmt.Printf("CLOJURE VARIABLE %s from %s:%s\n", c, ci.SourceFile.Name, ci.Def)
			}
			if out != nil && unbuf_out != os.Stdout {
				out.WriteString(ci.Def)
			}
		})

	SortedTypeInfoMap(v.Types,
		func(t string, ti TypeInfo) {
			if !ti.Custom() {
				return
			}
			if outputCode {
				fmt.Printf("CLOJURE TYPE %s from %s:%s\n", t, GoFilenameForTypeSpec(ti.UnderlyingTypeSpec()), ClojureCodeForType[ti])
			}
			if out != nil && unbuf_out != os.Stdout {
				out.WriteString(ClojureCodeForType[ti])
			}
		})

	SortedCodeMap(v,
		func(f string, w *FnCodeInfo) {
			if outputCode {
				fmt.Printf("CLOJURE FUNC %s.%s from %s:%s\n",
					pkgDirUnix, f, w.SourceFile.Name, w.FnCode)
			}
			if out != nil && unbuf_out != os.Stdout {
				out.WriteString(w.FnCode)
			}
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
			fnCode := fmt.Sprintf(`
(def
  ^{:doc %s
    :added "1.0"
    :tag "GoType"
%s    :go "&%s"}
  %s)
`,
				strconv.Quote(typeDoc), specificity, tmn, fmt.Sprintf(ti.ClojurePattern(), ti.ClojureBaseName()))
			if outputCode {
				fmt.Printf("CLOJURE TYPE %s:%s\n",
					ti.ClojureName(), fnCode)
			}
			if out != nil && unbuf_out != os.Stdout {
				out.WriteString(fnCode)
			}
		})

	if out != nil {
		out.Flush()
		if unbuf_out != os.Stdout {
			unbuf_out.Close()
		}
	}
}

func outputGoCode(pkgDirUnix string, v CodeInfo, clojureLibDir string, outputCode, generateEmpty bool) {
	pkgBaseName := path.Base(pkgDirUnix)
	pi := PackagesInfo[pkgDirUnix]
	pi.HasGoFiles = true
	pkgDirNative := filepath.FromSlash(pkgDirUnix)

	var out *bufio.Writer
	var unbuf_out *os.File

	if clojureLibDir != "" && clojureLibDir != "-" &&
		(generateEmpty || pi.NonEmpty) {
		gf := filepath.Join(clojureLibDir, pkgDirNative,
			pkgBaseName+"_native.go")
		var e error
		e = os.MkdirAll(filepath.Dir(gf), 0777)
		Check(e)
		unbuf_out, e = os.Create(gf)
		Check(e)
	} else if generateEmpty || pi.NonEmpty {
		unbuf_out = os.Stdout
	}
	if unbuf_out != nil {
		out = bufio.NewWriterSize(unbuf_out, 16384)
	}

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
		fmt.Fprintf(out,
			`// Auto-generated by gostd at `+curTimeAndVersion()+`, do not edit!!

package %s

import (%s
)
`,
			pkgBaseName,
			pi.ImportsNative.QuotedList("\n\t"))
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
			if outputCode {
				fmt.Printf("GO TYPE %s from %s:%s%s\n", t, GoFilenameForTypeSpec(ti.UnderlyingTypeSpec()), GoCodeForType[ti], ctor)
			}
			if t == "crypto.Hash" {
				// fmt.Printf("output.go: %s aka %s @%p: %+v\n", t, ti.ClojureName(), ti, ti)
			}
			if out != nil && unbuf_out != os.Stdout {
				out.WriteString(GoCodeForType[ti])
				out.WriteString(ctor)
			}
		})

	SortedCodeMap(v,
		func(f string, w *FnCodeInfo) {
			if outputCode {
				fmt.Printf("GO FUNC %s.%s from %s:%s\n",
					pkgDirUnix, f, w.SourceFile.Name, w.FnCode)
			}
			if out != nil && unbuf_out != os.Stdout {
				out.WriteString(w.FnCode)
			}
		})

	SortedTypeDefinitions(v.InitTypes,
		func(ti TypeInfo) {
			tmn := ti.TypeMappingsName()
			if tmn == "" || !ti.IsExported() || ti.IsArbitraryType() {
				return
			}
			tmn = fmt.Sprintf("var %s GoTypeInfo\n", tmn)
			if outputCode && tmn != "" {
				fmt.Printf("GO VARDEF FOR TYPE %s from %s:\n%s\n", ti.ClojureName(), WhereAt(ti.DefPos()), tmn)
			}
			if out != nil && unbuf_out != os.Stdout && tmn != "" {
				out.WriteString(tmn)
			}
		})

	const initInfoTemplate = `
	%s = GoTypeInfo{Name: "%s",
		GoType: &GoType{T: &%s},
%s		Members: GoMembers{
%s		},
%s	}

`

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
			o := fmt.Sprintf(initInfoTemplate[1:], tmn, k1, tmn, ctor, mem, "" /*"Type:"..., but probably not needed*/)
			if outputCode {
				fmt.Printf("GO INFO FOR TYPE %s from %s:\n%s\n", ti.ClojureName(), WhereAt(ti.DefPos()), o)
			}
			if out != nil && unbuf_out != os.Stdout {
				out.WriteString(o)
			}
		})

	SortedTypeDefinitions(v.InitTypes,
		func(ti TypeInfo) {
			tmn := ti.TypeMappingsName()
			if tmn == "" || !ti.IsExported() || ti.IsArbitraryType() {
				return
			}
			o := fmt.Sprintf("\tGoTypesVec[%d] = &%s\n", Ordinal[ti], tmn)
			if outputCode {
				fmt.Printf("GO VECSET FOR TYPE %s from %s:\n%s\n", ti.ClojureName(), WhereAt(ti.DefPos()), o)
			}
			if out != nil && unbuf_out != os.Stdout {
				out.WriteString(o)
			}
		})

	if ensure != "" {
		if outputCode {
			fmt.Printf("GO ENSURE-LOADED FOR %s:\n%s\n", pi.Pkg.Name, ensure)
		}
		if out != nil && unbuf_out != os.Stdout {
			out.WriteString(ensure)
		}
	}

	if out != nil {
		out.WriteString("}\n")
		if unbuf_out == os.Stdout {
			out.WriteString("\n") // separate from next "file" output for testing
		}
	}

	if out != nil {
		out.Flush()
		if unbuf_out != os.Stdout {
			unbuf_out.Close()
		}
	}
}

func OutputPackageCode(clojureLibDir string, outputCode, generateEmpty bool) {
	SortedPackageMap(ClojureCode,
		func(pkgDirUnix string, v CodeInfo) {
			outputClojureCode(pkgDirUnix, v, clojureLibDir, outputCode, generateEmpty)
		})

	SortedPackageMap(GoCode,
		func(pkgDirUnix string, v CodeInfo) {
			outputGoCode(pkgDirUnix, v, clojureLibDir, outputCode, generateEmpty)
		})
}

func init() {
	TemplatesFuncMap["curtimeandversion"] = curTimeAndVersion
}
