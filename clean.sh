#!/usr/bin/env bash

# Undo "side-effects" of running tools/gostd/gostd.

rm -fr docs/index.html docs/go.std.* core/a_*_data.go

(cd tools/gostd && go build .) && ./tools/gostd/gostd --undo --joker .

(cd docs; ../joker generate-docs.joke --no-go)
