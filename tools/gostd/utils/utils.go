package utils

import (
	. "go/ast"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	. "strings"
)

func Check(e error) {
	if e != nil {
		panic(e)
	}
}

// func SortedStringMap(m map[string]string, f func(key, value string)) {
// 	var keys []string
// 	for k, _ := range m {
// 		keys = append(keys, k)
// 	}
// 	sort.Strings(keys)
// 	for _, k := range keys {
// 		f(k, m[k])
// 	}
// }

// func ReverseJoin(a []string, infix string) string {
// 	j := ""
// 	for idx := len(a) - 1; idx >= 0; idx-- {
// 		if idx != len(a)-1 {
// 			j += infix
// 		}
// 		j += a[idx]
// 	}
// 	return j
// }

func Unix(p string) string {
	return filepath.ToSlash(p)
}

func CommentGroupAsString(d *CommentGroup) string {
	s := ""
	if d != nil {
		s = d.Text()
	}
	return s
}

func CommentGroupInQuotes(doc *CommentGroup, jokIn, jokOut, goIn, goOut string) string {
	var d string
	if doc != nil {
		d = doc.Text()
	}
	if goIn != "" {
		if d != "" {
			d = Trim(d, " \t\n") + "\n\n"
		}
		d += "Go input arguments: " + goIn
	}
	if goOut != "" {
		if d != "" {
			d = Trim(d, " \t\n") + "\n\n"
		}
		d += "Go returns: " + goOut
	}
	if jokIn != "" {
		if d != "" {
			d = Trim(d, " \t\n") + "\n\n"
		}
		d += "Joker input arguments: " + jokIn
	}
	if jokOut != "" {
		if d != "" {
			d = Trim(d, " \t\n") + "\n\n"
		}
		d += "Joker returns: " + jokOut
	}
	return Trim(strconv.Quote(d), " \t\n")
}

var outs map[string]struct{}

func StartSortedOutput() {
	outs = map[string]struct{}{}
}

func AddSortedOutput(s string) {
	outs[s] = struct{}{}
}

func EndSortedOutput() {
	var keys []string
	for k, _ := range outs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		os.Stdout.WriteString(k)
	}
}
