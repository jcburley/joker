package utils

import (
	"os"
	"sort"
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
