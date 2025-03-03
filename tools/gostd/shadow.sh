#!/usr/bin/env bash

# Check for shadowed variables.

if which shadow >/dev/null 2>/dev/null; then
    SHADOW="shadow"
elif $(go tool vet nonexistent.go 2>&1 | grep -q -v unsupported); then
    SHADOW="go vet -shadow=true"
fi

if [ -n "$SHADOW" ]; then
    $SHADOW *.go
else
    echo >&2 "Not performing shadowed-variables check; consider installing shadow tool via:"
    echo >&2 "  go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow"
    echo >&2 "and rerunning this script."
    exit 99
fi
