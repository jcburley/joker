package main

import (
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"os"
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

func notOption(arg string) bool {
	return arg == "-" || !strings.HasPrefix(arg, "-")
}

func usage() {
	fmt.Print(`
Usage: gostd options...

Options:
  --go <go-source-dir>        # Location of Golang's src/ subdirectory (defaults to build.Default.GOROOT)
  --overwrite                 # Overwrite any existing <joker-std-subdir> files, leaving existing files intact
  --replace                   # 'rm -fr <joker-std-subdir>' before creating <joker-std-subdir>
  --fresh                     # (Default) Refuse to overwrite existing <joker-std-subdir> directory
  --joker <joker-source-dir>  # Modify pertinent source files to reflect packages being created
  --verbose, -v               # Print info on what's going on
  --summary                   # Print summary of #s of types, functions, etc.
  --output-code               # Print generated code to stdout (used by test.sh)
  --empty                     # Generate empty packages (those with no Joker code)
  --dump                      # Use go's AST dump API on pertinent elements (functions, types, etc.)
  --no-timestamp              # Don't put the time (and version) info in generated/modified files
  --help, -h                  # Print this information

If <joker-std-subdir> is not specified, no Go nor Clojure source files
(nor any other files nor directories) are created, effecting a sort of
"dry run".
`)
	os.Exit(0)
}

func main() {
	fset = token.NewFileSet() // positions are relative to fset
	dump = false

	length := len(os.Args)
	goSourceDir := ""
	goSourceDirVia := ""
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
				if goSourceDir != "" {
					panic("cannot specify --go <go-source-dir> more than once")
				}
				if i < length-1 && notOption(os.Args[i+1]) {
					i += 1 // shift
					goSourceDir = os.Args[i]
					goSourceDirVia = "--go"
				} else {
					panic("missing path after --go option")
				}
			case "--joker":
				if jokerSourceDir != "" {
					panic("cannot specify --joker <joker-source-dir> more than once")
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

	if goSourceDir == "" {
		goSourceDir = build.Default.GOROOT
		goSourceDirVia = "build.Default.GOROOT"
	}

	goSourceDir = filepath.Join(goSourceDir, "src")

	if fi, e := os.Stat(filepath.Join(goSourceDir, "go")); e != nil || !fi.IsDir() {
		if m, e := filepath.Glob(filepath.Join(goSourceDir, "*.go")); e != nil || m == nil || len(m) == 0 {
			fmt.Fprintf(os.Stderr, "Does not exist or is not a Go source directory: %s (specified via %s);\n%v",
				goSourceDir, goSourceDirVia, m)
			os.Exit(2)
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

	root := filepath.Join(goSourceDir, ".")
	err := walkDirs(root, mode)
	if err != nil {
		panic("Error walking directory " + goSourceDir + ": " + fmt.Sprintf("%v", err))
	}

	sort.Strings(alreadySeen)
	for _, a := range alreadySeen {
		fmt.Fprintln(os.Stderr, a)
	}

	/* Generate function-code snippets in alphabetical order. */
	sortedFuncInfoMap(qualifiedFunctions,
		func(f string, v *funcInfo) {
			genFunction(v)
		})

	/* Generate type-code snippets in sorted order. For each
	/* package, types are generated only if at least one function
	/* is generated (above) -- so genFunction() must be called for
	/* all functions beforehand. */
	sortedTypeInfoMap(goTypes,
		func(t string, ti *goTypeInfo) {
			if ti.td != nil {
				genType(t, ti)
			}
		})

	outputPackageCode(jokerLibDir, outputCode, generateEmpty)

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
	}

	if verbose || summary {
		fmt.Printf("ABENDs:")
		printAbends(abends)
		fmt.Printf(`
Totals: functions=%d methods=%d (%s%%) standalone=%d (%s%%) generated=%d (%s%%)
        types=%d generated=%d (%s%%)
`,
			len(qualifiedFunctions)+numMethods, numMethods,
			pct(numMethods, len(qualifiedFunctions)+numMethods),
			len(qualifiedFunctions), pct(len(qualifiedFunctions), len(qualifiedFunctions)+numMethods),
			numGeneratedFunctions, pct(numGeneratedFunctions, len(qualifiedFunctions)),
			numDeclaredGoTypes, numGeneratedTypes, pct(numGeneratedTypes, numDeclaredGoTypes))
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
