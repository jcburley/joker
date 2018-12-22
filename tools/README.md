<img src="https://user-images.githubusercontent.com/882970/48048842-a0224080-e151-11e8-8855-642cf5ef3fdd.png" width="117px"/>

`_tests` is used instead of `tests` here because that directory includes snippets from Go's `std` subdirectory, which in turn appears to provide source code for packages imported by Joker. (At least, I'm assuming that's why I get build errors from `run.sh` when the subdirectory is named `gostd`.)
