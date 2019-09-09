#!/bin/bash

# Vet only the files in this directory; tests/* has Go's own std code,
# which (as of this writing) has many cases of shadowed
# variables. (That's why joker's run.sh file does not vet everything
# in its entire directory tree.)

ONERROR=":"
[ -r ONERROR.txt ] && ONERROR="$(cat ONERROR.txt)"

go vet -all *.go && go build && ./test.sh --on-error "$ONERROR"
