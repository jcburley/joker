#!/usr/bin/env bash

build() {
  go clean
  rm -f core/a_*.go  # In case switching from a gen-code branch or similar (any existing files might break the build here)
  go generate ./...
  (cd core; go fmt a_*.go > /dev/null)
  go vet ./...
  go build
}

set -e  # Exit on error.

if [ ! -x "$JOKER" ]; then
    ./clean.sh >/dev/null 2>/dev/null
    rm -f core-apis.dat  # Refresh list of 'core' APIs via tools/gostd/walk.go/findApis()
    build
    JOKER=../joker
    ALREADY_BUILT=true
else
    ALREADY_BUILT=false
fi

[ ! -f NO-GOSTD.flag ] && (cd tools/gostd && go build .) && ./tools/gostd/gostd --replace --clojure .

# Check for changes in std, and run just-built Joker, only when building for host os/architecture.
SUM256="$(go run tools/sum256dir/main.go std)"
if [ ! -f NO-GEN.flag ]; then
    OUT="$(cd std; $JOKER generate-std.joke 2>&1 | grep -v 'WARNING:.*already refers' | grep '.')" || : # grep returns non-zero if no lines match
    if [ -n "$OUT" ]; then
        echo "$OUT"
        echo >&2 "Unable to generate fresh library files; exiting."
        exit 2
    fi
fi
(cd std; go fmt ./... > /dev/null)
NEW_SUM256="$(go run tools/sum256dir/main.go std)"

if [ "$SUM256" != "$NEW_SUM256" ]; then
    $ALREADY_BUILT && echo 'std has changed, rebuilding...'
    build
    (cd docs; ../joker generate-docs.joke)
fi

if [ "$1" == "-v" ]; then
  ./joker -e '(print "\nLibraries available in this build:\n  ") (loaded-libs) (println)'
fi

./joker "$@"
