#!/usr/bin/env bash

build() {
    go clean -x

    go generate ./...

    # Don't vet things in tools/, they have their own vetting, plus "problematic" code for test purposes.

    go tool vet -all -shadow=true main.go

    go tool vet -all -shadow=true core std

    go build
}

set -e  # Exit on error.

if [ -e GO.link ] && which gostd2joker > /dev/null 2>&1; then
    go run tools/gostd/main.go --replace --joker .
fi

build

./joker -e '(print "\nLibraries available in this build:\n  ") *loaded-libs* (println)'

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
