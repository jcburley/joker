package importia

import (
	"./a"
	also_a "./also_a/a"
	"./b"
	c "./c/a"
	"fmt"
	_ "io"
	. "io/ioutil"
)

type ImportIa int

func ImportFunc() string {
	return "yes i am"
}

type ImportIaA a.ImportIa

type ImportIaB b.ImportIa

type ImportIaC c.ImportIa

type ImportIaAlsoA also_a.ImportIa

func test() {
	fmt.Printf("Testing; Discard=%v.\n", Discard)
}
