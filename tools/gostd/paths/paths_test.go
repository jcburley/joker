package paths

import (
	"path"
	"path/filepath"
	"testing"
)

var ut = "/home/somebody/this/is/a/file.txt"
var u = NewUnixPath(ut)

var nt = filepath.FromSlash(ut)
var n = NewNativePath(nt)

var utd = path.Dir(ut)
var ud = NewUnixPath(utd)

var ntd = filepath.Dir(nt)
var nd = NewNativePath(ntd)

var utb = path.Base(ut)
var ntb = filepath.Base(nt)

func TestUnixCtorAndString(t *testing.T) {
	if u.String() != ut {
		t.Fail()
	}
}

func TestNativeCtorAndString(t *testing.T) {
	if n.String() != nt {
		t.Fail()
	}
}

func TestUnixConversion(t *testing.T) {
	if u.ToNative() != n {
		t.Fail()
	}
}

func TestNativeConversion(t *testing.T) {
	if n.ToUnix() != u {
		t.Fail()
	}
}

func TestUnixJoin(t *testing.T) {
	if ud.Join(utb) != u {
		t.Fail()
	}
}

func TestNativeJoin(t *testing.T) {
	if nd.Join(ntb) != n {
		t.Fail()
	}
}

func TestUnixSplit(t *testing.T) {
	nud, b := u.Split()
	utdPlusSlash := NewUnixPath(utd + "/")
	if nud != utdPlusSlash {
		t.Error(nud, utdPlusSlash)
	}
	if b != utb {
		t.Error(b, utb)
	}
}

func TestNativeSplit(t *testing.T) {
	nnd, b := n.Split()
	ntdPlusSlash := NewNativePath(ntd + string(filepath.Separator))
	if nnd != ntdPlusSlash {
		t.Error(nnd, ntdPlusSlash)
	}
	if b != ntb {
		t.Error(b, ntb)
	}
}

func TestUnixDir(t *testing.T) {
	if u.Dir() != ud {
		t.Error(u.Dir(), ud)
	}
}

func TestNativeDir(t *testing.T) {
	if n.Dir() != nd {
		t.Error(n.Dir(), nd)
	}
}

func TestUnixBase(t *testing.T) {
	if u.Base() != utb {
		t.Error(u.Base(), utb)
	}
}

func TestNativeBase(t *testing.T) {
	if n.Base() != ntb {
		t.Error(n.Base(), ntb)
	}
}
