package paths

import (
	"path"
	"path/filepath"
)

type Path interface {
	String() string
	Join(el ...string) Path
	Split() (Path, string)
	Dir() Path
	Base() string
}

type UnixPath struct {
	path string
}

type NativePath struct {
	path string
}

func NewUnixPath(p string) UnixPath {
	return UnixPath{p}
}

func NewNativePath(p string) NativePath {
	return NativePath{p}
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

func (u UnixPath) Join(el ...string) Path {
	return NewUnixPath(path.Join(u.String(), path.Join(el...)))
}

func (n NativePath) Join(el ...string) Path {
	return NewNativePath(filepath.Join(n.String(), filepath.Join(el...)))
}

func (u UnixPath) Split() (Path, string) {
	d, b := path.Split(u.String())
	return NewUnixPath(d), b
}

func (n NativePath) Split() (Path, string) {
	d, b := filepath.Split(n.String())
	return NewNativePath(d), b
}

func (u UnixPath) Dir() Path {
	return NewUnixPath(path.Dir(u.String()))
}

func (n NativePath) Dir() Path {
	return NewNativePath(filepath.Dir(n.String()))
}

func (u UnixPath) Base() string {
	return path.Base(u.String())
}

func (n NativePath) Base() string {
	return filepath.Base(n.String())
}
