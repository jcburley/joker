package main

import (
	"fmt"
	"github.com/candid82/joker/tools/gostd/abends"
	. "github.com/candid82/joker/tools/gostd/gowalk"
	"github.com/candid82/joker/tools/gostd/types"
	. "github.com/candid82/joker/tools/gostd/utils"
	"go/build"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const VERSION = "0.1"

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

var goSourcePath string

func notOption(arg string) bool {
	return arg == "-" || !strings.HasPrefix(arg, "-")
}

func usage() {
	fmt.Print(`
Usage: gostd options...

Options:
  --go <go-source-dir>        # Location of Golang's src/ subdirectory (defaults to build.Default.GOROOT)
  --others <package-spec>...  # Location of another package directory, or a file with one <package-spec> per line
  --go-source-path <path>     # Overrides $GOPATH/src/ as "root" of <package-spec> specifications
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
  --undo                      # Undo effects of --joker ...
  --help, -h                  # Print this information

If <joker-std-subdir> is not specified, no Go nor Clojure source files
(nor any other files nor directories) are created, effecting a sort of
"dry run".
`)
	os.Exit(0)
}

func listOfOthers(other string) (others []string) {
	o := filepath.Join(goSourcePath, other)
	s, e := os.Stat(o)
	if e != nil {
		o = other // try original without $GOPATH/src/ prefix
		s, e = os.Stat(o)
	}
	Check(e)
	if s.IsDir() {
		return []string{o}
	}
	panic(fmt.Sprintf("files not yet supported: %s", other))
}

func main() {
	Fset = token.NewFileSet() // positions are relative to Fset
	Dump = false

	length := len(os.Args)
	goSourceDir := ""
	goSourceDirVia := ""
	goSourcePath = os.Getenv("GOPATH")
	var others []string
	var otherSourceDirs []string
	jokerSourceDir := ""
	replace := false
	overwrite := false
	summary := false
	generateEmpty := false
	outputCode := false
	undo := false

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
				Dump = true
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
				Verbose = true
			case "--summary":
				summary = true
			case "--output-code":
				outputCode = true
			case "--empty":
				generateEmpty = true
			case "--undo":
				undo = true
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
			case "--others":
				if i >= length-1 || !notOption(os.Args[i+1]) {
					panic("missing package-spec(s) after --others option")
				}
				for i < length-1 && notOption(os.Args[i+1]) {
					i += 1 // shift
					others = append(others, os.Args[i])
				}
			case "--go-source-path":
				if i < length-1 && notOption(os.Args[i+1]) {
					i += 1 // shift
					goSourcePath = os.Args[i]
				} else {
					panic("missing package-spec(s) after --go-source-path option")
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

	if Verbose {
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

	if goSourcePath == "" {
		panic("no Go source path defined via either $GOPATH or --go-source-path")
	}
	if fi, e := os.Stat(goSourcePath); e == nil && fi.IsDir() && filepath.Base(goSourcePath) != "src" {
		goSourcePath = filepath.Join(goSourcePath, "src")
	}

	if Verbose {
		fmt.Printf("goSourceDir: %s\n", goSourceDir)
		fmt.Printf("goSourcePath: %s\n", goSourcePath)
		for _, o := range others {
			otherSourceDirs = append(otherSourceDirs, listOfOthers(o)...)
		}
		for _, o := range otherSourceDirs {
			fmt.Printf("Other: %s\n", o)
		}
	}

	jokerLibDir := ""
	if jokerSourceDir != "" && jokerSourceDir != "-" {
		jokerLibDir = filepath.Join(jokerSourceDir, "std", "go", "std")
		if replace || undo {
			if e := os.RemoveAll(jokerLibDir); e != nil {
				panic(fmt.Sprintf("Unable to effectively 'rm -fr %s'", jokerLibDir))
			}
		}

		if undo {
			RegisterPackages([]string{}, jokerSourceDir)
			RegisterJokerFiles([]string{}, jokerSourceDir)
			RegisterGoTypeSwitch([]*types.TypeDefInfo{}, jokerSourceDir)
			os.Exit(0)
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

	AddMapping(goSourceDir, "go.std.")
	root := filepath.Join(goSourceDir, ".")
	AddWalkDir(goSourceDir, root, "go.std.")

	for _, o := range otherSourceDirs {
		root := filepath.Join(o, ".")
		AddWalkDir(o, root, "x.y.z.")
	}

	err, badDir := WalkAllDirs()
	if err != nil {
		panic("Error walking directory " + badDir + ": " + fmt.Sprintf("%v", err))
	}

	SortedConstantInfoMap(GoConstants,
		func(c string, ci *ConstantInfo) {
			GenConstant(ci)
		})

	SortedVariableInfoMap(GoVariables,
		func(c string, ci *VariableInfo) {
			GenVariable(ci)
		})

	/* Generate function-code snippets in alphabetical order. */
	SortedFuncInfoMap(QualifiedFunctions,
		func(f string, v *FuncInfo) {
			if v.Fd.Recv == nil {
				GenStandalone(v)
			} else {
				GenReceiver(v)
			}
		})

	/* Generate type-code snippets in sorted order. For each
	/* package, types are generated only if at least one function
	/* is generated (above) -- so genFunction() must be called for
	/* all functions beforehand. */
	SortedTypeInfoMap(GoTypes,
		func(t string, ti *GoTypeInfo) {
			if ti.Td != nil {
				GenType(t, ti)
			}
		})

	OutputPackageCode(jokerLibDir, outputCode, generateEmpty)

	if jokerSourceDir != "" && jokerSourceDir != "-" {
		var packagesArray = []string{} // Relative package pathnames in alphabetical order
		var dotJokeArray = []string{}  // Relative package pathnames in alphabetical order

		SortedPackagesInfo(PackagesInfo,
			func(p string, i *PackageInfo) {
				if !generateEmpty && !i.NonEmpty {
					return
				}
				if i.HasGoFiles {
					packagesArray = append(packagesArray, p)
				}
				dotJokeArray = append(dotJokeArray, p)
			})
		RegisterPackages(packagesArray, jokerSourceDir)
		RegisterJokerFiles(dotJokeArray, jokerSourceDir)
		RegisterGoTypeSwitch(types.AllTypesSorted(), jokerSourceDir)
	}

	if Verbose || summary {
		fmt.Printf("ABENDs:")
		abends.PrintAbends()
		fmt.Printf(`
Totals: functions=%d generated=%d (%s%%)
          non-receivers=%d (%s%%) generated=%d (%s%%)
          receivers=%d (%s%%) generated=%d (%s%%)
        types=%d generated=%d (%s%%)
          hits expr=%d alias=%d fullname=%d
        constants=%d generated=%d (%s%%)
        variables=%d generated=%d (%s%%)
`,
			NumFunctions, NumGeneratedFunctions, pct(NumGeneratedFunctions, NumFunctions),
			NumStandalones, pct(NumStandalones, NumFunctions), NumGeneratedStandalones, pct(NumGeneratedStandalones, NumStandalones),
			NumReceivers, pct(NumReceivers, NumFunctions), NumGeneratedReceivers, pct(NumGeneratedReceivers, NumReceivers),
			NumTypes, NumGeneratedTypes, pct(NumGeneratedTypes, NumTypes),
			types.NumExprHits, types.NumAliasHits, types.NumFullNameHits,
			NumConstants, NumGeneratedConstants, pct(NumGeneratedConstants, NumConstants),
			NumVariables, NumGeneratedVariables, pct(NumGeneratedVariables, NumVariables))
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
}
