#!/bin/bash

export GOENV="_tests/gold/$(go env GOARCH)-$(go env GOOS)"
mkdir -p "$GOENV"

EXIT="exit 99"
if [ "$1" = "--on-error" ]; then
    EXIT="$2"
    shift 2
fi

RC=0

rm -fr $GOENV/joker core-apis.dat
mkdir -p $GOENV/joker/{core/data,std}
git show @:../../core/go_templates/g_goswitch.gotemplate > $GOENV/joker/core/g_goswitch.go
git show @:../../core/go_templates/g_customlibs.joketemplate > $GOENV/joker/core/data/g_customlibs.joke

if [ "$1" = "--reset" ]; then
    exit 0
fi

[ ! -x gostd ] && echo >&2 "No executable to test." && exit 99

./gostd --no-timestamp --output-code --verbose --joker ../.. --go _tests/small 2>&1 | grep -v '^Default context:' > $GOENV/small.gold
git diff --quiet -u $GOENV/small.gold || { echo >&2 "FAILED: small test"; RC=1; $EXIT; }

./gostd --no-timestamp --output-code --verbose --joker ../.. --go _tests/big --replace --clojure $GOENV/joker --import-from -- 2>&1 | grep -v '^Default context:' > $GOENV/big.gold
git diff --quiet -u $GOENV/big.gold || { echo >&2 "FAILED: big test"; RC=1; $EXIT; }

./gostd --no-timestamp --output-code --verbose --joker ../.. 2>&1 | grep -v '^Default context:' > $GOENV/gosrc.gold
git diff --quiet -u $GOENV/gosrc.gold || { echo >&2 "FAILED: gosrc test"; RC=1; $EXIT; }

exit $RC
