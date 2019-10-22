#!/usr/bin/env bash

# Undo "side-effects" of running tools/gostd/gostd.

rm -fr docs/go.std.* core/a_*_data.go

(cd tools/gostd && go build .) && ./tools/gostd/gostd --undo --joker .

(cd docs; ../joker generate-docs.joke --no-go)

# Delete regenerated file that is not in the repo (in this fork/branch).
rm -fr docs/index.html