package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
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

func outputClojureCode(pkgDirUnix string, v codeInfo, jokerLibDir string, outputCode, generateEmpty bool) {
	var out *bufio.Writer
	var unbuf_out *os.File

	if jokerLibDir != "" && jokerLibDir != "-" &&
		(generateEmpty || packagesInfo[pkgDirUnix].nonEmpty) {
		jf := filepath.Join(jokerLibDir, filepath.FromSlash(pkgDirUnix)+".joke")
		var e error
		e = os.MkdirAll(filepath.Dir(jf), 0777)
		unbuf_out, e = os.Create(jf)
		check(e)
	} else if generateEmpty || packagesInfo[pkgDirUnix].nonEmpty {
		unbuf_out = os.Stdout
	}
	if unbuf_out != nil {
		out = bufio.NewWriterSize(unbuf_out, 16384)
	}

	pi := packagesInfo[pkgDirUnix]

	if out != nil {
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

	sortedTypeInfoMap(v.types,
		func(t string, ti *typeInfo) {
			if outputCode {
				fmt.Printf("JOKER TYPE %s from %s:%s\n", t, ti.sourceFile.name, ti.clojureCode)
			}
			if out != nil && unbuf_out != os.Stdout {
				out.WriteString(ti.clojureCode)
			}
		})

	sortedCodeMap(v,
		func(f string, w fnCodeInfo) {
			if outputCode {
				fmt.Printf("JOKER FUNC %s.%s from %s:%s\n",
					pkgDirUnix, f, w.sourceFile.name, w.fnCode)
			}
			if out != nil && unbuf_out != os.Stdout {
				out.WriteString(w.fnCode)
			}
		})

	if out != nil {
		out.Flush()
		if unbuf_out != os.Stdout {
			unbuf_out.Close()
		}
	}
}

func outputGoCode(pkgDirUnix string, v codeInfo, jokerLibDir string, outputCode, generateEmpty bool) {
	pkgBaseName := path.Base(pkgDirUnix)
	pi := packagesInfo[pkgDirUnix]
	packagesInfo[pkgDirUnix].hasGoFiles = true
	pkgDirNative := filepath.FromSlash(pkgDirUnix)

	var out *bufio.Writer
	var unbuf_out *os.File

	if jokerLibDir != "" && jokerLibDir != "-" &&
		(generateEmpty || packagesInfo[pkgDirUnix].nonEmpty) {
		gf := filepath.Join(jokerLibDir, pkgDirNative,
			pkgBaseName+"_native.go")
		var e error
		e = os.MkdirAll(filepath.Dir(gf), 0777)
		check(e)
		unbuf_out, e = os.Create(gf)
		check(e)
	} else if generateEmpty || packagesInfo[pkgDirUnix].nonEmpty {
		unbuf_out = os.Stdout
	}
	if unbuf_out != nil {
		out = bufio.NewWriterSize(unbuf_out, 16384)
	}

	importCore := ""
	if _, f := pi.importsNative[pkgDirUnix]; f {
		importCore = `
	. "github.com/candid82/joker/core"`
	}

	if out != nil {
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

	sortedTypeInfoMap(v.types,
		func(t string, ti *typeInfo) {
			if outputCode {
				fmt.Printf("GO TYPE %s from %s:%s\n", t, ti.sourceFile.name, ti.goCode)
			}
			if out != nil && unbuf_out != os.Stdout {
				out.WriteString(ti.goCode)
			}
		})

	sortedCodeMap(v,
		func(f string, w fnCodeInfo) {
			if outputCode {
				fmt.Printf("GO FUNC %s.%s from %s:%s\n",
					pkgDirUnix, f, w.sourceFile.name, w.fnCode)
			}
			if out != nil && unbuf_out != os.Stdout {
				out.WriteString(w.fnCode)
			}
		})

	if out != nil {
		out.Flush()
		if unbuf_out != os.Stdout {
			unbuf_out.Close()
		}
	}
}

func outputPackageCode(jokerLibDir string, outputCode, generateEmpty bool) {
	sortedPackageMap(clojureCode,
		func(pkgDirUnix string, v codeInfo) {
			outputClojureCode(pkgDirUnix, v, jokerLibDir, outputCode, generateEmpty)
		})

	sortedPackageMap(goCode,
		func(pkgDirUnix string, v codeInfo) {
			outputGoCode(pkgDirUnix, v, jokerLibDir, outputCode, generateEmpty)
		})
}