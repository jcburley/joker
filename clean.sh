#!/usr/bin/env bash

# Undo "side-effects" of running tools/gostd/gostd.

set -e  # Exit on error.

rm -fr docs/go.std.* core/a_*_data.go

(cd tools/gostd && go build .)
./tools/gostd/gostd --undo --clojure .

(cd docs; ../joker generate-docs.joke --no-go || ../joker-good generate-docs.joke --no-go || :)  # ok if failure here

# Delete regenerated file that is not in the repo (in this fork/branch).
rm -fr docs/index.html
