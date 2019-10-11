package abends

import (
	"fmt"
	"regexp"
	"sort"
)

var abendRegexp *regexp.Regexp

var abends = map[string]int{}

func TrackAbends(a string) {
	subMatches := abendRegexp.FindAllStringSubmatch(a, -1)
	//	fmt.Printf("trackAbends: %v %s => %#v\n", abendRegexp, a, subMatches)
	for _, m := range subMatches {
		if len(m) != 2 {
			panic(fmt.Sprintf("len(%v) != 2", m))
		}
		n := m[1]
		if _, ok := abends[n]; !ok {
			abends[n] = 0
		}
		abends[n] += 1
	}
}

func PrintAbends() {
	type ac struct {
		abendCode  string
		abendCount int
	}
	a := []ac{}
	for k, v := range abends {
		a = append(a, ac{abendCode: k, abendCount: v})
	}
	sort.Slice(a,
		func(i, j int) bool {
			if a[i].abendCount == a[j].abendCount {
				return a[i].abendCode < a[j].abendCode
			}
			return a[i].abendCount > a[j].abendCount
		})
	for _, v := range a {
		fmt.Printf(" %s(%d)", v.abendCode, v.abendCount)
	}
}

func init() {
	abendRegexp = regexp.MustCompile(`ABEND([0-9]+)`)
}
