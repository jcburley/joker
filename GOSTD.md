# GOSTD Usage

Build the version of Joker on the `gostd` branch as described in the [Joker README](README.md#the-gostd-namespaces).

After building, HTML documentation is available in the `docs` directory. For example, I use a URL to [my local docs tree](file:///home/craig/go/src/github.com/candid82/joker/docs/index.html) to get the latest info.

Or, use [the GOSTD-specific namespace documentation](https://burleyarch.com/joker/docs) to get an idea of what is available, as those pages are generally updated when new features (supporting more, or better, conversions/wrappers of Go packages to Joker) are pushed to the repository. (The Windows pages are updated less frequently.)

Note that `gostd` is still very much a "work in progress". It does not convert the entire `std` library provided by Go. Omissions are generally due to language features (of Go), used by packages (their types, constants, variables, standalone functions, and receivers), that the `gostd` tool does not yet convert, and so omits from the generated code that gets built into Joker.

## Recent Design Changes

### 2020-12-17

Type aliases (e.g. `type foo = bar`) are now ignored, so their Clojure names are not available for use.

Their use (in the Go standard library) seems to have been introduced (with respect to the timeframe in which `gostd` has existed) in Go version `1.16.1beta1`.

Ideally, `gostd` would support them, but as they seem designed for only short-term use around refactoring, properly implementing them in Joker seems less urgent than numerous other matters.

For more information on type aliases, see [Proposal: Type Aliases](https://go.googlesource.com/proposal/+/master/design/18130-type-alias.md) by Russ Cox and Robert Griesemer, December 16, 2016.

### 2020-04-17

Values returned by functions (this includes receivers and methods) are now returned "as-is", rather than (for some types) autoconverted to suitable Joker representations.

For example, calling a Go function returning `[]string` now returns a `GoObject` wrapping that same object (which has a type named `arrayOfstring`, since `[` and `]` are invalid symbol characters), rather than a vector of String objects.

Use `(vec ...)` to perform an explicit conversion, in this example, as `GoObject`s that wrap appropriate types are `Seqable` and thus support `(count ...`), `(rest ...)`, and so on.

This change improves performance in cases where the returned value will be used as-is, or only limited information (such as a given element or the number of elements) is needed, by Joker code, and where the number of returned elements (or their individual elements) is large.

*Note:* Vectors are still returned when the called Go function returns multiple arguments, since Go does not define multiple arguments as a single type.

## Sample Usage

This sends a completely empty (and thus technically invalid SMTP) message, as the conversion of a `string` to `[]int` is TBD:

```
user=> (use 'go.std.net.smtp)
nil
user=> (def au (PlainAuth "" "james@burleyarch.com" "NOTMYPASSWORD" "p25.llamail.com"))
#'user/au
user=> (SendMail "p25.llamail.com:smtp" au "james@burleyarch.com" ["james-recipient@burleyarch.com"] [])
""
user=>
```

## Design Principles

The `go.std.` namespaces being automatically generated, they are not necessarily intended for direct use by business logic:
* They aren't the same on all architectures
* They're very low-level -- little effort is made to provide Clojure-like wrapping beyond the minimum necessary

Yet, by (someday) providing _all_ the (supported) APIs, Joker enables higher-level, Clojure-like, APIs (that call these low-level API wrappers) to be written without requiring changes to the Joker codebase or executable itself.

## Including Other Go Packages

*NOTE:* This is work-in-progress and not yet complete.

```
$ touch NO-GOSTD.flag # Inhibit automatic running of gostd tool
$ build # Build canonical Joker
$ go get golang.org/x/crypto/ssh # Grab a sample package
$ (cd tools/gostd && go build) # Build gostd
$ ./tools/gostd/gostd --others golang.org/x/crypto/ssh --replace --joker . # Wrap both go.std.* and golang.org/x/crypto/ssh packages
$ build # Build Joker again, this time with additional packages
$
```

## Types

Named types (other than aliases), defined by the packages wrapped by the `gostd` tool, are themselves wrapped as `Object`s of type `GoType`.
`GoType` objects are found in the pertinent wrapper namespaces keyed by the type names.

For example, the `MX` type defined in the `net` package is wrapped as `go.std.net/MX`, which is a `GoType` that serves as a "handle" for all type-related activities, such as:

* Constructing a new instance: `(def mx (new go.std.net/MX {:Host "burleyarch.com" :Pref 10}))` => `&{burleyarch.com 10}`
* Identifying the type of an object: `(GoTypeOf (deref mx))` => `go.std.net/MX`
* Comparing types of objects: `(= (GoTypeOf mx) (GoTypeOf something-else)`

Each package-defined type has a reference (pointed-to) version that is also provided (e.g. `*MX`) in the namespace.

Some types have receivers. E.g. [`*go.std.os/File`](https://burleyarch.com/joker/docs/amd64-linux/go.std.os.html#*File) has a number of receivers, such as `Name`, `WriteString`, and `Close`, that maybe be invoked on it via e.g. `(Go f :Name)`, where `f` is (typically) returned from a call to `Create` or `Open` in the `go.std.os` namespace, or could be `(deref Stdin)` (to take a snapshot, usually in the form of a `GoObject`, of the `GoVar` named `Stdin`).

Methods on `interface{}` (abstract) types are now supported, though only some of them as of this writing. E.g. `(go.std.os/Stat "existing-file")` returns (inside the returned vector) a concrete type that is actually private to the Go library, so is not directly manipulatable via Joker, but which also implements the [`go.std.os/FileInfo`](https://burleyarch.com/joker/docs/amd64-linux/go.std.os.html#FileInfo) abstract type. Accordingly, `(Go fi :ModTime)` works on such an object.

## Constants

(Most) constants, defined in packages, are converted and thus available for reference. In some cases, their type is `Number` when an `Int` would suffice; this is due to how the conversion code is currently implemented, in that it doesn't attempt to fully evaluate the constant expressions in all cases, just provide some "guesses".

## Variables

Pointers to global variables are wrapped in `GoVar{}` objects that can be unwrapped via `(deref gv)`, yielding corresponding objects that are "snapshots" of the values as of the invocation of `deref`. Such objects are (per GoObject-creation rules) either `GoObject` or native Joker wrappers (such as `Int` and `String`).

`(var-set var newval)`, where `var` is a GoVar, assigns `newval` to the variable. `newval` may be an ordinary object such as a `String`, `Int`, or `Boolean`; or it may be a `Var`, `GoVar`, or `GoObject`, in which case the underlying value is used (and potentially dereferenced once, if that enables assignment, though the original value is nevertheless returned by the function).

## GoObject

A `GoObject` is a Joker object that wraps a Go object (of type `interface{}`). E.g.:

```
$ joker
Welcome to joker v0.15.7-gostd. Use '(exit)', EOF (Ctrl-D), or SIGINT (Ctrl-C) to exit.
user=> (def r (go.std.net/Interfaces))
#'user/r
user=> r
[[{1 16384 lo0  up|loopback|multicast} {2 1280 gif0  pointtopoint|multicast} {3 1280 stf0  0} {5 1500 en0 78:4f:43:84:9e:b3 up|broadcast|multicast} {6 2304 p2p0 0a:4f:43:84:9e:b3 up|broadcast|multicast} {7 1484 awdl0 fe:e0:5f:62:7a:ec up|broadcast|multicast} {8 1500 llw0 fe:e0:5f:62:7a:ec up|broadcast|multicast} {9 1500 en3 82:c9:92:c2:a0:01 up|broadcast|multicast} {10 1500 en1 82:c9:92:c2:a0:00 up|broadcast|multicast} {11 1500 en4 82:c9:92:c2:a0:05 up|broadcast|multicast} {12 1500 en2 82:c9:92:c2:a0:04 up|broadcast|multicast} {13 1500 bridge0 82:c9:92:c2:a0:00 up|broadcast|multicast} {14 1380 utun0  up|pointtopoint|multicast} {15 2000 utun1  up|pointtopoint|multicast} {4 1500 en5 ac:de:48:00:11:22 up|broadcast|multicast}] nil]
user=> (type r)
Vector
user=> (def i (r 0))
#'user/i
user=> i
[{1 16384 lo0  up|loopback|multicast} {2 1280 gif0  pointtopoint|multicast} {3 1280 stf0  0} {5 1500 en0 78:4f:43:84:9e:b3 up|broadcast|multicast} {6 2304 p2p0 0a:4f:43:84:9e:b3 up|broadcast|multicast} {7 1484 awdl0 fe:e0:5f:62:7a:ec up|broadcast|multicast} {8 1500 llw0 fe:e0:5f:62:7a:ec up|broadcast|multicast} {9 1500 en3 82:c9:92:c2:a0:01 up|broadcast|multicast} {10 1500 en1 82:c9:92:c2:a0:00 up|broadcast|multicast} {11 1500 en4 82:c9:92:c2:a0:05 up|broadcast|multicast} {12 1500 en2 82:c9:92:c2:a0:04 up|broadcast|multicast} {13 1500 bridge0 82:c9:92:c2:a0:00 up|broadcast|multicast} {14 1380 utun0  up|pointtopoint|multicast} {15 2000 utun1  up|pointtopoint|multicast} {4 1500 en5 ac:de:48:00:11:22 up|broadcast|multicast}]
user=> (type i)
GoObject
user=> (GoTypeOf i)
go.std.net/arrayOfInterface
user=> (def j (get i 1))
#'user/j
user=> j
{2 1280 gif0  pointtopoint|multicast}
user=> (type j)
GoObject
user=> (GoTypeOf j)
go.std.net/Interface
user=>
```

In the above case, a `GoObject` that wraps an instance of `[]net.Interface` is returned by a single call to `go.std.net/Interface`: though not a (Clojure) vector, `get` works on that array similarly. That `GoObject` is in turn is wrapped in a Clojure vector along with the `error` return value, per the "Go return type" shown by `doc`.

### Copying GoObjects

Generally, Joker avoids ever _copying_ a `GoObject`, in order to permit maximum flexibility of use (such as when one contains active state), to preserve some semblance of performance, and to avoid issues when they have members of type `sync.Mutex` (which cannot always be copied).

As a result, references to such objects are generally used.

### Dereferencing a GoObject

`(deref obj)` can be used to dereference a wrapped object, returning another `GoObject[]` with the dereferenced object as of that dereference, or to the original object if it wasn't a pointer to an object.

### Obtaining a Reference to a GoObject

Similarly, `(ref obj)` returns a (`GoObject` wrapping a) reference to either the original (underlying) Go object, if it supports that; or, more likely, to a copy that is made for this purpose.

The `reflect` package (in Go) is used here; see `reflect.CanAddr()`, which is used to determine whether the original underlying object allows a reference to be made to it. `reflect.New()` and `reflect.Indirect().Set()` are otherwise used to create a new, referencable, object that is set to the value of the original.

### Rules Governing GoObject Creation

When considering whether to wrap a given object in a `GoObject`, Joker normally substitutes a suitable Joker type (such as `Int`, `Number`, or `String`) when one is available and suitable for the underlying type (not just the value). For example, instead of wrapping an `int64` in a `GoObject`, Joker will wrap it in a `Number`, even if the value is small (such as zero).

### Constructing a GoObject

Akin to Clojure, `(new type ...)` is supported for (some) `GoObject` types:

```
user=> (use '[go.std.os])
nil
user=> (new FileMode 0321)
--wx-w---x
user=> (use '[go.std.html.template])
nil
user=> (def h (new HTML "this is an html object"))
#'user/h
user=> (type h)
GoObject
user=> (GoTypeOf h)
go.std.html.template/HTML
user=> (def le (new LinkError {:Op "hey" :Old "there" :New "you" :Err "silly"]))
#'user/le
user=> le
hey there you: silly
user=> (type le)
GoObject
user=> (GoTypeOf le)
*go.std.os/LinkError
user=> (goobject? le)
true
user=> (goobject? "foo")
false
user=>
```

If a particular constructor is missing, that indicates lack of support for the underlying type, or that the underlying type is abstract (`interface{}`).

### Converting a GoObject to a Joker (Clojure) Datatype

Given a `GoObject`, one may convert (to a native Joker type) and/or examine it via:
* `count`, which returns the number of elements (`.Len()`) for anything `seq` supports, without converting any of the elements themselves
* `deref`, which dereferences (indirects through the pointer wrapped by) the `GoObject` and returns the resulting "snapshot" of its value, either as a native Joker object or (if one isn't suitable) a `GoObject`; or, if the `GoObject` does not wrap a pointer, the underlying object is converted to a Joker object if possible, else wrapped by a newly constructed `GoObject`
* `get`, which returns the value corresponding to the given key for structs (the key can named via a string, keyword, or symbol), maps, arrays, slices, and strings; note, however, that a `GoObject` might be returned if a native Joker object is not suitable
* `if`, `and`, `or`, and similar, which convert to `bool` (and all `GoObject`'s evaluate as `true`)
* `seq`, which works on arrays, channels, maps, slices, strings, and structs, but is (currently) not lazily evaluated
* `vec`, like `seq` but returns a vector instead of a sequence
* `=`, `not=`, `compare`, and similar, which compare via Go's `==` operator (though with no autoconversion to `Number` or similar types) and return a `bool` or `int` result

#### Converting a Struct to a Joker (Clojure) Datatype

As touched on above, `count`, `seq`, and `vec` operate on Go `struct` types (dereferencing them once if necessary). Explaining further:
* `count` returns the number of fields (which will therefore be the same value for a given struct's type, regardless of the values in the struct itself)
* `seq` returns `([key-1 value-1] [key-2 value-2] ... [key-N value-N])`, where `N` equals `(count ...)` for the same object, where `key-n` is a keyword named after the Go field name (typically capitalized, due to being public)
* `vec` is like `seq`, but returns a vector of key-value pairs

Though it might be faster to retrieve individual fields as needed, a Joker (Clojure) map of field names (as keywords) to values can be constructed via e.g.:

```
(apply hash-map (flatten (seq goobject-wrapping-a-struct)))
```

NOTE: `seq` returns vectors of key/value pairs for Go `struct` objects to be consistent with how `seq` converts arguments of `Map` type.

### Calling a Go API

Calling a Go wrapper function (for a Go function, receiver, or method) in Joker requires ensuring the input arguments (if any) are of the proper types and then handling the returned results (if any) properly.

#### Input Arguments

Generally, the types of an input argument (to a Go wrapper function) must be either a built-in type (such as `int`) or a `GoObject` wrapping an object of the same (named) type as the corresponding input argument to the Go API.

Arguments of built-in types must be passed appropriate Clojure objects (`Int`, `String`, and so on) -- no "unwrapping" of `GoObject`'s is supported. However, GoObject-creation rules take this into account, substituting appropriate Clojure objects when the types are compatible.

Other arguments (with named types) are passed `GoObject` instances that can be:
* Constructed
* Extracted as members of other `GoObject` instances
* Returned by Go API wrappers

However, Joker does support some implicit conversion of Joker objects (such as `Int`) _to_ `GoObject`, in some ways beyond what the Go language itself provides, as explained below.

##### Implicit Conversion from Joker (Clojure) Type to GoObject

Though somewhat strongly typed, the Go language makes some common operations convenient via implicit type conversion. Consider `go/std/os.Chmod()`, for example:

```
user=> (use '[go.std.os :as o])
nil
user=> (doc o/Chmod)
-------------------------
go.std.os/Chmod
([_name _mode])
  Chmod changes the mode of the named file to mode.
[...]

Go input arguments: (name string, mode FileMode)

Go return type: error

Joker input arguments: [^String name, ^go.std.os/FileMode mode]

Joker return type: Error
nil
user=>
```

Note the second input argument, which is type `FileMode` (in the same package).

A Go program may perform an implicit conversion via e.g. `os.Chmod("sample.txt", 0644)`, in that `0644` is an untyped numeric constant. Such a constant defaults to `int`, but in this case it is implicitly converted to `uint32`, the underlying type of `go/std/os.FileMode`. Implicit conversion also works for an expression with only numeric-constant operands.

However, there's no implicit conversion when one or more _variables_ (even `const` "variables") are involved in the expression. So, given `const i int = 0644`, the Go compiler rejects `os.Chmod("sample.txt", i)` with:

```
./chmod.go:7:11: cannot use i (type int) as type os.FileMode in argument to os.Chmod
```

While this appears to discourage declaring a constant once in a package and then using it, instead of the constant itself, throughout the program, it does solve some thorny issues, as described in [this Go Blog post](https://blog.golang.org/constants). Further, one can work around it fairly easily by explicitly converting to the required type: `os.Chmod("sample.txt", os.FileMode(i))`. (That's awkward, but at least one needn't always specify e.g. `os.FileMode(0644)` when specifying a literal constant, as is the case in some strongly-typed languages.)

Joker offers similar implicit conversion, but (in accordance with the relatively laid-back type checking provided by Clojure) supports it regardless of whether the expression is constructed entirely out of constants. E.g.:

```
user=> (joker.os/sh "touch" "sample.txt")
{:success true, :exit 0, :out "", :err ""}
user=> (defn ll [] (:out (joker.os/sh "ls" "-l" "sample.txt")))
#'user/ll
user=> (ll)
"-rw-rw-r-- 1 craig craig 0 Jan 19 07:12 sample.txt\n"
user=> (o/Chmod "sample.txt" 0333)
""
user=> (ll)
"--wx-wx-wx 1 craig craig 0 Jan 19 07:12 sample.txt\n"
user=> (def i 0222)
#'user/i
user=> (type i)
Int
user=> (o/Chmod "sample.txt" i)
""
user=> (ll)
"--w--w--w- 1 craig craig 0 Jan 19 07:12 sample.txt\n"
user=> (def i 0111N)
#'user/i
user=> (o/Chmod "sample.txt" i)
""
user=> (ll)
"---x--x--x 1 craig craig 0 Jan 19 07:12 sample.txt\n"
user=> (def i "hey this is a string")
#'user/i
user=> (o/Chmod "sample.txt" i)
<repl>:51:1: Eval error: Arg[1] of go.std.os/Chmod must have type GoObject[os.FileMode], got String
user=> (ll)
"---x--x--x 1 craig craig 0 Jan 19 07:12 sample.txt\n"
user=> (def i 999999999999999999999999999)
#'user/i
user=> (o/Chmod "sample.txt" i)
<repl>:54:1: Eval error: Arg[1] of go.std.os/Chmod must have type uint32, got BigInt
user=> (ll)
"---x--x--x 1 craig craig 0 Jan 19 07:12 sample.txt\n"
user=> (def i 1.2)
#'user/i
user=> (o/Chmod "sample.txt" i)
""
user=> (ll)
"---------x 1 craig craig 0 Jan 19 07:12 sample.txt\n"
user=>
```

As shown above, implicit conversion even from `BigInt` and `Double` (as long as the value doesn't overflow the underlying type, which is `uint32` in this case) is supported.

Similarly, implicit conversion of `String` expressions to Go types that have `string` as their underlying (e.g. alias) type is supported. (Conversion to the floating-point and complex types is currently not supported, but only because these types are not easily tested due to there being no applicable APIs.)

#### Specifying the Target Function

For standalone functions, their Go name is (sometimes) directly usable as a Joker (Clojure) function. E.g. `(go.std.os/Chmod "sample.txt" 0777)`, where `Chmod` is the function name.

For receivers, given an object of the appropriate type, the `Go` function (specific to this version of Joker) is used, specifying the object, the name (as an expression that evaluates to a keyword, string, or symbol) of the receiver, and any arguments:

```
user=> (use 'go.std.net)
nil
user=> (def ip (IPv4 1 2 3 4))
#'user/ip
user=> ip
1.2.3.4
user=> (def im (IPv4Mask 252 0 0 0))
#'user/im
user=> im
fc000000
user=> (Go im :Size)
[6 32]
user=> (Go ip :Equal ip)
true
user=> (Go ip :Equal im)
<joker.core>:4458:3: Eval error: Argument 0 passed to (_net.IP)Equal() should be type GoObject[go.std.net/IP], but is GoObject[net.IPMask]
Stacktrace:
  global <repl>:20:1
  core/Go <joker.core>:4458:3
user=>
```

(Note the diagnostic produced when passing an object of incorrect type to a receiver, just as happens when passing the wrong thing to a standalone function.)

**IMPORTANT:** The `Go` function is, like `gostd` generally, a proof-of-concept prototype. Its name was chosen to set it apart from all other Joker code and specifically to identify it as referring to the Go language and its runtime. It might well be changed (incompatibly) or removed in the future.

Also note that Clojure's `.foo` form and its `.` special operator are not (yet?) supported. When they are, they'll (likely) be much more stable than `Go`.

#### Returned Values

Multiple return values are converted to a (Clojure) vector of the arguments, each treated as its own return value as far as this section of the document is concerned.

Types are returned as `GoObject` wrappers, and numbers are returned as `Int`, `BigInt`, `Double`, or whatever is best suited to handle the range of possible return values.

Returned `GoObject` instances can:
* Be ignored (they'll presumably be garbage-collected at some point)
* Be stringized (via e.g. `(str goobj)`)
* Be converted to a suitable Joker representation
* Be passed as arguments to Go API wrappers
* Be provided as members in a newly constructed `GoObject` instance (of the same or, more typically, some other, type)
* Have receivers/methods, defined on them, invoked via the `Go` function

Built-in type instances are converted directly to appropriate Joker (Clojure) types. For example, a Go API that returns `uint64` will be converted to a `Number` so as to ensure the full range of potential values is supported:


```
user=> (use '[go.std.math.rand :as r])
nil
user=> (doc r/Uint64)
-------------------------
go.std.math.rand/Uint64
([])
  Uint64 returns a pseudo-random 64-bit value as a uint64
from the default Source.

Go return type: uint64

Joker input arguments: []

Joker return type: BigInt
nil
user=> (r/Uint64)
13211699322299636880N
user=> (r/Uint64)
18275342588295813334N
user=> (r/Uint64)
1178250799499678761N
user=> (r/Uint64)
16901804822320105684N
user=> (r/Uint64)
15617289313243222146N
user=>
```

### Referencing a Member of a GoObject

#### Fields in Structures

`(Go obj field)` returns a `GoVar` wrapping the field named (typically via a keyword) by `field`. `obj` must denote a structure (`struct` type in Go).

The resulting `GoVar` can be dereferenced, as in `(deref var)` or `(var-get var)`, yielding a snapshot of the value of that field at that time.

It can also be changed, as if via Go's assignment statement, via `(var-set var newval)`.

For example:

```
user=> (use 'go.std.os)
nil
user=> (def le (new LinkError {:Op "hi" :Old "there" :New "you" :Err "silly"}))
#'user/le
user=> (str le)
"hi there you: silly"
user=> (Go le :Old)
0xc000f56a10
user=> (def v (Go le :Old))
#'user/v
user=> (var-set v "golly")
"golly"
user=> (str le)
"hi golly you: silly"
user=> (var-set Stdout "whoa")
<joker.core>:4517:3: Eval error: Cannot assign a string to a *os.File
Stacktrace:
  global <repl>:3:1
  core/Go <joker.core>:4517:3
user=>
```

#### Receivers and Methods

`(Go obj receiver [args...])`, where `obj` is a `GoObject`, calls a receiver (or method) for `obj` with the specified arguments.

As `Go` is a function, `receiver` (like `obj` and `args`) is evaluated. Typically it will be a self-evaluating form, as it must evaluate to a keyword, symbol, or string, which are supported as equivalent:

```
user=> (use 'go.std.os)
nil
user=> (def file (get (Create "TEMP.txt") 0))
#'user/file
user=> file
&{0xc000c01b60}
user=> (Go file :Name)
"TEMP.txt"
user=> (Go file :WriteString "Hello, world!\n")
[14 nil]
user=> (Go file "Close)
nil
user=> (Go file 'Name)  ;; Same as (Go file "Name") and (Go file :Name)
"TEMP.txt"
user=> (Go file "WriteString" "Hello, world again!\n")
[0 "write TEMP.txt: file already closed"]
user=> (slurp "TEMP.txt")
"Hello, world!\n"
user=> (Go (deref Stdin) :Name)
"/dev/stdin"
user=>
```

## Developer Notes

The version of `run.sh` on this branch invokes `tools/gostd/gostd` to create the `go.std...` namespaces, generate wrappers, and so on.

Before building Joker by hand, one can optionally run the `gostd` tool against a Go source tree (the default is found via `go/build.Default.GOROOT`), which _must_ correspond to the version of Go used to build Joker itself (as it likely will). It contains a complete `src` subdirectory, which `gostd` walks and parses, in order to populate `std/go/` and modify related Joker source files. Further, the build parameters (`$GOARCH`, `$GOOS`, etc.) must match -- so `build-all.sh` would have to pass those to this tool (if it was to be used) for each of the targets.

This is still a work in progress; for example, `net.LookupMX()` returns a vector including a `GoObject` wrapping a `[]*net.MX` object, which is not yet itself fully itself as a type, but can be examined. E.g.:

```
user=> (def mxe (go.std.net/LookupMX "burleyarch.com"))
#'user/mxe
user=> mxe
[[0xc00059e160] nil]
user=> (type mxe)
Vector
user=> (def mx (get mxe 0))
#'user/mx
user=> mx
[0xc00059e160]
user=> (type mx)
GoObject
user=> (GoTypeOf mx)
<joker.core>:4678:3: Eval error: Unsupported Go type []*net.MX
Stacktrace:
  global <repl>:20:1
  core/GoTypeOf <joker.core>:4678:3
user=> (deref mx)
[0xc00059e160]
user=> (def m0 (get mx 0))
#'user/m0
user=> m0
&{p25.llamail.com. 10}
user=> (type m0)
GoObject
user=> (GoTypeOf m0)
go.std.net/refToMX
user=> (deref m0)
{p25.llamail.com. 10}
user=> (type (deref m0))
GoObject
user=> (GoTypeOf (deref m0))
go.std.net/MX
user=> (def m (deref m0))
#'user/m
user=> m
{p25.llamail.com. 10}
user=> (get m 0)
<joker.core>:1105:4: Eval error: interface conversion: core.Int is not core.Fieldable: missing method AsFieldName
Stacktrace:
  global <repl>:31:1
  core/get <joker.core>:1105:4
user=> (get m :Pref)
10
user=> (get m :Host)
"p25.llamail.com."
user=>
```

You can run `gostd` standalone like this:

```
$ cd tools/gostd
$ go run . --output-code 2>&1 | less
```

Then page through the output. Code snippets intended for e.g. `std/go/std/net.joke` are printed to `stdout`, making iteration (during development of this tool) much easier. Specify `--joker <joker-source-directory>` (typically `--joker .`) to get all the individual `*.joke` and `*.go` files in `<dir>/std/go/`, along with modifications to `<dir>/custom.go`, `<dir>/core/data/core.joke`, and `<dir>/std/generate-std.joke`.

Most anything not supported results in either a `panic` or, more often, the string `ABEND` along with some kind of explanation. The latter is used to auto-detect a non-convertible function, in which case the snippet(s) are still output, but commented-out, so it's easy to see what's missing and (perhaps) why.

Among things to do to "productize" this:

* MOSTLY DONE: Might have to replace the current ad-hoc tracking of Go packages with something that respects `import` and the like
* Improve docstrings for constructors (show and document the members)
* Document the code better
* Assess performance impact (especially startup time) on Joker

### Evaluation Tests

A handful of tests (assertions) can be found in `tests/eval/go-objects.joke`. This is automatically run by the `eval-tests.sh` script (run in turn by `all-tests.sh`), available in "canonical" Joker.

`go-objects.joke` should be kept up-to-date as far as "smoke testing" basic capabilities. It should therefore be helpful as a guide as to which features are expected to work in a given version.

### Run gostd Tests

The `test.sh` script in `joker/tools/gostd/` runs tests against a small, then larger, then full, copy of Go 1.11's `golang/go/src/` tree. Invoke `test.sh` either with no options, or with `--on-error :` to run the `:` (`true`) command when it detects an error (the default being `exit 99`).

E.g.:

```
$ ./test.sh
$
```

The script currently runs tests in this order:

1. `_tests/small`
2. `_tests/big`
3. `build.Default.GOROOT`

After each test it runs, it uses `git diff` to compare the resulting `.gold` file with the checked-out version and, if there are any differences, it runs the command specified via `--on-error` (again, the default is `exit 99`, so the script will exit as soon as it sees a failing test).

### Update Tests on Other Machines

The Go standard library is customized per system architecture and OS, and `gostd` picks up these differences via its use of Go's build-related packages. That's why `_tests/gold/` has a subdirectory for each combination of `$GOARCH` and `$GOOS`. Updating another machine's copy of the `gostd` repo is somewhat automated via `update.sh` -- e.g.:

```
$ ./update.sh
remote: Enumerating objects: 8, done.
remote: Counting objects: 100% (8/8), done.
remote: Compressing objects: 100% (4/4), done.
remote: Total 6 (delta 4), reused 4 (delta 2), pack-reused 0
Unpacking objects: 100% (6/6), done.
From github.com:jcburley/joker
   2f356e5..b643457  gostd      -> origin/gostd
Updating 2f356e5..b643457
Fast-forward
 tools/gostd/main.go | 8 ++++++--
 tools/gostd/test.sh | 6 +++---
 2 files changed, 9 insertions(+), 5 deletions(-)
No changes to amd64-darwin test results.
$
```

(Note the final line of output, indicating the value of `$GOARCH-$GOOS` in the `go` environment.)

If there are changes to the test results, they'll be displayed (via `git diff`), and the script will then prompt as to whether to accept and update them:

```
Accept and update amd64-darwin test results? y
[gostd 5cfed10] Update amd64-darwin tests
 3 files changed, 200 insertions(+), 200 deletions(-)
Counting objects: 8, done.
Delta compression using up to 8 threads.
Compressing objects: 100% (8/8), done.
Writing objects: 100% (8/8), 3.90 KiB | 266.00 KiB/s, done.
Total 8 (delta 4), reused 0 (delta 0)
remote: Resolving deltas: 100% (4/4), completed with 4 local objects.
To github.com:jcburley/joker
   339fbba..5cfed10  master -> master
$
```

(Don't forget to `git pull gostd gostd` on your other development machines after updating test results, to avoid having to do the `git merge` dance when you make changes on them and try to `git push`.)

### Clean Up After Full Build

After building the `gostd` branch, numerous additional files will have been created, and several existing files (in the source distribution, including generated files) will have been modified.

Restore them via:

```
$ ./cleanup.sh
```

This should result in `git` showing no differences (tracked nor untracked files) if only `gostd` has made changes to the source tree. If Joker hadn't previously been successfully built, there'll be a diagnostic; but the result should still be a "cleaned" tree.

### Caching of Core-API Information

To ease development, `gostd` dynamically determines the list of exported functions in Joker's `core` package, and avoids generating calls to unlisted functions. This helps to catch missing APIs earlier in the development process (mainly, while building and testing `gostd` in isolation, versus requiring a build of Joker itself).

The resulting list is cached in `./core-apis.dat`, and reused by `./test.sh` for the second and third tests, to save time (which amounts to a substantial portion of the time it takes to build and test `gostd`).

As building Joker takes enough longer to make this caching less useful, `run.sh` deletes the cache prior to each build.


### Caching of Working Joker Version

To save time when building Joker, either an environment variable `$JOKER` or a copy (or link) named `joker-good` (in the top-level Joker directory), pointing to a previously-built and working version of Joker, can be provided.

This causes `run.sh` to skip building an initial version of Joker, and instead use that existing version to run `std/generate-std.joke` and then decide whether it is necessary to rebuild Joker with the resulting libraries (as will always be the case when `gostd` is involved).
