package main

import (
	"bufio"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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
					!strings.Contains(e.Error(), "The system cannot find the")) { // Sometimes "...file specified"; other times "...path specified"!
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
				fmt.Printf("  %s\n", fileAt(ti.where))
			})
	}

	/* Generate function code snippets in alphabetical order, to stabilize test output in re unsupported types. */
	sortedFuncInfoMap(qualifiedFunctions,
		func(f string, v *funcInfo) {
			genFunction(v)
		})

	var out *bufio.Writer
	var unbuf_out *os.File

	sortedPackageMap(clojureCode,
		func(pkgDirUnix string, v codeInfo) {
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
				func(f string, w fnCodeInfo) {
					if outputCode {
						fmt.Printf("JOKER FUNC %s.%s from %s:%v\n",
							pkgDirUnix, f, w.sourceFile.name, w.fnCode)
					}
					if out != nil {
						out.WriteString(w.fnCode)
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
				func(f string, w fnCodeInfo) {
					if outputCode {
						fmt.Printf("GO FUNC %s.%s from %s:%v\n",
							pkgDirUnix, f, w.sourceFile.name, w.fnCode)
					}
					if out != nil {
						out.WriteString(w.fnCode)
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
