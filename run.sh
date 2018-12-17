#!/usr/bin/env bash

set -e  # Exit on error.

if [ -e GO.link ]; then
    go run tools/gostd/main.go --replace --joker .
fi

go generate ./... && go tool vet ./ && go build && ./joker "$@"
