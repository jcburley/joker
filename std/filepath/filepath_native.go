package filepath

import (
	"os"
	"path/filepath"

	. "github.com/candid82/joker/core"
)

func fileSeq(root string) *ArrayVector {
	res := EmptyArrayVector()
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		PanicOnErr(err)
		m := FileInfoMap(path, info)
		res.Append(m)
		return nil
	})
	return res
}

func initNative() {
}
