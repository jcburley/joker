package main

import (
	"encoding/gob"
	"fmt"
	"github.com/candid82/joker/tools/gostd/abends"
	"github.com/candid82/joker/tools/gostd/godb"
	"github.com/candid82/joker/tools/gostd/paths"
	"go/build"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
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

func Check(err error) {
	if err != nil {
		panic(err)
	}
}

var goPath paths.NativePath
var importStdRoot = paths.NewUnixPath(path.Join("std", "gostd")) // Relative to --output dir.

var goStdPrefix = paths.NewUnixPath("go/std/")

var goNsPrefix = strings.ReplaceAll(goStdPrefix.String(), "/", ".")

var generatedPkgPrefix string

func notOption(arg string) bool {
	return arg == "--" || arg == "-" || !strings.HasPrefix(arg, "-")
}

func usage() {
	fmt.Print(`
Usage: gostd options...

Options:
  --go-path <gopath-dir>      # Overrides $GOPATH as "root" of <package-spec> specifications
  --go-root <goroot-dir>      # Location of Golang's src/ subdirectory (defaults to build.Default.GOROOT)
  --others <package-spec>...  # Location of other package directories, or a file with one <package-spec> per line
  --output <new-code-dir>     # Modify pertinent source files here to reflect packages being created (default: ".")
  --joker <joker-dir>         # Where to find the core/ directory with the APIs that generated Joker code may thus call (default: <new-code-dir>)
  --overwrite                 # Overwrite any existing <new-code-dir> files, leaving existing files intact
  --replace                   # 'rm -fr <new-code-dir>/std/gostd' before creating <new-code-dir>
  --fresh                     # (Default) Refuse to overwrite existing <new-code-dir> directory
  --import-from <dir>         # Override <joker-dir> with <dir> for use in generated import decls (default: <joker-dir>)
  --verbose, -v               # Print info on what's going on
  --summary                   # Print summary of #s of types, functions, etc.
  --empty                     # Generate empty packages (those with no Clojure code)
  --dump                      # Use go's AST dump API on pertinent elements (functions, types, etc.)
  --no-timestamp              # Don't put the time (and version) info in generated/modified files
  --help, -h                  # Print this information
`)
	os.Exit(0)
}

func listOfOthers(other paths.NativePath) (others []paths.NativePath) {
	o := goPath.JoinPaths(other).String()
	s, e := os.Stat(o)
	if e != nil {
		o = other.String() // try original without $GOPATH/src/ prefix
		s, e = os.Stat(o)
	}
	Check(e)
	if s.IsDir() {
		return []paths.NativePath{paths.NewNativePath(o)}
	}
	fmt.Fprintf(os.Stderr, "files not yet supported: %s\n", other)
	os.Exit(3)
	return
}

var coreApiFilename = "core-apis.dat"
var definedApis = map[string]struct{}{}

func readCoreApiFile(src paths.NativePath) {
	start := getCPU()
	defer func() {
		end := getCPU()
		if godb.Verbose && !noTimeAndVersion && start != 0 && end != 0 {
			fmt.Printf("readCoreApiFile() took %d ns.\n", end-start)
		}
	}()

	f, err := os.Open(coreApiFilename)
	if err != nil {
		if godb.Verbose {
			fmt.Printf("The list of core APIs is missing; file '%s' does not exist.\n", coreApiFilename)
		}

		coreDir := src.Join("core")
		definedApis = findApis(coreDir)
		if len(definedApis) == 0 {
			panic(fmt.Sprintf("no APIs found at %s", coreDir))
		}

		if godb.Verbose {
			fmt.Printf("Writing Core APIs found at %s to %s.\n", coreDir, coreApiFilename)
		}
		f, err = os.Create(coreApiFilename)
		Check(err)
		enc := gob.NewEncoder(f)
		err = enc.Encode(definedApis)
		Check(err)
		return
	}
	Check(err)
	dec := gob.NewDecoder(f)
	err = dec.Decode(&definedApis)
	Check(err)
	//	fmt.Printf("Core APIs: %+v\n", definedApis)
}

func NewDefinedApi(api, src string) {
	definedApis[api] = struct{}{}
	//	fmt.Printf("%s: Defined API '%s'\n", src, api)
}

var Templates *template.Template
var TemplatesFuncMap = template.FuncMap{}

func main() {
	godb.Fset = token.NewFileSet() // positions are relative to Fset

	length := len(os.Args)
	var goRoot, outputDir, clojureImportDir, jokerSourceDir paths.NativePath
	goRootVia := ""
	GOPATH := os.Getenv("GOPATH")
	goPath = paths.NewNativePath(filepath.FromSlash(GOPATH))
	var others []paths.NativePath
	var otherSourceDirs []paths.NativePath
	replace := false
	overwrite := false
	summary := false
	generateEmpty := false

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
				godb.Dump = true
				WalkDump = true
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
				godb.Verbose = true
			case "--summary":
				summary = true
			case "--empty":
				generateEmpty = true
			case "--go-root":
				if i < length-1 && notOption(os.Args[i+1]) {
					i += 1 // shift
					goRoot = paths.NewPathAsNative(os.Args[i])
					goRootVia = "--go-root"
				} else {
					fmt.Fprintf(os.Stderr, "missing path after --go-root option\n")
					os.Exit(1)
				}
			case "--others":
				if i >= length-1 || !notOption(os.Args[i+1]) {
					fmt.Fprintf(os.Stderr, "missing package-spec(s) after --others option\n")
					os.Exit(1)
				}
				for i < length-1 && notOption(os.Args[i+1]) {
					i += 1 // shift
					others = append(others, paths.NewPathAsNative(os.Args[i]))
				}
			case "--go-path":
				if i < length-1 && notOption(os.Args[i+1]) {
					i += 1 // shift
					goPath = paths.NewPathAsNative(os.Args[i])
				} else {
					fmt.Fprintf(os.Stderr, "missing path after --go-path option\n")
					os.Exit(1)
				}
			case "--output":
				if i < length-1 && notOption(os.Args[i+1]) {
					i += 1 // shift
					outputDir = paths.NewPathAsNative(os.Args[i])
				} else {
					fmt.Fprintf(os.Stderr, "missing path after --output option\n")
					os.Exit(1)
				}
			case "--import-from":
				if i < length-1 && notOption(os.Args[i+1]) {
					i += 1 // shift
					clojureImportDir = paths.NewPathAsNative(os.Args[i])
				} else {
					fmt.Fprintf(os.Stderr, "missing path after --import-from option; got %s, which looks like an option\n", os.Args[i+1])
					os.Exit(1)
				}
			case "--joker":
				if i < length-1 && notOption(os.Args[i+1]) {
					i += 1 // shift
					jokerSourceDir = paths.NewPathAsNative(os.Args[i])
				} else {
					fmt.Fprintf(os.Stderr, "missing path after --joker option; got %s, which looks like an option\n", os.Args[i+1])
					os.Exit(1)
				}
			default:
				fmt.Fprintf(os.Stderr, "unrecognized option %s\n", a)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "extraneous argument(s) starting with: %s\n", a)
			os.Exit(1)
		}
	}

	if godb.Verbose {
		fmt.Printf("Default context: %v\n", build.Default)
	}

	if jokerSourceDir.IsEmpty() {
		jokerSourceDir = outputDir
	}

	if clojureImportDir.IsEmpty() {
		clojureImportDir = jokerSourceDir
	}

	if goRoot.IsEmpty() {
		goRoot = paths.NewPathAsNative(build.Default.GOROOT)
		goRootVia = "build.Default.GOROOT"
	}

	goRootSrc := goRoot.Join("src").ToNative()

	for _, o := range others {
		otherSourceDirs = append(otherSourceDirs, listOfOthers(o)...)
	}

	if fi, e := os.Stat(goRootSrc.Join("go").String()); e != nil || !fi.IsDir() {
		if m, e := goRootSrc.Join("*.go").Glob(); e != nil || m == nil || len(m) == 0 {
			fmt.Fprintf(os.Stderr, "Does not exist or is not a Go source directory: %s (specified via %s);\n%v",
				goRootSrc, goRootVia, m)
			os.Exit(2)
		}
	}

	if goPath.String() == "" {
		fmt.Fprintf(os.Stderr, "no Go source path defined via either $GOPATH or --go-path")
		os.Exit(1)
	}
	if fi, e := os.Stat(goPath.String()); e == nil && fi.IsDir() && goPath.Base() != "src" {
		goPath = goPath.Join("src").ToNative()
	}

	godb.SetClojureSourceDir(clojureImportDir, goPath)
	generatedPkgPrefix = godb.ClojureSourceDir.JoinPaths(importStdRoot).String() + "/"
	importMe := path.Join(generatedPkgPrefix, goStdPrefix.String())

	if godb.Verbose {
		fmt.Printf("goRootSrc: %s\n", goRootSrc)
		fmt.Printf("GOPATH: %s\n", GOPATH)
		fmt.Printf("goPath: %s\n", goPath)
		fmt.Printf("ClojureSourceDir: %s\n", godb.ClojureSourceDir)
		fmt.Printf("outputDir: %s\n", outputDir)
		fmt.Printf("clojureImportDir: %s\n", clojureImportDir)
		fmt.Printf("jokerSourceDir: %s\n", jokerSourceDir)
		fmt.Printf("importStdRoot: %s\n", importStdRoot)
		fmt.Printf("generatedPkgPrefix: %s\n", generatedPkgPrefix)
		fmt.Printf("importMe: %s\n", importMe)
		for _, o := range otherSourceDirs {
			fmt.Printf("other: %s\n", o)
		}
	}

	readCoreApiFile(jokerSourceDir)

	Templates = template.Must(template.New("Templates").Funcs(TemplatesFuncMap).ParseGlob(jokerSourceDir.Join("tools", "gostd", "templates", "*.tmpl").String()))
	if godb.Verbose {
		strs := strings.Split(Templates.DefinedTemplates()+",", " ")
		sort.Strings(strs[4:]) // skip "; defined templates are: " in [0-3]
		fmt.Println(strings.TrimRight(strings.Join(strs, " "), ","))
	}

	outputGoStdDir := ""
	if outputDir.String() != "" {
		outputGoStdDir = outputDir.JoinPaths(importStdRoot.ToNative()).String()
		if replace {
			if e := os.RemoveAll(outputGoStdDir); e != nil {
				fmt.Fprintf(os.Stderr, "Unable to effectively 'rm -fr %s'\n", outputGoStdDir)
				os.Exit(1)
			}
		}

		if !overwrite {
			if _, e := os.Stat(outputGoStdDir); e == nil ||
				(!strings.Contains(e.Error(), "no such file or directory") &&
					!strings.Contains(e.Error(), "The system cannot find the")) { // Sometimes "...file specified"; other times "...path specified"!
				msg := "already exists"
				if e != nil {
					msg = e.Error()
				}
				fmt.Fprintf(os.Stderr, "Refusing to populate existing directory %s; please 'rm -fr' first, or specify --overwrite or --replace: %s\n",
					outputGoStdDir, msg)
				os.Exit(1)
			}
			if e := os.MkdirAll(outputGoStdDir, 0777); e != nil {
				fmt.Fprintf(os.Stderr, "Cannot 'mkdir -p %s': %s\n", outputGoStdDir, e.Error())
				os.Exit(1)
			}
		}
	}

	godb.AddMapping(goRootSrc, goNsPrefix, importMe)
	root := goRootSrc.Join(".")
	AddWalkDir(goRootSrc, root, goNsPrefix, importMe)

	for _, o := range otherSourceDirs {
		AddWalkDir(o, o.Join("."), "x.y.z.", path.Join(generatedPkgPrefix, "x/y/z/"))
	}

	err, badDir := WalkAllDirs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory %s: %v", badDir, err)
		os.Exit(1)
	}

	allTypesSorted := SortAllTypes()

	SetSwitchableTypes(allTypesSorted)

	SortedConstantInfoMap(GoConstants,
		func(c string, ci *ConstantInfo) {
			GenConstant(ci)
		})

	SortedVariableInfoMap(GoVariables,
		func(c string, ci *VariableInfo) {
			GenVariable(ci)
		})

	/* Generate type-code snippets in sorted order. */
	SortedTypeInfoMap(TypesByGoName(),
		func(t string, ti TypeInfo) {
			GenType(t, ti)
		})

	GenTypeCtors(allTypesSorted)

	GenQualifiedFunctionsFromEmbeds(allTypesSorted)

	/* Generate function-code snippets in alphabetical order. */
	SortedFuncInfoMap(QualifiedFunctions,
		func(f string, v *FuncInfo) {
			//			fmt.Printf("main.go: Qualifiedfunctions[%s]\n", f)
			if v.Fd != nil && v.Fd.Recv == nil {
				GenStandalone(v)
			} else {
				GenReceiver(v)
			}
		})

	OutputPackageCode(path.Join(outputGoStdDir, goStdPrefix.String()), generateEmpty)

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
	RegisterPackages(packagesArray, outputDir)
	RegisterClojureFiles(dotJokeArray, outputDir)

	RegisterGoTypeSwitch(allTypesSorted, outputDir)

	if godb.Verbose || summary {
		fmt.Printf("ABENDs:")
		abends.PrintAbends()
		fmt.Printf(`
Totals: functions=%d generated=%d (%s%%)
          non-receivers=%d (%s%%) generated=%d (%s%%)
          receivers=%d (%s%%) generated=%d (%s%%)
          methods=%d (%s%%) generated=%d (%s%%)
        types=%d
          constructable=%d ctors=%d (%s%%)
        constants=%d generated=%d (%s%%)
        variables=%d generated=%d (%s%%)
`,
			NumFunctions, NumGeneratedFunctions, pct(NumGeneratedFunctions, NumFunctions),
			NumStandalones, pct(NumStandalones, NumFunctions), NumGeneratedStandalones, pct(NumGeneratedStandalones, NumStandalones),
			NumReceivers, pct(NumReceivers, NumFunctions), NumGeneratedReceivers, pct(NumGeneratedReceivers, NumReceivers),
			godb.NumMethods, pct(godb.NumMethods, NumFunctions), godb.NumGeneratedMethods, pct(godb.NumGeneratedMethods, godb.NumMethods),
			NumTypes,
			NumCtableTypes, NumGeneratedCtors, pct(NumGeneratedCtors, NumCtableTypes),
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
