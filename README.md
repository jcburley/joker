<img src="https://user-images.githubusercontent.com/882970/48048842-a0224080-e151-11e8-8855-642cf5ef3fdd.png" width="117px"/>

[![CircleCI](https://circleci.com/gh/candid82/joker.svg?style=svg)](https://circleci.com/gh/candid82/joker)

# Joker

Joker is a small Clojure interpreter, linter, and formatter written in Go.

[This **gostd** experimental fork](https://github.com/jcburley/joker/) extends official Joker by reading Golang source code (the Go standard library) and "wrapping" some of their functions, types, constants, and variables so that code written in Joker can access them. See [below](#gostd) for information on the [`go.std.*` namespaces](https://burleyarch.com/joker/docs/) thereby provided.

## Installation

On macOS, the easiest way to install Joker is via Homebrew:

```
brew install candid82/brew/joker
```

The same command can be used on Linux if you use [Linuxbrew](http://linuxbrew.sh/).

If you use Arch Linux, there is [AUR package](https://aur.archlinux.org/packages/joker-bin/).

If you use [Nix](https://nixos.org/nix/), then you can install Joker with

```
nix-env -i joker
```

On other platforms (or if you prefer manual installation), download a [precompiled binary](https://github.com/candid82/joker/releases) for your platform and put it on your PATH.

You can also [build](#building) Joker from the source code.

## Usage

`joker` - launch REPL. Exit via `(exit)`, **EOF** (such as `Ctrl-D`), or **SIGINT** (such as `Ctrl-C`).

`joker <filename>` - execute a script. Joker uses `.joke` filename extension. For example: `joker foo.joke`. Normally exits after executing the script, unless `--exit-to-repl` is specified before `--file <filename>`
in which case drops into the REPL after the script is (successfully) executed. (Note use of `--file` in this case, to ensure `<filename>` is not treated as a `<socket>` specification for the repl.)

`joker --eval <expression>` - execute an expression. For example: `joker -e '(println "Hello, world!")'`. Normally exits after executing the script, unless `--exit-to-repl` is specified before `--eval`,
in which case drops into the REPL after the expression is (successfully) executed.

`joker -` - execute a script on standard input (os.Stdin).

`joker --lint <filename>` - lint a source file. See [Linter mode](#linter-mode) for more details.

`joker --lint --working-dir <dirname>` - recursively lint all Clojure files in a directory.

`joker --format <filename>` - format a source file and write the result to standard output. See [Format mode](#format-mode) for more details.

`joker --format -` - read Clojure source code from standard input, format it and print the result to standard output.

## Documentation

[Standard library reference](https://candid82.github.io/joker/)

Dash docset: `dash-feed://https%3A%2F%2Fburleyarch.com%2Fjoker%2Fdocs%2Fjoker.xml`

(either copy and paste this link to your browser's url bar or open it in a terminal with `open` command)

[Joker slack channel](https://clojurians.slack.com/messages/C9VURUUNL/)

[Organizing libraries (namespaces)](LIBRARIES.md)

[Developer notes](DEVELOPER.md)

## Project goals

These are high level goals of the project that guide design and implementation decisions.

- Be suitable for scripting (lightweight, fast startup). This is something that Clojure is not good at and my personal itch I am trying to scratch.
- Be user friendly. Good error messages and stack traces are absolutely critical for programmer's happiness and productivity.
- Provide some tooling for Clojure and its dialects. Joker has [linter mode](#linter-mode) which can be used for linting Joker, Clojure and ClojureScript code. It catches some basic errors.
  Joker can also format (pretty print) Clojure code (see [format mode](#format-mode)) or EDN data structures. For example, the following command can be used to pretty print EDN data structure (read from stdin):

```
joker --hashmap-threshold -1 -e "(pprint (read))"
```

There is [Sublime Text plugin](https://github.com/candid82/sublime-pretty-edn) that uses Joker for pretty printing EDN files. [Here](https://github.com/candid82/joker/releases/tag/v0.8.8) you can find the description of `--hashmap-threshold` parameter, if curious.

- Be as close (syntactically and semantically) to Clojure as possible. Joker should truly be a dialect of Clojure, not a language inspired by Clojure. That said, there is a lot of Clojure features that Joker doesn't and will never have. Being close to Clojure only applies to features that Joker does have.

## Project Non-goals

- Performance. If you need it, use Clojure. Joker is a naive implementation of an interpreter that evaluates unoptimized AST directly. I may be interested in doing some basic optimizations but this is definitely not a priority.
- Have all Clojure features. Some features are impossible to implement due to a different host language (Go vs Java), others I don't find that important for the use cases I have in mind for Joker. But generally Clojure is a pretty large language at this point and it is simply unfeasible to reach feature parity with it, even with naive implementation.

## Differences with Clojure

1. Primitive types are different due to a different host language and desire to simplify things. Scripting doesn't normally require all the integer and float types, for example. Here is a list of Joker's primitive types:

| Joker type | Corresponding Go type |
| ---------- | --------------------- |
| BigFloat   | big.Float (see below) |
| BigInt     | big.Int               |
| Boolean    | bool                  |
| Char       | rune                  |
| Double     | float64               |
| Int        | int                   |
| Keyword    | n/a                   |
| Nil        | n/a                   |
| Ratio      | big.Rat               |
| Regex      | regexp.Regexp         |
| String     | string                |
| Symbol     | n/a                   |
| Time       | time.Time             |

See [Floating-point Constants and the BigFloat Type](docs/misc/bigfloat.md) for more on `BigFloat` (`M`-suffixed) constants.

Note that `Nil` is a type that has one value: `nil`.

1. The set of persistent data structures is much smaller:

| Joker type | Corresponding Clojure type                                                                                |
| ---------- | --------------------------------------------------------------------------------------------------------- |
| ArrayMap   | PersistentArrayMap                                                                                        |
| MapSet     | PersistentHashSet (or hypothetical PersistentArraySet, depending on which kind of underlying map is used) |
| HashMap    | PersistentHashMap                                                                                         |
| List       | PersistentList                                                                                            |
| Vector     | PersistentVector                                                                                          |

1. Joker doesn't have the same level of interoperability with the host language (Go) as Clojure does with Java or ClojureScript does with JavaScript. It doesn't have access to arbitrary Go types and functions. There is only a small fixed set of built-in types and interfaces. Dot notation for calling methods is not supported (as there are no methods). All Java/JVM specific functionality of Clojure is not implemented for obvious reasons.
1. Joker is single-threaded with no support for parallelism. Therefore no refs, agents, futures, promises, locks, volatiles, transactions, `p*` functions that use multiple threads. Vars always have just one "root" binding. Joker does have core.async style support for concurrency. See `go` macro [documentation](https://candid82.github.io/joker/joker.core.html#go) for details.
1. The following features are not implemented: protocols, records, structmaps, chunked seqs, transients, tagged literals, unchecked arithmetics, primitive arrays, custom data readers, transducers, validators and watch functions for vars and atoms, hierarchies, sorted maps and sets.
1. Unrelated to the features listed above, the following function from clojure.core namespace are not currently implemented but will probably be implemented in some form in the future: `subseq`, `iterator-seq`, `reduced?`, `reduced`, `mix-collection-hash`, `definline`, `re-groups`, `hash-ordered-coll`, `enumeration-seq`, `compare-and-set!`, `rationalize`, `load-reader`, `find-keyword`, `comparator`, `resultset-seq`, `file-seq`, `sorted?`, `ensure-reduced`, `rsubseq`, `pr-on`, `seque`, `alter-var-root`, `hash-unordered-coll`, `re-matcher`, `unreduced`.
1. Built-in namespaces have `joker` prefix. The core namespace is called `joker.core`. Other built-in namespaces include `joker.string`, `joker.json`, `joker.os`, `joker.base64` etc. See [standard library reference](https://candid82.github.io/joker/) for details.
1. Joker doesn't support AOT compilation and `(-main)` entry point as Clojure does. It simply reads s-expressions from the file and executes them sequentially. If you want some code to be executed only if the file it's in is passed as `joker` argument but not if it's loaded from other files, use `(when (= *main-file* *file*) ...)` idiom. See https://github.com/candid82/joker/issues/277 for details.
1. Miscellaneous:

- `case` is just a syntactic sugar on top of `condp` and doesn't require options to be constants. It scans all the options sequentially.
- `slurp` only takes one argument - a filename (string). No options are supported.
- `ifn?` is called `callable?`
- Map entry is represented as a two-element vector.
- resolving unbound var returns `nil`, not the value `Unbound`. You can still check if the var is bound with `bound?` function.

## Linter mode

To run Joker in linter mode pass `--lint --dialect <dialect>` flag, where `<dialect>` can be `clj`, `cljs`, `joker` or `edn`. If `--dialect <dialect>` is omitted, it will be set based on file extension. For example, `joker --lint foo.clj` will run linter for the file `foo.clj` using Clojure (as opposed to ClojureScript or Joker) dialect. `joker --lint --dialect cljs -` will run linter for standard input using ClojureScript dialect. Linter will read and parse all forms in the provided file (or read them from standard input) and output errors and warnings (if any) to standard output (for `edn` dialect it will only run read phase and won't parse anything). Let's say you have file `test.clj` with the following content:

```clojure
(let [a 1])
```

Executing the following command `joker --lint test.clj` will produce the following output:

```
test.clj:1:1: Parse warning: let form with empty body
```

The output format is as follows: `<filename>:<line>:<column>: <issue type>: <message>`, where `<issue type>` can be `Read error`, `Parse error`, `Parse warning` or `Exception`.

### Integration with editors

- Emacs: [flycheck syntax checker](https://github.com/candid82/flycheck-joker)
- Sublime Text: [SublimeLinter plugin](https://github.com/candid82/SublimeLinter-contrib-joker)
- Atom: [linter-joker](https://atom.io/packages/linter-joker)
- Vim: [syntastic-joker](https://github.com/aclaimant/syntastic-joker), [ale](https://github.com/w0rp/ale)
- VSCode: [VSCode Linter Plugin (alpha)](https://github.com/martinklepsch/vscode-joker-clojure-linter)
- Kakoune: [clj-kakoune-joker](https://github.com/w33tmaricich/clj-kakoune-joker)

[Here](https://github.com/candid82/SublimeLinter-contrib-joker#reader-errors) are some examples of errors and warnings that the linter can output.

### Reducing false positives

Joker lints the code in one file at a time and doesn't try to resolve symbols from external namespaces. Because of that and since it's missing some Clojure(Script) features it doesn't always provide accurate linting. In general it tries to be unobtrusive and error on the side of false negatives rather than false positives. One common scenario that can lead to false positives is resolving symbols inside a macro. Consider the example below:

```clojure
(ns foo (:require [bar :refer [def-something]]))

(def-something baz ...)
```

Symbol `baz` is introduced inside `def-something` macro. The code is totally valid. However, the linter will output the following error: `Parse error: Unable to resolve symbol: baz`. This is because by default the linter assumes external vars (`bar/def-something` in this case) to hold functions, not macros. The good news is that you can tell Joker that `bar/def-something` is a macro and thus suppress the error message. To do that you need to add `bar/def-something` to the list of known macros in Joker configuration file. The configuration file is called `.joker` and should be in the same directory as the target file, or in its parent directory, or in its parent's parent directory etc up to the root directory. When reading from stdin Joker will look for a `.joker` file in the current working directory. The `--working-dir <path/to/file>` flag can be used to override the working directory that Joker starts looking in. Joker will also look for a `.joker` file in your home directory if it cannot find it in the above directories. The file should contain a single map with `:known-macros` key:

```clojure
{:known-macros [bar/def-something foo/another-macro ...]}
```

Please note that the symbols are namespace qualified and unquoted. Also, Joker knows about some commonly used macros (outside of `clojure.core` namespace) like `clojure.test/deftest` or `clojure.core.async/go-loop`, so you won't have to add those to your config file.

Joker also allows you to specify symbols that are introduced by a macro:

```clojure
{:known-macros [[riemann.streams/where [service event]]]}
```

So each element in :known-macros vector can be either a symbol (as in the previous example) or a vector with two elements: macro's name and a list of symbols introduced by this macro. This allows to avoid symbol resolution warnings in macros that intern specific symbols implicitly.

Additionally, if you want Joker to ignore some unused namespaces (for example, if they are required for their side effects) you can add the `:ignored-unused-namespaces` key to your `.joker` file:

```clojure
{:ignored-unused-namespaces [foo.bar.baz]}
```

Sometimes your code may refer to a namespace that is not explicitly required in the same file. This is rarely needed, but if you face such situation you can add that namespace to `:known-namespaces` list to avoid "No namespace found" or "Unable to resolve symbol" warnings:

```clojure
{:known-namespaces [clojure.spec.gen.test]}
```

If your code uses tagged literals that Joker doesn't know about, add them to `:known-tags` list:

```clojure
{:known-tags [db/fn]}
```

If you use `:refer :all` Joker won't be able to properly resolve symbols because it doesn't know what vars are declared in the required namespace (i.e. `clojure.test`). There are generally three options here:

1. Refer specific symbols. For example: `[clojure.test :refer [deftest testing is are]]`. This is usually not too tedious, and you only need to do it once per file.
2. Use alias and qualified symbols:

```clojure
(:require [clojure.test :as t])
(t/deftest ...)
```

3. "Teach" Joker declarations from referred namespace. Joker executes the following files (if they exist) before linting your file: `.jokerd/linter.cljc` (for both Clojure and ClojureScript), `.jokerd/linter.clj` (Clojure only), `.jokerd/linter.cljs` (ClojureScript only), or `.jokerd/linter.joke` (Joker only). The rules for locating `.jokerd` directory are the same as for locating `.joker` file.

   - :warning: Joker can be made aware of any additional declarations (like `deftest` and `is`) by providing them in `.jokerd/linter.clj[s|c]` files. However, this means Joker cannot check that the symbols really are declared in your namespace, so this feature should be used sparingly.
   - If you really want some symbols to be considered declared _in any namespace no matter what_, you can add `(in-ns 'joker.core)` to your `linter.clj[s|c]` and then declare those symbols.
     (see issues [52](https://github.com/candid82/joker/issues/52) and [50](https://github.com/candid82/joker/issues/50) for discussion).

I generally prefer first option for `clojure.test` namespace.

### Linting directories

To recursively lint all files in a directory pass `--working-dir <dirname>` parameter. Please note that if you also pass file argument (or `--file` parameter) Joker will lint that single file and will only use `--working-dir` to locate `.joker` config file. That is,

```bash
joker --lint --working-dir my-project
```

lints all Clojure files in `my-project` directory, whereas

```bash
joker --lint --working-dir my-project foo.clj
```

lints single file `foo.clj` but uses `.joker` config file from `my-project` directory.

When linting directories Joker lints all files with the extension corresponding to the selected dialect (`*.clj`, `*.cljs`, `*.joke`, or `*.edn`). To exclude certain files specify regex patterns in `:ignored-file-regexes` vector in `.joker` file, e.g. `:ignored-file-regexes [#".*user\.clj" #".*/dev/profiling\.clj"]`.

When linting directories Joker can report globally unused namespaces and public vars. This is turned off by default but can be enabled with `--report-globally-unused` flag, e.g. `joker --lint --working-dir my-project --report-globally-unused`. This is useful for finding "dead" code. Some namespaces or vars are intended to be used by external systems (e.g. public API of a library or main function of a program). To exclude such namespaces and vars from being reported as globally unused list them in `:entry-points` vector in `.joker` file, which may contain the names of namespaces or fully qualified names of vars. For example:

```clojure
{:entry-points [my-project.public-api
                my-project.core/-main]}
```

### Optional rules

Joker supports a few configurable linting rules. To turn them on or off set their values to `true` or `false` in `:rules` map in `.joker` file. For example:

```clojure
{:rules {:if-without-else true
         :no-forms-threading false}}
```

Below is the list of all configurable rules.

| Rule                   | Description                                           | Default value |
| ---------------------- | ----------------------------------------------------- | ------------- |
| `if-without-else`      | warn on `if` without the `else` branch                | `false`       |
| `no-forms-threading`   | warn on threading macros with no forms, i.e. `(-> a)` | `true`        |
| `unused-as`            | warn on unused `:as` binding                          | `true`        |
| `unused-keys`          | warn on unused `:keys`, `:strs`, and `:syms` bindings | `true`        |
| `unused-fn-parameters` | warn on unused fn parameters                          | `false`       |
| `fn-with-empty-body`   | warn on fn form with empty body                       | `true`        |

Note that `unused binding` and `unused parameter` warnings are suppressed for names starting with underscore.

### Valid Identifiers

Symbols and keywords (collectively referred to herein as "identifiers") can be comprised of nearly any encodable character ("rune" in Go), especially when composed from a `String` via e.g. `(symbol "arbitrary-string")`.

Unlike most popular programming languages, Clojure allows "extreme flexibility" (as does Joker) in choosing characters for identifiers in source code, permitting many control and other invisible characters, even as the first character. In short, any character not specifically allocated to another purpose (another lexeme) by the Clojure language defaults to starting or continuing an identifier lexeme: `(def ^@ "test")`, where `^@` denotes the ASCII `NUL` (`0x00`) character, works.

When _linting_ an identifier (versus composing one at runtime), Joker ensures its characters are members of a more "reasonable" set, aligned with those used by the core libraries of Clojure (as well as Joker).

This "core set" of characters, as a Regex, is `#"[a-zA-Z0-9*+!?<=>&_.'-]"`. It represents the intersection of a limited set of letters, digits, symbols, and punctuation within the (7-bit) ASCII encoding range. The letters are the ASCII-range members of Unicode category L, while the digits are the ASCII-range members of category Nd.

Thus, Joker will warn about using an em dash (instead of an ASCII hyphen-minus (`0x2D`)), a non-breaking space (`&nbsp;` in HTML), an accented letter (e.g. `é`), or a control character (even `NUL`), in an identifier.

The `.joker` file may specify key/value pairs that change this default:

| Key               | Value      | Meaning                                           |
| ----------------- | ---------- | ------------------------------------------------- |
| `:character-set`  | `:core`    | `#"[*+!?<=>&_.'\-$:#%]"` plus categories L and Nd |
|                   | `:symbol`  | `:core` plus symbols (category S)                 |
|                   | `:visible` | `:symbol` plus punctuation (P) and marks (M)      |
|                   | `:any`     | any category                                      |
| `:encoding-range` | `:ascii`   | only 7-bit ASCII (`<= unicode.MaxASCII`)          |
|                   | `:unicode` | only Unicode (`<= unicode.MaxRune`)               |
|                   | `:any`     | any encodable character                           |

The intersection of these specifications governs how identifiers are linted; any character outside the resulting set yields a linter warning.

If `:valid-ident` is not fully specified, the defaults are the core character set in the ASCII range, as if `.joker` contained:

```clojure
{:valid-ident {:character-set :core
               :encoding-range :ascii}}
```

Changing `:core` to `:symbol` would allow, for example, `|` in identifiers; whereas changing `:ascii` to `:unicode` would allow `é`.

## Format mode

To run Joker in format mode pass `--format` flag. For example:

`joker --format <filename>` - format a source file and write the result to standard output.

`joker --format -` - read Clojure source code from standard input, format it and print the result to standard output.

### Integration with editors

- Sublime Text: [sublime-pretty-clojure](https://github.com/candid82/sublime-pretty-clojure) - formats Clojure code when saving the file.

## Building

Joker requires Go v1.13 or later.
Below commands should get you up and running.

```
go get -d github.com/candid82/joker
cd $GOPATH/src/github.com/candid82/joker
./run.sh --version && go install
```

### Cross-platform Builds

After building the native version (to autogenerate appropriate files, "vet" the source code, etc.), set the appropriate environment variables and invoke `go build`. E.g.:

```
$ GOOS=linux GOARCH=arm GOARM=6 go build
```

The `run.sh` script does not support cross-platform building directly, but can be used in conjunction with `build-arm.sh` to cross-build from a Linux **amd64** or **386** system to a Linux **arm** system via:

```
$ ./run.sh --version && ./build-arm.sh
```

Note that cross-building from 64-bit to 32-bit machines (and _vice versa_) is problematic due to the `gen_code` step of building Joker. This step builds a faster-startup version of Joker than
was built in earlier versions, prior to the introduction of `gen_code`. It does this by building much of Joker code into `gen_code` itself, running the (expensive) dynamic-initialization
code to build up core namespaces from (nearly) scratch, then using reflection to discover the resulting data structures and output their contents as Go code that (mostly) statically recreates
them when built into the Joker executable itself. (See `DEVELOPER.md` for more information on `gen_code`.)

As types such as `int` are 32-bit on 32-bit machines, and 64-bit on 64-bit machines, the final Joker executable must be built with code generated by a same-word-size build of `gen_code`. Otherwise,
compile-time errors might well result due to too-large integers; or, run-time errors might result due to too-small integers.

Since Linux (on **amd64**) supports building _and running_ 32-bit (**386**) executables, it's a good candidate for cross-building to 32-bit architectures such as **arm**.

## Coding Guidelines

- Dashes (`-`) in namespaces are not converted to underscores (`_`) by Joker, so (unlike with Clojure) there's no need to name `.joke` files accordingly.
- Avoid `:refer :all` and the `use` function, as that reduces the effectiveness of linting.

## <a name="gostd"></a>The go.std.* Namespaces

On this experimental branch, Joker is built along with the results of an automated analysis of the Golang source directory in order to pull in and "wrap" functions, types, constants, and variables provided by Go `std` packages.

NOTE: Only Joker versions >= 0.16 are now supported by this branch.

### Quick Start

To make this "magic" happen:

1. Ensure you can build the "canonical" version of Joker
1. `go get -u -d github.com/candid82/joker` (This will download and update dependent packages as well, but not build Joker itself.)
1. `cd $GOPATH/src/github.com/candid82/joker`
1. `git remote add gostd git@github.com:jcburley/joker.git`
1. `git fetch gostd`
1. `git checkout gostd`
1. `./run.sh`, specifying optional args such as `--version`, `-e '(println "i am here")'`, or even:

```
-e "(require '[go.std.net :as n]) (print \"\\nNetwork interfaces:\\n  \") (n/Interfaces) (println)"
```
8. `./joker` invokes the just-built Joker executable.
1. `go install` installs Joker (and deletes the local copy).

### Sample Usage

Assuming Joker has been built as described above:

```
$ ./joker
Welcome to joker v0.17.2-gostd. Use '(exit)', EOF (Ctrl-D), or SIGINT (Ctrl-C) to exit.
user=> (require '[go.std.net :as n])
nil
user=> (sort (map #(key %) (ns-map 'go.std.net)))
(*AddrError *Buffers *DNSConfigError *DNSError *Dialer *Flags *HardwareAddr *IP *IPAddr *IPConn *IPMask *IPNet *Interface *InvalidAddrError *ListenConfig *MX *NS *OpError *ParseError *Resolver *SRV *TCPAddr *TCPConn *TCPListener *UDPAddr *UDPConn *UnixAddr *UnixConn *UnixListener *UnknownNetworkError Addr AddrError Buffers CIDRMask Conn DNSConfigError DNSError DefaultResolver Dial DialIP DialTCP DialTimeout DialUDP DialUnix Dialer ErrClosed ErrWriteToConnected Error FileConn FileListener FilePacketConn FlagBroadcast FlagLoopback FlagMulticast FlagPointToPoint FlagUp Flags HardwareAddr IP IPAddr IPConn IPMask IPNet IPv4 IPv4Mask IPv4allrouter IPv4allsys IPv4bcast IPv4len IPv4zero IPv6interfacelocalallnodes IPv6len IPv6linklocalallnodes IPv6linklocalallrouters IPv6loopback IPv6unspecified IPv6zero Interface InterfaceAddrs InterfaceByIndex InterfaceByName Interfaces InvalidAddrError JoinHostPort Listen ListenConfig ListenIP ListenMulticastUDP ListenPacket ListenTCP ListenUDP ListenUnix ListenUnixgram Listener LookupAddr LookupCNAME LookupHost LookupIP LookupMX LookupNS LookupPort LookupSRV LookupTXT MX NS OpError PacketConn ParseCIDR ParseError ParseIP ParseMAC Pipe ResolveIPAddr ResolveTCPAddr ResolveUDPAddr ResolveUnixAddr Resolver SRV SplitHostPort TCPAddr TCPConn TCPListener UDPAddr UDPConn UnixAddr UnixConn UnixListener UnknownNetworkError arrayOfAddr arrayOfAddrError arrayOfBuffers arrayOfConn arrayOfDNSConfigError arrayOfDNSError arrayOfDialer arrayOfError arrayOfFlags arrayOfHardwareAddr arrayOfIP arrayOfIPAddr arrayOfIPConn arrayOfIPMask arrayOfIPNet arrayOfInterface arrayOfInvalidAddrError arrayOfListenConfig arrayOfListener arrayOfMX arrayOfNS arrayOfOpError arrayOfPacketConn arrayOfParseError arrayOfResolver arrayOfSRV arrayOfTCPAddr arrayOfTCPConn arrayOfTCPListener arrayOfUDPAddr arrayOfUDPConn arrayOfUnixAddr arrayOfUnixConn arrayOfUnixListener arrayOfUnknownNetworkError)
user=> (n/Interfaces)
[[{1 65536 lo  up|loopback} {2 1500 enp7s0 e0:d5:5e:2a:49:1b up|broadcast|multicast} {3 1500 enp6s0 e0:d5:5e:2a:49:19 up|broadcast|multicast} {4 1500 wlp5s0 3c:f0:11:3c:51:95 up|broadcast|multicast}] nil]
user=>
$
```

### Further Reading

See [GOSTD Usage](GOSTD.md) for more information.

## Developer Notes

See [`DEVELOPER.md`](DEVELOPER.md) for information on Joker internals, such as adding new namespaces to the Joker executable.

## License

```
Copyright (c) Roman Bataev. All rights reserved.
The use and distribution terms for this software are covered by the
Eclipse Public License 1.0 (http://opensource.org/licenses/eclipse-1.0.php)
which can be found in the LICENSE file.
```

Joker contains parts of Clojure source code (from `clojure.core` namespace). Clojure is licensed as follows:

```
Copyright (c) Rich Hickey. All rights reserved.
The use and distribution terms for this software are covered by the
Eclipse Public License 1.0 (http://opensource.org/licenses/eclipse-1.0.php)
which can be found in the file epl-v10.html at the root of this distribution.
By using this software in any fashion, you are agreeing to be bound by
the terms of this license.
You must not remove this notice, or any other, from this software.
```
