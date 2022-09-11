# GOSTD Usage

Build the version of Joker on the **gostd** branch as described in the [Joker README](README.md#gostd).

After building, HTML documentation is available in the `docs` directory. For example, I use a URL to [my local docs tree](file:docs/index.html) to get the latest info.

Or, use [the GOSTD-specific namespace documentation](https://burleyarch.com/joker/docs) to get an idea of what is available, as those pages are generally updated when new features (supporting more, or better, conversions/wrappers of Go packages to Joker) are pushed to the repository. (The Windows pages are updated less frequently.)

Note that **gostd** is still very much a "work in progress". It does not convert the entire `std` library provided by Go, though is now at or greater than 90% coverage in most cases. Omissions are generally due to language features (of Go), used by packages (their types, standalone functions, and methods/receivers), that the **gostd** tool does not yet convert, and so omits from the generated code that gets built into Joker. Further, some key "standalone" functions (such as those in the `builtin` package) are not yet available.

Current conversion-rate stats on `amd64-darwin`:

```
Totals: functions=4461 generated=4304 (96.48%)
          non-receivers=1564 (35.06%) generated=1445 (92.39%)
          receivers=2121 (47.55%) generated=2089 (98.49%)
          methods=776 (17.40%) generated=770 (99.23%)
        types=2843
          constructable=797 ctors=672 (84.32%)
        constants=4169 generated=4169 (100.00%)
        variables=439 generated=439 (100.00%)
```

## Recent Design Changes

### 2021-10-12 (v0.14)

Many methods, available to Go code via types embedded in interfaces and structs, are now also made available to Joker code. E.g. `(.Close c)` on a `net.TCPConn` connection `c` now works, even though `TCPConn` itself does not implement `Close`, because its struct embeds the (non-exported) `net.conn` type, which does implement `Close`. Note that `gostd` tries to avoid wrapping a method, available via an embed, that is explicitly implemented by the containing type. For example, `text/template/parse.DotNode` is a struct that embed `NodeType`, for which the `Type()` method is defined; but, since `DotNode` implements its own `Type()`, that method, not `NodeType`'s, is left as the wrapped type for e.g. `(.Type d)`, where `d` is type`DotNode`.

A value receiver for a reference (wrapped by a `GoObject`) can now be called without having to explicitly dereference the object. Some support for calling a pointer receiver for a value is now provided, but doesn't work in the straightforward case of a `GoObject` wrapping that value, as the Go runtime (specifically, the `reflect` package) does not always see such a value as capable of being addressable (`reflect.CanAddr()` fails).

Pointer types are named using `*`, instead of `refTo`, as `*` is a reasonably common character in Clojure symbols. However, as `[` and `]` are not valid (without escaping in some fashion), arrays/slices are still named using `arrayOf`, leading to some bodges such as `arrayOf*Foo`. It's unclear whether any elegant solution to this problem exists.

Preliminary support for `func()` types (taking no arguments and returning no value) is provided. However, it's purely experimental, and no attempt is made to ensure single-threading behavior with respect to other running Joker code. As the only test case currently defined invokes the function on a separate thread from the main thread passing the function, it's either happenstance, or the very limited use case of the Joker code that implements the function, that allows that test case to pass.

Substantial refactoring of `gostd` continues with this preliminary release, but much more is planned, some of which will likely be visible to Joker code calling the wrapped namespaces.

### 2021-04-28 (v0.13)

The `(set! ...)` special form is implemented, and supersedes use of `(var ...)` on field references. E.g. instead of `(var-set (var (.member instance)) value)`, use `(set! (.member instance) value)`, which is much more Clojure-like.

The `GoVar` object no longer exists. Variables are now implemented as `GoObject`s that wrap references (pointers) to the respective variables, with automatic derefencing for ordinary usage. So, `SomeVar` yields a snapshot of the `SomeVar` variable; now, `(set! SomeVar new-value)` is used to change the value of the variable, and `(set! (.SomeField SomeVar) new-value)` to change the value of a specific field when `SomeVar` is a `struct`.  (`(deref SomeVar)`, aka `@SomeVar`, no longer work on global variables as previously described for earlier versions of this fork. Similarly, `(var (...))` is no longer supported.)

The `ref` function (in `joker.core`) has been removed, as it was not similar enough to Clojure's `ref`. A replacement function (or possibly a special form) is being contemplated for a future release.

Values of type `error` are now wrapped in `GoObject`s instead of being converted into `String`s. This allows receivers for those values to be invoked, and is consistent with preserving the native type when not precisely implemented by a Joker type (in this case, `Error`, an abstract type). When an `error` type is needed, the supplied expression may be `String` or `GoObject[error]`.

### Earlier Changes

See [below](#earlier-design-changes) for more history.

## Sample Usage

This sends a completely empty (and thus technically invalid SMTP) message:

```
user=> (use 'go.std.net.smtp)
nil
user=> (def au (PlainAuth "" "james@burleyarch.com" "NOTMYPASSWORD" "p25.llamail.com"))
#'user/au
user=> (SendMail "p25.llamail.com:smtp" au "james@burleyarch.com" ["james-recipient@burleyarch.com"] [])
""
user=>
```

## Design Goals

Normally, to make Go APIs available from Joker code, one might write a custom Joker source file, such as `std/some-namespace.joke`, perhaps accompanied by `std/os/some-namespace_native*.go` files, and then rebuild Joker.

Alternately, one could add a namespace to `core/data/some-namespace.joke` and have it call new Go code, calling the desired APIs, provided in `core/procs.go` (or similar), which can be more complicated due to needing to modify other files and then rebuild Joker.

Either way, the source code of Joker has to be modified, and Joker rebuilt, so the API can be called. Both approaches require the kind of in-depth knowledge described, for developers working on Joker internals, in [`DEVELOPER.md`](https://github.com/candid82/joker/blob/master/DEVELOPER.md).

This **gostd** fork strives to provide out-of-the-box access to all the Go standard-library packages via very low-level Joker access. Each such Go package (say, `html/template`) is wrapped by a Joker namespace named with the `go.std.` prefix (so, `go.std.html.template`) and containing functions, methods, receivers, constants, variables, and types, all providing access to the corresponding public elements of the Go package.

The `go.std.` namespaces being automatically generated, they are not necessarily intended for direct use by business logic:
* They aren't the same on all architectures
* They're very low-level; little effort is made to provide Clojure-like wrapping beyond the minimum necessary

Yet, by (someday) providing _all_ the (supported) APIs, Joker enables higher-level, Clojure-like, APIs (that call these low-level API wrappers) to be written without requiring changes to the Joker codebase or executable itself.

That allows one to write custom Joker libraries (namespaces) as “pure” Joker code. These would provide Joker-like (Clojure-like) APIs that are implemented by calling the **gostd**-provided wrappers underneath. No rebuild of Joker would be required, nor the source tree changed (in the source-control sense; it is actually changed when building the **gostd** version, in a fashion more expansive than that which official Joker does via `go generate`).

This "pure" Joker code, that one might write to provide Clojure-like access to raw Go APIs wrapped by **gostd**, would presumably be deployed alongside one’s application as yet another namespace, or perhaps more widely if used across an organization or in a project.

That is, such Joker code would be organized and deployed as described in [`LIBRARIES.md`](https://github.com/candid82/joker/blob/master/LIBRARIES.md), intended for developers of Joker code, rather than the earlier-cited document, which is intended for developers working on Joker internals.

This does not yet enable, out of the box,  access to APIs outside the Go standard library. Select “popular” libraries might be added to the canonical version; but it’s likely a mechanism will be added allowing easy configuration of additional libraries (packages, wrapped by Joker namespaces) to be included in a particular build of Joker. This (of course) would require rebuilding Joker, though not to contain substantial new Joker code just to provide decent, Clojure-like, access to those APIs.

Going forward, **gostd** might make some or all of those "low-level" APIs private (and thus largely undocumented by default, though the `generate-docs.joke` tool does now have an option to generate docs for private as well as public members of each namespace). It would then provide useful automatic generation of Clojure-like wrappers, using various heuristics, where feasible.

Such heuristics might (in at least some cases) eliminate the need to hand-write one’s own Joker libraries to wrap the **gostd**-generated ones more elegantly.

## Types

Named types (other than aliases), defined by the packages wrapped by the **gostd** tool, are similar to builtin Joker types (`Type` objects), but are always found in namespaces and are not "reserved" keywords.

For example, the `MX` type defined in the `net` package is type `go.std.net/MX`, which supports:

* Constructing a new instance: `(def mx (new go.std.net/MX {:Host "burleyarch.com" :Pref 10}))` => `&{burleyarch.com 10}`
* Identifying the type of an object: `(GoTypeOf @mx)` => `go.std.net/MX`
* Comparing types of objects: `(= (GoTypeOf mx) (GoTypeOf something-else))`

Each package-defined type has a reference (pointed-to) version that is also provided (e.g. `*MX`) in the namespace as well as an array-of version (e.g. `[]MX`, named `arrayOfMX`).

Some types have receivers. E.g. [`*go.std.os/File`](https://burleyarch.com/joker/docs/amd64-linux/go.std.os.html#*File) has a number of receivers, such as `Name`, `WriteString`, and `Close`, that maybe be invoked on it via e.g. `(. f Name)` (or `(.Name f)`), where `f` is (typically) returned from a call to `Create` or `Open` in the `go.std.os` namespace. `f` could be `Stdin`, for example, to take a snapshot, usually in the form of a `GoObject`, of the `GoObject` (global variable in Go's `go.std.os` package) named `Stdin`.

Methods on `interface{}` (abstract) types are now supported, though only some of them as of this writing. E.g. `(go.std.os/Stat "existing-file")` returns (inside the returned vector) a concrete type that is actually private to the Go library, so is not directly manipulatable via Clojure, but which also implements the [`go.std.os/FileInfo`](https://burleyarch.com/joker/docs/amd64-linux/go.std.os.html#FileInfo) abstract type. Accordingly, `(. fi ModTime)` works on such an object.

## Constants

All exported constants, defined in packages, are converted and thus available for reference. In some cases, their type is `BigInt` when an `Int` would suffice; this is due to their values being too large to fit in an `Int`.

However, besides `BigInt` being well-supported in normal Joker usage, certain conversions to native Go types (mainly `interface{}` in the arguments to `(format <pattern> <arguments...>)`) check to see whether a more-suitable native type (such as `int`) can precisely and accurately represent the value. If so, that conversion is performed, even in official Joker. An example of this effect herein is:

```
user=> (type go.std.hash.crc64/ISO)
BigInt
user=> (format "%x" go.std.hash.crc64/ISO)
"d800000000000000"
user=>
```

If `format` (or any other function that attempts such a conversion) cannot precisely and accurately coerce a non-native `Number`, such as a `BigInt`, `BigFloat`, or `Ratio`, to a suitable native type, it'll use the stringized form of the number, which might not be desirable:

```
user=> (def bignum 99999999999999999999999999999999N)
#'user/bignum
user=> (type bignum)
BigInt
user=> (format "%s" bignum)
"99999999999999999999999999999999N"
user=> (format "%x" bignum)
"39393939393939393939393939393939393939393939393939393939393939394e"
user=>
```

Here, the number is too large to convert to a native numeric Go type, so it is converted to a string. While that looks reasonable formatted by `%s`, the result of formatting it via `%x` is the hex form of the stringized version; that's probably not desirable.

### Floating-point Constants

Floating-point constants that are not accurately and precisely representable via `Double` are promoted to `BigFloat`. Compare the `gostd` rendition of _e_ with Joker's official version:

```
user=> joker.math/e
2.718281828459045
user=> go.std.math/E
2.71828182845904523536028747135266249775724709369995957496696763M
user=>
```

Further, `BigFloat`s created from strings (as is `math/big.E`, or when a constant such as `1.3M` is parsed by Joker) are given a _minimum_ precision of 53 (the same precision as a `Double`, aka `float64`) and a _maximum_ precision based on the number of digits and the number of bits each digit represents (3.3 for decimal; 1, 3, or 4 for binary, octal, and hex).

A new `joker.core/precision` function has been introduced mainly to inspect `BigFloat` types, though it supports a few others.

This combines to produce a fairly useful set of default behaviors:

```
user=> (def c1 1.3M)
#'user/c1
user=> (def c2 0000000000001.3000000000000M)
#'user/c2
user=> (precision c1)
53
user=> (precision c2)
86
user=> (= c1 c2)
false
user=> (+ c1 c2)
2.600000000000000044408921M
user=>
```

Note the inequality of the two values: when `c1` is binary-extended to match the precision of `c2`, it represents a slightly different value than does `c1`, as confirmed when adding the two values. This effect is not observed when using binary-based (binary, octal, or hexadecimal) encoding, since they encode the mantissa and thus the value precisely, which base-10 encoding (shown above) does not, due to the underlying representation using binary (rather than decimal) digits. E.g.:

```
user=> (= 0x1.fM 0x1.fM)
true
user=> (= 0x1.fM 0x1.f00000000000000000000000000M)
true
user=>
```

## Variables

Global variables (in Go packages) are implemented as `GoObject` objects wrapping references (pointers) to the global variables. When named in a context requiring their evaluation, they are automatically dereferenced.

As such, `SomeVar`, if a **Var** that wraps a global Go variable, yields a snapshot of that variable at the time of evaluation. Such snapshots are (per GoObject-creation rules) either `GoObject` or native Clojure wrappers (such as `Int` and `String`).

Use `(set! SomeVar new-value)` to assign `new-value` to that variable. `new-value` may be an ordinary object such as a `String`, `Int`, or `Boolean`; or it may be a `Var` or `GoObject`, in which case the underlying value is used (and potentially dereferenced once, if that enables assignment, though the original value is nevertheless returned by the function).

Or, use `(set! (.SomeField SomeVar) new-value)` to change the value of a specific field, when `SomeVar` is a `struct`.

## GoObjects

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

`(deref obj)` (or `@obj`) can be used to dereference a wrapped object, returning another `GoObject` with the dereferenced object as of the moment of that dereference. If the original object isn't a pointer, a panic results.

### Obtaining a Reference to a GoObject

This is TBD, as `ref` has been removed due to conflicting with Clojure's `ref` while being too dissimilar.

### Rules Governing GoObject Creation

When considering whether to wrap a given object in a `GoObject`, Joker normally substitutes a suitable Clojure type (such as `Int`, `Number`, or `String`) when one is available and suitable for the underlying type (not just the value). For example, instead of wrapping an `int64` in a `GoObject`, Joker will wrap it in a `Number`, even if the value is small (such as zero).

This substitution does not occur for non-builtin, named, types. Preserving the original type allows invocation of methods for the type.

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
user=> (def le (new LinkError {:Op "hey" :Old "there" :New "you" :Err "silly"}))
#'user/le
user=> le
hey there you: silly
user=> (type le)
GoObject
user=> (GoTypeOf le)
go.std.os/*LinkError
user=> (goobject? le)
true
user=> (goobject? "foo")
false
user=>
```

If a particular constructor is missing, that indicates lack of support for the underlying type, or that the underlying type is abstract (`interface{}`).

Note that, as in Clojure `(t. init)` is macro-expanded to `(new t init)`. However, unlike in Clojure, `new` is not a special form; and, as there are
no constructors (in the formal sense) in Go, the constructors described above are implemented by `gostd` for each supported type.

### Converting a GoObject to a Joker (Clojure) Datatype

Given a `GoObject`, one may convert (to a native Clojure type) and/or examine it via:
* `count`, which returns the number of elements (`.Len()`) for anything `seq` supports, without converting any of the elements themselves
* `deref`, which dereferences (indirects through the pointer wrapped by) the `GoObject` and returns the resulting "snapshot" of its value, either as a native Clojure object or (if one isn't suitable) a `GoObject`
* `get`, which returns the value corresponding to the given key for structs (the key must evaluate to a symbol), maps, arrays, slices, and strings; note, however, that a `GoObject` might be returned if a native Clojure object is not suitable
* `if`, `and`, `or`, and similar, which convert to `bool` (and all `GoObject`'s evaluate as `true`)
* `seq`, which works on arrays, channels, maps, slices, strings, and structs, but is (currently) not lazily evaluated
* `vec`, like `seq` but returns a vector instead of a sequence
* `=`, `not=`, `compare`, and similar, which compare via Go's `==` operator (though with no autoconversion to `Number` or similar types) and return a `bool` or `int` result

#### Converting a Struct to a Joker (Clojure) Datatype

As touched on above, `count`, `seq`, and `vec` operate on Go `struct` types (dereferencing them once if necessary). Explaining further:
* `count` returns the number of fields (which will therefore be the same value for a given struct's type, regardless of the values in the struct itself)
* `seq` returns `([key-1 value-1] [key-2 value-2] ... [key-N value-N])`, where `N` equals `(count ...)` for the same object, where `key-n` is a keyword named after the Go field name (typically capitalized, due to being public)
* `vec` is like `seq`, but returns a vector of key-value pairs

Though it might be faster to retrieve individual fields as needed, a Clojure map of field names (as keywords) to values can be constructed via e.g.:

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

However, Joker does support some implicit conversion of Clojure objects (such as `Int`) _to_ `GoObject`, in some ways beyond what the Go language itself provides, as explained below.

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

Clojure input arguments: [^String name, ^go.std.os/FileMode mode]

Clojure return type: Error
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

For receivers, given an object of the appropriate type, the `.` special form (specific to this version of Joker) is used, specifying the object, the name (as a symbol that is unevaluated) of the receiver, and any arguments:

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
user=> (. im Size)
[6 32]
user=> (. ip Equal ip)
true
user=> (.Equal ip im)
<joker.core>:4609:3: Eval error: Arg[0] (_v_x) of (net.IP)Equal() must have type GoObject[net.IP], got GoObject[net.IPMask]
Stacktrace:
  global <repl>:4:1
  core/Go <joker.core>:4609:3
user=>
```

Note the diagnostic produced when passing an object of incorrect type to a receiver, just as happens when passing the wrong thing to a standalone function.

Also note that Clojure's `(.receiver instance args*)` special form, which macroexpands to `(. instance receiver args*)`, is supported.

#### Returned Values

Multiple return values are converted to a (Clojure) vector of the arguments, each treated as its own return value as far as this section of the document is concerned.

Most package-defined types are returned as `GoObject` wrappers, while builtin types are returned as `String`, `Int`, `BigInt`, `Double`, or whatever is best suited to handle the range of possible return values.

Returned `GoObject` instances can:
* Be ignored (they'll presumably be garbage-collected at some point)
* Be stringized (via e.g. `(str goobj)`)
* Be converted to a suitable Clojure representation
* Be passed as arguments to Go API wrappers
* Be provided as members in a newly constructed `GoObject` instance (of the same or, more typically, some other, type)
* Have receivers/methods, defined on them, invoked via the `Go` function

Builtin type instances are converted directly to appropriate Joker (Clojure) types. For example, a Go API that returns `uint64` will be converted to a `Number` so as to ensure the full range of potential values is supported:


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

Clojure input arguments: []

Clojure return type: BigInt
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

Note that returned objects that are considered (by Go) to be `error` or `string` types, but are not the builtin forms of their types (for example, they're actually defined types or non-predeclared undefined types), are wrapped in `GoObject` as usual so that type-based actions (such as invoking receivers) work on them.

### Referencing a Member of a GoObject

#### Fields in Structures

`(. obj field)` returns the value of the named field, when `obj` denotes a structure (`struct` type in Go) wrapped by (underlying a) `GoObject`. (TBD?: `obj` may also wrap a `map[]`, array, slice, or string.) An alternate syntax, supported by Clojure as well, is `(.field obj)`.

If `field` is not a member of `obj` (that is, of the type of `obj`), a (try/catchable) panic results. Or, `(get obj 'field)` will return `nil` in such a situation, while `(get obj 'field not-found)` will return the value of `not-found`. (Note how the symbol name is quoted so it evalutes to a symbol, since `get` evaluates all its arguments; such quoting is neither necessary nor permitted for the `.` special form.)

The target of a field reference, as well as a global variable, can also be changed, as in Go's assignment (`=`) statement, via `(set! target newval)` (which returns `newval` as its result).

For example:

```
user=> (use 'go.std.os)
nil
user=> (def le (new LinkError {:Op "hi" :Old "there" :New "you" :Err "silly"}))
#'user/le
user=> (str le)
"hi there you: silly"
user=> (.Old le)
"there"
user=> (set! (.Old le) "golly"))
"golly"
user=> (str le)
"hi golly you: silly"
user=> (set! Stdout "whoa")
<repl>:7:1: Eval error: Cannot assign a string to a *os.File
user=> (use 'go.std.go.build)
nil
user=> Default
{amd64 darwin /usr/local/go /Users/craig/go  true false gc [] [go1.1 go1.2 go1.3 go1.4 go1.5 go1.6 go1.7 go1.8 go1.9 go1.10 go1.11 go1.12 go1.13 go1.14 go1.15 go1.16]  <nil> <nil> <nil> <nil> <nil> <nil> <nil>}
user=> (set! (.GOOS Default) "hey")
"hey"
user=> Default
{amd64 hey /usr/local/go /Users/craig/go  true false gc [] [go1.1 go1.2 go1.3 go1.4 go1.5 go1.6 go1.7 go1.8 go1.9 go1.10 go1.11 go1.12 go1.13 go1.14 go1.15 go1.16]  <nil> <nil> <nil> <nil> <nil> <nil> <nil>}
user=>
```

#### Receivers and Methods

`(. obj receiver [args...])`, where `obj` is a `GoObject`, calls `receiver` (an unevaluated symbol) for `obj` with the specified arguments. For examples:

```
user=> (use 'go.std.os)
nil
user=> (def file (get (Create "TEMP.txt") 0))
#'user/file
user=> file
&{0xc000c01b60}
user=> (. file Name)
"TEMP.txt"
user=> (. file WriteString "Hello, world!\n")
[14 nil]
user=> (. file Close)
nil
user=> (.Name file)
"TEMP.txt"
user=> (. file WriteString "Hello, world again!\n")
[0 "write TEMP.txt: file already closed"]
user=> (slurp "TEMP.txt")
"Hello, world!\n"
user=> (. Stdin Name)
"/dev/stdin"
user=>
```

# GOSTD Deeper Dive

## Developer Notes

The version of `run.sh` on this branch invokes `tools/gostd/gostd` to create the `go.std...` namespaces, generate wrappers, and so on.

Before building Joker by hand, one can optionally run the **gostd** tool against a Go source tree (the default is found via `go/build.Default.GOROOT`), which _must_ correspond to the version of Go used to build Joker itself (as it likely will). It contains a complete `src` subdirectory, which **gostd** walks and parses, in order to populate `std/gostd/` and modify related Joker source files. Further, the build parameters (`$GOARCH`, `$GOOS`, etc.) must match -- so `build-all.sh` would have to pass those to this tool (if it was to be used) for each of the targets.

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
<joker.core>:1466:3: Eval error: GoObject is not a reference (pointer) nor a GoObject wrapping a potential Joker object
Stacktrace:
  global <repl>:8:2
  core/deref <joker.core>:1466:3
user=> (def m0 (get mx 0))
#'user/m0
user=> m0
&{p25.llamail.com. 10}
user=> (type m0)
GoObject
user=> (GoTypeOf m0)
go.std.net/*MX
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
user=> (. m 0)
<repl>:14:7: Parse error: dot form member not an unqualified symbol
user=> (. m Pref)  ;; (get m 'Pref) would also work.
10
user=> (. m Host)
"p25.llamail.com."
user=>
```

### To Be Done and Work in Progress

Among things to do to "productize" this:

* Support explicit conversion (e.g. via constructors) of `Vector`, `Seq`, and such to `[]<type>` (and *vice versa*), though to `String` is already implemented
* Avoid generating unused code (`ReceiverArg_ns_*`, etc) to reduce executable size
* Support `func()` types (somehow)
* Support ad-hoc `map` types
* Support ad-hoc `struct{...}` types
* Create a `go.std.builtin` space with useful functions (not exported in the usual sense, but built in to the compiler) from the `builtin` package
* Automagically support all ad-hoc types, such as `[64]byte`, that appear in package source code, including "mashups" like `map[foo.TypeA][bar.TypeB]`, which can't really belong to one namespace or the other
* Support `deftype` (which currently is defined as a macro that supports only internal use by `gostd`) and the like
* Make `(new ...)` a special form, so it can't be shadowed
* Autogenerate Clojure-like APIs *a la* how they tend to be hand-written for Joker, so (for example) a low-level API returning `(string, error)` would be further wrapped by one, likely named with a lowercase initial letter, that returns merely `String` and panics if `error` came back non-`nil`
* Support wrapping arbitrary 3rd-party packages
* Support (perhaps via autogeneration) more-complicated types built on builtins, such as `[][]byte`, in constructors
* Support vectors (instead of only maps with keys) in constructors
* Improve docstrings for constructors (show and document the members)
* Consider promoting/lifting `GoObject` into `Object`, etc, if feasible and sufficiently useful
* Clean up and document the (mostly **gostd**) code better
* Assess performance impact (especially startup time) on Joker

### Evaluation Tests

A handful of tests (assertions) can be found in `tests/eval/go-objects.joke`. This is automatically run by the `eval-tests.sh` script (run in turn by `all-tests.sh`), available in "canonical" Joker.

`go-objects.joke` should be kept up-to-date as far as "smoke testing" basic capabilities. It should therefore be helpful as a guide as to which features are expected to work in a given version.

### Run gostd Tests

The `test.sh` script in `joker/tools/gostd/` runs tests against the full copy of Go's `golang/go/src/` tree. E.g.:

```
$ ./test.sh
$
```

After running the test, it uses `git diff` to compare the resulting `gosrc.gold` file with the checked-out version.

### Update Tests on Other Machines

The Go standard library is customized per system architecture and OS, and **gostd** picks up these differences via its use of Go's build-related packages. That's why `_tests/gold/` has a subdirectory for each combination of `$GOARCH` and `$GOOS`. Updating another machine's copy of the **gostd** repo is somewhat automated via `update.sh` -- e.g.:

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

After building the **gostd** branch, numerous additional files will have been created, and several existing files (including a few copied from elsewhere in the source distribution to become "stubs" for the first build of Joker) will have been modified.

Clean these out (essentially resetting to the base state) via:

```
$ ./clean.sh
```

This should result in `git` showing no differences (tracked nor untracked files) if only **gostd** has made changes to the source tree. If Joker hadn't previously been successfully built (if there's no `joker` executable in the top-level directory), there'll be a diagnostic; but the result should still be a "cleaned" tree.

When switching away from a **gostd** to a non-**gostd** branch (such as `master`), do this before attempting to build:

```
rm -fv g_* core/g_* core/data/g_*
rm -fr std/go*
```

That'll clean out the files left over from running **gostd**, which a non-**gostd** branch won't know how to clean up but which will likely get caught up in a subsequent `go build` command, leading to build errors. (There might still be numerous files left over in `docs/`, but those won't disturb a fresh build.)

### Caching of Core-API Information

To ease development, **gostd** dynamically determines the list of exported functions in Joker's `core` package, and avoids generating calls to unlisted functions. This helps to catch missing APIs earlier in the development process (mainly, while building and testing **gostd** in isolation, versus requiring a build of Joker itself).

The resulting list is cached in `./core-apis.dat`, though currently not reused by `./test.sh` to save time (which amounts to a substantial portion of the time it takes to build and test **gostd**).

As building Joker takes enough longer to make this caching less useful, `run.sh` deletes the cache prior to each build.

### Caching of Working Joker Version

To save time when building Joker, either an environment variable `$JOKER` specifying the path to a working Joker executable, or a copy of (or link to) such an executable named `joker-good` (in the top-level Joker directory), can be provided.

This causes `run.sh` to skip building an initial version of Joker, and instead use that existing version to run `std/generate-std.joke` and then decide whether it is necessary to rebuild Joker with the resulting libraries (as will always be the case when **gostd** is involved).

## FUTURE: Including Other Go Packages

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

## Design Considerations

### Why GoObject (Versus Object)?

A `GoObject` has a single member:
* `O interface{}`

While this approach has eased development somewhat, especially early on, it seems likely that `GoObject` will (like `GoType` and `GoVar`) someday be eliminated in favor of named types in all cases where said types are supported (as input and output arguments, and as members of constructable types).

There would presumably be a `go.std.builtin/interface` type by then, which would be the semantic equivalent of `GoObject`, without being "special" nor widely used. Core functions such as `goobject?` would be removed.

### Why GoReceiver?

`GoReceiver` is of type `func(GoObject, Object) Object`, connecting a `(. obj receiver args*)` call to a concrete implementation. It implements `Object` so an instance can be stored in the `.Value` field of a `Var`.

## Earlier Design Changes

### 2021-04-19 (v0.12.1)

The `GoType` object no longer exists and is thus no longer generated by `gostd`. All types become `Type` objects, just like the builtin Joker types, also supporting constructors and lists of methods and receivers.

### 2021-04-07

The dot (`.`) special form, `(. <instance> <member> <args>*)`, is implemented and the corresponding
functionality of the `joker.core/Go` function (invoking methods/receivers and reference fields)
has been removed. The `set!` special operator is not implemented; instead, use `(var ...)` to wrap
the special form, yielding a `GoVar` that can then be the first operand of a `var-set`.
(The `joker.core/Go` function, now private, returned a `var` for a field, rather than the value of
field, requiring a `deref` to retrieve the value.)

The `(.<member> <instance> <args>*)` macroexpansion to `(. <instance> <member> <args>*)` is
also implemented.

The `(t. init)` macroexpansion to `(new t init)` is implemented.

`(get obj key)` on a GoObject wrapping a `struct{}` now requires `key` to evaluate to a symbol.
Instead of `(get obj 'SomeKey)`, however, just use `(. obj SomeKey)` or `(.SomeKey obj)`.

### 2021-04-01

All (exported) constants are now wrapped; evaluation of constant expressions is now done via `go/types`, `go/importer`, and `go/constant`, rather than via custom code in `gostd`.

Constant types are now preserved. E.g. `go.std.net/FlagUp` is of type `net.Flags`, not `Number`, so this now works (as `go.std.net/Flags` is the type for `FlagUp` and defines a `String()` method for it and other constants), instead of just `1N` being printed:

```
user=> go.std.net/FlagUp
up
user=> (int go.std.net/FlagUp)
1
user=>
```

Constants and variables whose names match Joker type names are no longer renamed away. Use explicit namespace resolution to specify them; or, if referring a namespace that shadows a Joker type (which is not recommended), quote the type when using it as a tag. E.g.:

```
user=> Int
Int
user=> (type Int)
Type
user=> go.std.go.constant/Int
3
user=> (use 'go.std.go.constant)
nil
user=> Int
3
user=> (type Int)
GoObject
user=> (defn ^Int foo [])
#'user/foo
user=> (meta (var foo))
{:line 12, :column 1, :file "<repl>", :ns user, :name foo, :tag 3, :arglists ([])}
user=> (defn ^"Int" foo [])
#'user/foo
user=> (meta (var foo))
{:line 14, :column 1, :file "<repl>", :ns user, :name foo, :tag "Int", :arglists ([])}
user=>
```

Note that the first argument to `catch` in a `try` is known to specify a type, so the builtin Joker types will be checked first, as will likely be desirable (since `catch` does not yet support namespace-based `Type` objects). E.g.:

```
user=> (use 'go.std.go.scanner)
nil
user=> Error
go.std.go.scanner/Error
user=> (type Error)
Type
user=> (try true (catch Error e))
true
user=> (try true (catch go.std.go.scanner/Error e))
<repl>:26:18: Parse error: Unable to resolve type: go.std.go.scanner/Error, got: {:type :var-ref, :var #'go.std.go.scanner/Error}
user=>
```

The `BigFloat` type is now supported for `std`-generated libraries (namespaces in the `std` subdirectory and built into Joker).

Further, `BigFloat`s created from strings (such as `1.3M`) are given a minimum precision of 53 (the same precision as a `Double`, aka `float64`) or more precision based on the number of digits and the number of bits each digit represents (3.3 for decimal; 1, 3, or 4 for binary, octal, and hex).

All arrays with constant lengths are now supported. Previously only easily-evaluated lengths (such as literal constants) were supported; now, any constant expression works. This results in more functions being wrapped.

A new `joker.core/precision` function has been introduced mainly to inspect `BigFloat` types, though it supports a few others.

### 2021-03-24

tl;dr: As of now, this fork is verging on usefulness!

`...` arguments are now supported for functions, methods, and receivers, unless the underlying type is any other type Joker decides to pass by reference (currently these are all non-empty `struct{}` types; as a result, `reflect.Append()` is unsupported). E.g.:

```
user=> (go.std.fmt/Println (go.std.net/IPv4 1 2 3 4) (go.std.net/IPv4Mask 255 255 255 127))
1.2.3.4 ffffff7f
[17 nil]
user=>
```

`^GoObject` arguments are extended to support many native Joker types via their underlying (`Native`) types. E.g.:

```
user=> (go.std.fmt/Println 1 2 3 4)
1 2 3 4
[8 nil]
user=>
```

`[]byte` arguments and fields are now supported in function calls and ctors, and also support automatic conversion from strings (e.g. an object of type `String`). There are more conversions like this to come, but this is a nice proof-of-concept, as it enables (for example) sending an email to an SMTP server.

A `Vector` or `Seq` may be provided in lieu of a `String` (this might be too general!). E.g.:

```
user=> (go.std.fmt/Printf [65 66 67 10])
ABC
[4 nil]
user=>
```

Got `chan` and (empty) `struct{}` types working at a rudimentary level.

Receivers that have no return values are now implemented; their Joker wrappers return `NIL`, just as for regular functions and methods.

Types defined (recursively) in terms of builtin types are now pass-by-value (and construct-by-value), as is `struct{}`.

Note that constructors for pass-by-reference types return reference types. E.g. `(new Foo ...)` returns `refToFoo`. Assuming the Go code defines methods on `*Foo`, they should work on the result; else, one should be able to dereference and then invoke them, as in `(Go (deref my-foo) :SomeMethod ...)`.

Though some namespaces redefine (shadow) built-in types such as `String` and `Error`, `(catch <type> ...)` now first checks *type* to see whether it is a symbol that names one of the built-in types (and does not specify a namespace). If so, that type is used, rather than an error resulting.

### 2021-03-07

The `:gostd` reader conditional has been introduced, primarily to allow `docs/generate-docs.joke` to work on any version of Joker. E.g. `#?(:gostd ...)` will process the `...` only if run by a (recent version of) this **gostd** fork of Joker. It's unlikely to have the same name by the time this fork gains some sort of official status (well beyond proof-of-concept).

Incompatible types for arguments passed to receivers no longer crash Joker; they can be caught, and otherwise produce reasonable diagnostics, just as occurs with normal functions. E.g. `./joker -e '(Go (new go.std.os/File {}) :Write "hi there")'` no longer crashes Joker.

`map`, `struct` (except empty `struct{}`), and `func` types, as well as `...` in argument lists, are explicitly disabled for now.

The build process no longer changes files known to Git; generated files, whose names are all prefixed with `g_`, start out as stubs copied in by `run.sh` and are then overwritten by `tools/gostd/gostd` later by that script.

The **gostd** tool itself has been simplified, including replacing some substantial `fmt.Sprintf` calls (which can be difficult to maintain) with use of templates (via Go's `text/template` package), which are kept in `tools/gostd/templates/`. More work is expected in this area (simplification as well as templatization) in the future.

### 2020-12-17

Type aliases (e.g. `type foo = bar`) are now ignored, so their Clojure names are not available for use.

Their use (in the Go standard library) seems to have been introduced (with respect to the timeframe in which **gostd** has existed) in Go version `1.16.1beta1`.

Ideally, **gostd** would support them, but as they seem designed for only short-term use around refactoring, properly implementing them in Joker seems less urgent than numerous other matters.

For more information on type aliases, see [Proposal: Type Aliases](https://go.googlesource.com/proposal/+/master/design/18130-type-alias.md) by Russ Cox and Robert Griesemer, December 16, 2016.

### 2020-04-17

Values returned by functions (this includes receivers and methods) are now returned "as-is", rather than (for some types) autoconverted to suitable Joker representations.

For example, calling a Go function returning `[]string` now returns a `GoObject` wrapping that same object (which has a type named `arrayOfstring`, since `[` and `]` are invalid symbol characters), rather than a vector of String objects.

Use `(vec ...)` to perform an explicit conversion, in this example, as `GoObject`s that wrap appropriate types are `Seqable` and thus support `(count ...`), `(rest ...)`, and so on.

This change improves performance in cases where the returned value will be used as-is, or only limited information (such as a given element or the number of elements) is needed, by Clojure code, and where the number of returned elements (or their individual elements) is large.

*Note:* Vectors are still returned when the called Go function returns multiple arguments, since Go does not define multiple arguments as a single type.
