package main

import (
	"fmt"
	"io/ioutil"
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

func updateGenerateCustom(pkgs []string, f string) {
	if verbose {
		fmt.Printf("Adding custom loaded libraries to %s\n", filepath.ToSlash(f))
	}

	m := ";;;; Auto-modified by gostd at " + curTimeAndVersion() + "\n\n"

	newImports := ""
	importPrefix := "'go.std."
	curLine := "(def custom-namespaces ["
	for _, p := range pkgs {
		more := importPrefix + strings.Replace(p, "/", ".", -1)
		if curLine != "" && len(curLine)+len(more) > 77 {
			newImports += curLine + "\n "
			curLine = more
		} else {
			curLine += more
		}
		importPrefix = " 'go.std."
	}
	newImports += curLine + `])

(apply require :reload custom-namespaces)

(doseq [ns-sym custom-namespaces]
  (let [ns-name (str ns-sym)
        dir (rpl ns-name "." "/")
        ns-name-final (rpl ns-name #".*[.]" "")]
    (debug "Processing custom namespace" ns-name "in" dir "final name" ns-name-final)
    (spit (ns-file-name dir ns-name-final)
          (remove-blanky-lines (generate-ns ns-sym ns-name ns-name-final)))))
`

	m += newImports

	err := ioutil.WriteFile(f, []byte(m), 0777)
	check(err)
}
