package paths

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

var SkipDir = filepath.SkipDir

type Path interface {
	String() string

	// These always return the same concrete type as upon which they operate:
	// 	Dir() Path
	// 	EvalSymlinks() (Path, error)
	// 	Join(el ...string) Path
	// 	Split() (Path, string)
	//	Walk(WalkFunc) error

	Base() string
}

type UnixPath struct {
	Path
	path string
}

type NativePath struct {
	Path
	path string
}

type UnixWalkFunc func(path UnixPath, info os.FileInfo, err error) error
type NativeWalkFunc func(path NativePath, info os.FileInfo, err error) error

func NewUnixPath(p string) UnixPath {
	return UnixPath{path: p}
}

func NewNativePath(p string) NativePath {
	return NativePath{path: p}
}

func (u UnixPath) Base() string {
	return path.Base(u.String())
}

func (n NativePath) Base() string {
	return filepath.Base(n.String())
}

func (u UnixPath) Dir() UnixPath {
	return NewUnixPath(path.Dir(u.String()))
}

func (n NativePath) Dir() NativePath {
	return NewNativePath(filepath.Dir(n.String()))
}

func (u UnixPath) EvalSymlinks() (UnixPath, error) {
	p, e := filepath.EvalSymlinks(u.ToNative().String())
	return NewUnixPath(p), e
}

func (n NativePath) EvalSymlinks() (NativePath, error) {
	p, e := filepath.EvalSymlinks(n.String())
	return NewNativePath(p), e
}

func (u UnixPath) Join(el ...string) UnixPath {
	return NewUnixPath(path.Join(u.String(), path.Join(el...)))
}

func (n NativePath) Join(el ...string) NativePath {
	return NewNativePath(filepath.Join(n.String(), filepath.Join(el...)))
}

func (u UnixPath) Split() (UnixPath, string) {
	d, b := path.Split(u.String())
	return NewUnixPath(d), b
}

func (n NativePath) Split() (NativePath, string) {
	d, b := filepath.Split(n.String())
	return NewNativePath(d), b
}

func (u UnixPath) String() string {
	return u.path
}

func (n NativePath) String() string {
	return n.path
}

func (u UnixPath) ToNative() NativePath {
	return NewNativePath(filepath.FromSlash(u.String()))
}

func (n NativePath) ToUnix() UnixPath {
	return NewUnixPath(filepath.ToSlash(n.String()))
}

func (u UnixPath) Walk(walkFn UnixWalkFunc) error {
	return filepath.Walk(u.ToNative().String(),
		func(p string, info os.FileInfo, err error) error {
			return walkFn(NewNativePath(p).ToUnix(), info, err)
		})
}

func (n NativePath) Walk(walkFn NativeWalkFunc) error {
	return filepath.Walk(n.String(),
		func(p string, info os.FileInfo, err error) error {
			return walkFn(NewNativePath(p), info, err)
		})
}

func (u UnixPath) RelativeTo(prefix UnixPath) (UnixPath, bool) {
	us := u.String()
	beyond := strings.TrimPrefix(us, prefix.String()+"/")
	if us == beyond {
		return u, false
	}
	return NewUnixPath(beyond), true
}

func (n NativePath) RelativeTo(prefix NativePath) (NativePath, bool) {
	ns := n.String()
	beyond := strings.TrimPrefix(ns, prefix.String()+string(filepath.Separator))
	if ns == beyond {
		return n, false
	}
	return NewNativePath(beyond), true
}
