# gostd2joker

## Quick Start

To make this "magic" happen:

* Ensure you're running Go version 1.11.2 (see `go version`), as the copy of the subset of some supported Go packages, that comes with `gostd2joker`, comes from that version (which will matter only if you want to run tests, as described below)
* `go get github.com/jcburley/gostd2joker`, which should build and install `gostd2joker`
* Get a copy of the Go source tree, e.g. [https://github.com/golang/go](from Github), and check out the tag/branch corresponding to the version of Go you're running (see `go version`)
* Check out the `gostd2joker` branch of [https://github.com/jcburley/joker.git](my fork of Joker) and `cd` to it
* Create a symlink from that copy of the Go source tree you checked out (above) to `./GO.link` (in the top-level Joker source directory)
* `./run.sh`, specifying optional args such as `--version`, `-e '(println "i am here")'`, or even:

```
-e "(require '[joker.go.net :as n]) (print \"\\nNetwork interfaces:\\n  \") (n/Interfaces) (println)"
```

## Overview of Tool's Relationship to Joker and Go

Before building Joker, one can optionally run this tool against a Go source tree, which _must_ correspond to the version of Go used to build Joker itself, to populate `joker/std/go/` and modify related Joker source files. Further, the build parameters (`$GOARCH`, `$GOOS`, etc.) must match -- so `build-all.sh` would have to pass those to this tool (if it was to be used) for each of the targets.

At the moment, this is just a proof of concept, focusing initially on `net.LookupMX()`. E.g. run it standalone like this:

```
$ cd joker # Joker source directory
$ ln -s <go-source-directory> GO.link
$ gostd2joker 2>&1 | less
```

Then page through it. Code snippets intended for e.g. `joker/std/go/net.joke` are printed to `stdout`, making iteration (during development of this tool) much easier. Or, specify `--joker <joker-source-directory>` (typically `--joker .`) to get all the individual `*.joke` and `*.go` files in `<dir>/std/go/`, along with modifications to `<dir>/main.go`, `<dir>/core/data/core.joke`, and `<dir>/std/generate-std.joke`.

Anything not supported results in either a `panic` or, more often, the string `ABEND` along with some kind of explanation. The latter is used to auto-detect a non-convertible function, in which case the snippet(s) are still output, but commented-out, so it's easy to see what's missing and (perhaps) why.

Among things to do to "productize" this:

* NEEDED?: Change `generate-std.joke` to support this tool's output (this avoids the tool having to generate `*_native.go` snippets or files in many, if not all, cases)
* Might have to replace the current ad-hoc tracking of Go packages with something that respects `import` and the like
* SOMEWHAT DONE: Have tool generate more-helpful docstrings than just copying the ones with the Go functions -- e.g. the parameter types, maybe decorated with extra information?
* Document the code better
* Assess performance impact (especially startup time) on Joker, and mitigate as appropriate

## Sample Usage

Assuming Joker has been built as described above:

```
$ ./joker
Welcome to joker v0.10.1. Use EOF (Ctrl-D) or SIGINT (Ctrl-C) to exit.
user=> (require '[joker.go.net :as n])
nil
user=> (map #(key %) (ns-map 'joker.go.net))
(JoinHostPort LookupCNAME LookupHost LookupTXT ResolveIPAddr ResolveTCPAddr ResolveUDPAddr CIDRMask IPv4 InterfaceByIndex LookupAddr LookupPort ParseMAC ResolveUnixAddr SplitHostPort IPv4Mask InterfaceByName Interfaces LookupMX LookupNS LookupIP LookupSRV ParseCIDR ParseIP)
user=> (n/Interfaces)
[[{:Index 1, :MTU 65536, :Name "lo", :HardwareAddr [], :Flags 5} {:Index 2, :MTU 1500, :Name "eth0", :HardwareAddr [20 218 233 31 200 87], :Flags 19} {:Index 3, :MTU 1500, :Name "docker0", :HardwareAddr [2 66 188 97 92 58], :Flags 19}] nil]
user=>
$
```

## Run Tests

The `test.sh` script runs tests against a small, then larger, then
(optionally) full, copy of Go 1.11's `golang/go/src/` tree.

If you have a copy of the Go source tree available, define the `GOSRC` environment variable to point to its `src/` subdirectory, ideally via a the relative symlink `./GO.link`, to be compatible with existing test output. E.g.:

```
$ ln -s ~/golang/go ./GO.link  # If $GOSRC is undefined, ./GO.link will be tried
```

Then, invoke `test.sh` either with no options, or with `--on-error :` to run the `:` (`true`) command when it detects an error (the default being `exit 99`).

E.g.:

```
$ ./test.sh
$
```

The script currently runs tests in this order:

1. `tests/small`
2. `tests/big`
3. `./GOSRC` (or, if `$GOSRC` is non-null and points to a directory, `$GOSRC`)

After each test it runs, it uses `git diff` to compare the resulting `.gold` file with the checked-out version and, if there are any differences, it runs the command specified via `--on-error` (again, the default is `exit 99`, so the script will exit as soon as it sees a failing test).

## Update Tests on Other Machines

The Go standard library is customized per system architecture and OS, and `gostd2joker` picks up these differences via its use of Go's build-related packages. That's why `tests/gold/` has a subdirectory for each combination of `$GOARCH` and `$GOOS`. Updating another machine's copy of the `gostd2joker` repo is somewhat automated via `update.sh` -- e.g.:

```
$ ./update.sh 
remote: Enumerating objects: 8, done.
remote: Counting objects: 100% (8/8), done.
remote: Compressing objects: 100% (4/4), done.
remote: Total 6 (delta 4), reused 4 (delta 2), pack-reused 0
Unpacking objects: 100% (6/6), done.
From github.com:jcburley/gostd2joker
   5cfed10..3c00773  master     -> origin/master
Updating 5cfed10..3c00773
Fast-forward
 README.md | 63 +++++++++++++++++++++++++++++++++++----------------------------
 1 file changed, 35 insertions(+), 28 deletions(-)
No changes to amd64-darwin test results.
$
```

(Note the final line of output, indicating the value of `$GOARCH-$GOOS` in the `go` environment.)

If there are changes to the test results, they'll be displayed (via `git diff`), and the script will then prompt as to whether to accept and update them:

```
Accept and update amd64-darwin test results? y
[master 5cfed10] Update amd64-darwin tests
 3 files changed, 200 insertions(+), 200 deletions(-)
Counting objects: 8, done.
Delta compression using up to 8 threads.
Compressing objects: 100% (8/8), done.
Writing objects: 100% (8/8), 3.90 KiB | 266.00 KiB/s, done.
Total 8 (delta 4), reused 0 (delta 0)
remote: Resolving deltas: 100% (4/4), completed with 4 local objects.
To github.com:jcburley/gostd2joker
   339fbba..5cfed10  master -> master
$
```

(Don't forget to `git pull origin master` on your other development machines after updating test results, to avoid having to do the `git merge` dance when you make changes on them and try to `git push`.)
