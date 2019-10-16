# GOSTD Usage

Build the version of Joker on the `gostd` branch as described in the [Joker README](https://github.com/jcburley/joker/blob/gostd/README.md) under "The go.std.* Namespaces".

After building, HTML documentation is available in the `docs` directory. For example, I use a URL to [my local docs tree](file:///home/craig/go/src/github.com/candid82/joker/docs/index.html) to get the latest info.

Or, use [the GOSTD-specific namespace documentation](https://burleyarch.com/joker/docs) to get an idea of what is available, as those pages are generally updated when new features (supporting more, or better, conversions/wrappers of Go packages to Joker) are pushed to the repository. (The Windows pages are updated less frequently.)

Note that `gostd` is still very much a "work in progress". It does not convert the entire `std` library provided by Go. Omissions are generally due to language features (of Go), used by packages (their types, constants, variables, standalone functions, and receivers), that the `gostd` tool does not yet convert, and so omits from the generated code that gets built into Joker.

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

Named types, defined by the packages wrapped by the `gostd` tool, are themselves wrapped as `Object`s of type `GoType`.
`GoType` objects are found in the pertinent wrapper namespaces keyed by the type names.

For example, the `MX` type defined in the `net` package is wrapped as `go.std.net/MX`, which is a `GoType` that serves as a "handle" for all type-related activities, such as:

* Constructing a new instance: `(def mx (new go.std.net/MX ["burleyarch.com" 10]))` => `&{burleyarch.com 10}`
* Identifying the type of an object: `(GoTypeOf (deref mx))` => `go.std.net/MX`
* Comparing types of objects: `(= (GoTypeOf mx) (GoTypeOf something-else)`

Each package-defined type has a reference (pointed-to) version that is also provided (e.g. `*MX`) in the namespace.

Some types have receivers. E.g. [`*go.std.os/File`](https://burleyarch.com/joker/docs/amd64-linux/go.std.os.html#*File) has a number of receivers, such as `Name`, `WriteString`, and `Close`, that maybe be invoked on it via e.g. `(Go f :Name)`, where `f` is (typically) returned from a call to `Create` or `Open` in the `go.std.os` namespace, or could be `(deref Stdin)` (to take a snapshot, usually in the form of a `GoObject`, of the `GoVar` named `Stdin`).

Methods on `interface{}` (abstract) types are now supported, though only some of them as of this writing. E.g. `(go.std.os/Stat "existing-file")` returns (inside the returned vector) a concrete type that is actually private to the Go library, so is not directly manipulatable via Joker, but which also implements the [`go.std.os/FileInfo`](https://burleyarch.com/joker/docs/amd64-linux/go.std.os.html#FileInfo) abstract type. Accordingly, `(Go fi "Modtime")` works on such an object.

## Constants

(Most) constants, defined in packages, are converted and thus available for reference. In some cases, their type is `Number` when an `Int` would suffice; this is due to how the conversion code is currently implemented, in that it doesn't attempt to fully evaluate the constant expressions in all cases, just provide some "guesses".

## Variables

Pointers to global variables are wrapped in `GoVar[]` objects that can be unwrapped via `(deref gv)`, yielding corresponding objects that are "snapshots" of the values as of the invocation of `deref`. Such objects are (per GoObject-creation rules) either `GoObject` or native Joker wrappers (such as `Int` and `String`).

## GoObject

A `GoObject` is a Clojure (Joker) object that wraps a Go object (of type `interface{}`). E.g.:

```
$ joker
Welcome to joker v0.12.9. Use EOF (Ctrl-D) or SIGINT (Ctrl-C) to exit.
user=> (use '[go.std.net :as n])
nil
user=> (doc n/Interfaces)
-------------------------
go.std.net/Interfaces
([])
  Interfaces returns a list of the system's network interfaces.

Go return type: ([]Interface, error)

Joker input arguments: []

Joker return type: [(vector-of go.std.net/Interface) Error]
nil
user=> (def r (n/Interfaces))
#'user/r
user=> r
[[{1 65536 lo  up|loopback} {2 1500 eth0 14:da:e9:1f:c8:57 up|broadcast|multicast} {3 1500 docker0 02:42:6a:a9:a8:d8 up|broadcast|multicast}] nil]
user=> (type r)
Vector
user=> (type (r 0))
Vector
user=> (type ((r 0) 0))
GoObject
user=> (GoTypeOf ((r 0) 0))
go.std.net/Interface
user=> ((r 0) 0)
{1 65536 lo  up|loopback}
user=>
```

In the above case, multiple `GoObject` objects are returned by a single call to `go.std.net/Interface`: they are returned as a (Clojure) vector, which in turn is wrapped in a vector along with the `error` return value, per the "Go return type" shown by `doc`.

Generally, Joker avoids ever _copying_ a `GoObject`, in order to permit maximum flexibility of use (such as when one contains active state), to preserve some semblance of performance, and to avoid issues when they have members of type `sync.Mutex` (which cannot always be copied).

As a result, pointers to such objects are returned as `atom` references to the very same objects.

`(deref obj)` can be used to dereference a wrapped object, returning another `GoObject[]` with the dereferenced object as of that dereference, or to the original object if it wasn't a pointer to an object.

*NOTE*: For now, `(Go obj "&")` returns a (`GoObject` wrapping a) reference to either the original (underlying) Go object, if it supports that, or (more likely) to a copy that is made for this purpose. This might be replaced someday by a proper `(ref ...)` function, which would presumably also work for ordinary Joker objects as it does in Clojure.

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
user=> (def le (new LinkError ["hey" "there" "you" "silly"]))
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

### Calling a Go API

Calling a Go wrapper function (for a Go function, receiver, or method) in Joker requires ensuring the input arguments (if any) are of the proper types and then handling the returned results (if any) properly.

#### Input Arguments

Generally, the types of an input argument (to a Go wrapper function) must be either a built-in type (such as `int`) or a `GoObject` wrapping an object of the same (named) type as the corresponding input argument to the Go API.

Arguments with built-in types must be passed appropriate Clojure objects (`Int`, `String`, and so on) -- no "unwrapping" of `GoObject`'s is supported. However, GoObject-creation rules take this into account, substituting appropriate Clojure objects when the types are compatible.

Other arguments (with named types) are passed `GoObject` instances that can be:
* Constructed
* Extracted as members of other `GoObject` instances
* Returned by Go API wrappers

However, Joker does support some implicit conversion of Clojure objects (such as `Int`) _to_ `GoObject`, in some ways beyond what the Go language itself provides, as explained below.

##### Implicit Conversion from Clojure Type to GoObject

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

For standalone functions, their Go name is (sometimes) directly usable as a Clojure function. E.g. `(go.std.os/Chmod "sample.txt" 0777)`, where `Chmod` is the function name.

For receivers, given an object of the appropriate type, the `Go` function (specific to this version of Joker) is used, specifying the object, the name (as an expression that evaluates to a string or symbol) of the receiver, and any arguments:

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
user=> (Go im "Size")
[6 32]
user=> (Go ip "Equal" ip)
true
user=> (Go ip "Equal" im)
<joker.core>:4458:3: Eval error: Argument 0 passed to (_net.IP)Equal() should be type GoObject[go.std.net/IP], but is GoObject[net.IPMask]
Stacktrace:
  global <repl>:20:1
  core/Go <joker.core>:4458:3
user=>
```

(Note the diagnostic produced when passing an object of incorrect type to a receiver, just as happens when passing the wrong thing to a standalone function.)

**IMPORTANT:** The `Go` function is, like `gostd` generally, a proof-of-concept prototype. Its name was chosen to set it apart from all other Clojure code and specifically to identify it as referring to the Go language and its runtime. It might well be changed (incompatibly) or removed in the future.

Also note that Clojure's `.foo` form and its `.` special operator are not (yet?) supported. When they are, they'll (likely) be much more stable than `Go`.

#### Returned Values

Multiple return values are converted to a (Clojure) vector of the arguments, each treated as its own return value as far as this section of the document is concerned.

Arrays are returned as vectors, types are returned as `GoObject` wrappers, and numbers are returned as `Int`, `BigInt`, `Double`, or whatever is best suited to handle the range of possible return values.

Returned `GoObject` instances can:
* Be ignored (they'll presumably be garbage-collected at some point)
* Be stringized (via e.g. `(str goobj)`)
* Be converted to a suitable Clojure representation
* Be passed as arguments to Go API wrappers
* Be provided as members in a newly constructed `GoObject` instance (of the same or, more typically, some other, type)
* Have receivers/methods, defined on them, invoked via the `Go` function

Built-in type instances are converted directly to appropriate Clojure types. For example, a Go API that returns `uint64` will be converted to a `Number` so as to ensure the full range of potential values is supported:


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

`(Go obj receiver [args...])`, where `obj` is a `GoObject`, calls a receiver (or method) for `obj` with the specified arguments.

As `Go` is a function, `receiver` (like `obj` and `args`) is evaluated. Typically it will be a self-evaluating form, as it must evaluate to a symbol, string, or keyword, which are supported as equivalent:

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

*NOTE*: As a special case, a `receiver` of `'&` or `"&"` (with no further arguments) returns a reference (pointer) to the underlying object, wrapped by a `GoObject`. `(ref ...)` should someday replace this, as it's something of a kludge.

### Converting a GoObject to a Clojure Datatype

TBD.

## Developer Notes

The version of `run.sh` on this branch invokes `tools/gostd/gostd` to create the `go.std...` namespaces, generate wrappers, and so on.

Before building Joker by hand, one can optionally run the `gostd` tool against a Go source tree (the default is found via `go/build.Default.GOROOT`), which _must_ correspond to the version of Go used to build Joker itself (as it likely will). It contains a complete `src` subdirectory, which `gostd` walks and parses, in order to populate `std/go/` and modify related Joker source files. Further, the build parameters (`$GOARCH`, `$GOOS`, etc.) must match -- so `build-all.sh` would have to pass those to this tool (if it was to be used) for each of the targets.

This is still a work in progress; for example, `net.LookupMX()` returns a vector including a vector of pointers to `net.MX` objects, which cannot yet be properly examined (though Go's conversion to text is often reasonably helpful). E.g.:

```
user=> (n/LookupMX "github.com")
[[&{aspmx.l.google.com. 1} &{alt1.aspmx.l.google.com. 5} &{alt2.aspmx.l.google.com. 5} &{alt3.aspmx.l.google.com. 10} &{alt4.aspmx.l.google.com. 10}] nil]
user=> (def r0 (((n/LookupMX "github.com") 0) 0))
#'user/r0
user=> r0
&{aspmx.l.google.com. 1}
user=> (deref r0)
<joker.core>:1448:3: Eval error: Arg[0] of core/deref__ must have type Deref, got GoObject[*net.MX]
Stacktrace:
  global <repl>:15:1
  core/deref <joker.core>:1448:3
user=>
```

You can run it standalone like this:

```
$ cd tools/gostd
$ go run . --output-code 2>&1 | less
```

Then page through the output. Code snippets intended for e.g. `std/go/std/net.joke` are printed to `stdout`, making iteration (during development of this tool) much easier. Specify `--joker <joker-source-directory>` (typically `--joker .`) to get all the individual `*.joke` and `*.go` files in `<dir>/std/go/`, along with modifications to `<dir>/custom.go`, `<dir>/core/data/core.joke`, and `<dir>/std/generate-std.joke`.

Anything not supported results in either a `panic` or, more often, the string `ABEND` along with some kind of explanation. The latter is used to auto-detect a non-convertible function, in which case the snippet(s) are still output, but commented-out, so it's easy to see what's missing and (perhaps) why.

Among things to do to "productize" this:

* MOSTLY DONE: Might have to replace the current ad-hoc tracking of Go packages with something that respects `import` and the like
* Improve docstrings for constructors (show and document the members)
* Refactor `gotypes.go`, as was started and (for the time being) abandoned on 2019-08-19 in the `gostd-bad-refactor` branch (now in progress in the form of `types/types.go`)
* Document the code better
* Assess performance impact (especially startup time) on Joker, and mitigate as appropriate (good progress has already been made here, via lazy loading of namespaces; it seems to be not much slower than standard Joker)

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

This should result in `git` showing no differences (tracked nor untracked files) if only `gostd` has made changes to the source tree.
