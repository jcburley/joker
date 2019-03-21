#!/usr/bin/env bash

# Undo "side-effects" of running tools/gostd/gostd.

rm -fr docs/go.std.*

(cd tools/gostd && go build .) && ./tools/gostd/gostd --undo --joker .

(cd docs; ../joker generate-docs.joke --no-go)
