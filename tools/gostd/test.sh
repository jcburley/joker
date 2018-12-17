#!/bin/bash

export GOENV="tests/gold/$(go env GOARCH)-$(go env GOOS)"
mkdir -p "$GOENV"

EXIT="exit 99"
if [ "$1" = "--on-error" ]; then
    EXIT="$2"
fi

RC=0

./gostd2joker --no-timestamp -v --go tests/small 2>&1 | grep -v '^Default context:' > $GOENV/small.gold
git diff --quiet -u $GOENV/small.gold || { echo >&2 "FAILED: small test"; RC=1; $EXIT; }

rm -fr $GOENV/joker
cp -pr tests/joker.orig $GOENV/joker
./gostd2joker --no-timestamp -v --go tests/big --replace --joker $GOENV/joker 2>&1 | grep -v '^Default context:' > $GOENV/big.gold
git diff --quiet -u $GOENV/big.gold || { echo >&2 "FAILED: big test"; RC=1; $EXIT; }

if [ -z "$GOSRC" -a -e ./GO.link ]; then
    GOSRC=./GO.link
fi

if [ -n "$GOSRC" -a -d "$GOSRC" ]; then
    ./gostd2joker --no-timestamp -v --go "$GOSRC" 2>&1 | grep -v '^Default context:' > $GOENV/gosrc.gold
    git diff --quiet -u $GOENV/gosrc.gold || { echo >&2 "FAILED: \$GOSRC test"; RC=1; $EXIT; }
fi

exit $RC
