#!/usr/bin/env bash

build() {
    go clean -x

    rm -f core/a_*_data.go  # Eases development across multiple branches introducing new core/data/*.joke files

    go generate ./...

    # Don't vet things in tools/, they have their own vetting, plus "problematic" code for test purposes.

    go vet -all main.go

    go vet -all ./core/... ./std/...

    [ -n "$SHADOW" ] && (go vet -all "$SHADOW" main.go; go vet -all "$SHADOW" ./core/... ./std/...) && echo "Shadowed-variables check complete."

    go build
}

if which shadow >/dev/null 2>/dev/null; then
    # Install via: go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
    SHADOW="-vettool=$(which shadow)"
fi

set -e  # Exit on error.

(cd tools/gostd && go build .) && ./tools/gostd/gostd --replace --joker .

build

./joker -e '(print "\nLibraries available in this build:\n  ") (loaded-libs) (println)'

SUM256="$(go run tools/sum256dir/main.go std)"

(cd std; ../joker generate-std.joke)

NEW_SUM256="$(go run tools/sum256dir/main.go std)"

if [ "$SUM256" != "$NEW_SUM256" ]; then
    cat <<EOF
Rebuilding Joker, as the libraries have changed; then regenerating docs.

EOF
    build

    (cd docs; ../joker generate-docs.joke)
fi

./joker "$@"
