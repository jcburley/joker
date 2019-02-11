#!/bin/bash

# Vet only the files in this directory; tests/* has Go's own std code,
# which (as of this writing) has many cases of shadowed
# variables. (That's why joker's run.sh file does not vet everything
# in its entire directory tree.)

if which shadow >/dev/null 2>/dev/null; then
    # Install via: go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
    SHADOW="-vettool=$(which shadow)"
elif $(go tool vet nonexistent.go 2>&1 | grep -q -v unsupported); then
    SHADOW="-shadow=true"
fi

vet() {
    go vet -all

    if [ -n "$SHADOW" ]; then
        go vet -all "$SHADOW" && echo "Shadowed-variables check complete." || echo "Shadowed-variables check failed."
    else
        echo "Not performing shadowed-variables check; consider installing shadow tool via:"
        echo "  go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow"
        echo "and rebuilding."
    fi
}

vet && go build && ./test.sh --on-error :
