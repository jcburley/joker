#!/bin/bash

dircat() {
    find "$1" -type f | sort | while read f; do printf -- "-------- BEGIN %s:\n" "$f"; cat "$f"; printf -- "-------- END %s.\n" "$f"; done
}

[ ! -x gostd ] && echo >&2 "No executable to test." && exit 99

# GO_MINOR_VER=
if ! [[ "$(go env GOVERSION)" =~ go([0-9]+\.[0-9]+) ]]; then
    echo >&2 "Bad version: $(go env GOVERSION)"
    exit 99
fi
GO_MINOR_VERSION="${BASH_REMATCH[1]}"

export GOENV="_tests/gold/go${GO_MINOR_VERSION}/$(go env GOARCH)-$(go env GOOS)"
OUTDIR="$GOENV/joker"

rm -fr "$OUTDIR" core-apis.dat
mkdir -p "$OUTDIR"/{core/data,std}

git show gostd:../../core/go_templates/g_goswitch.gotemplate > "$OUTDIR/core/g_goswitch.go"
git show gostd:../../core/go_templates/g_customlibs.joketemplate > "$OUTDIR/core/data/g_customlibs.joke"

RC=0

{ ./gostd --no-timestamp --verbose --joker ../.. --replace --output "$OUTDIR" 2>&1 | grep -v '^Default context:'; dircat "$OUTDIR"; } > "$GOENV/gosrc.gold"
git diff --quiet -u "$GOENV/gosrc.gold" || { echo >&2 "FAILED: gosrc test"; RC=1; $EXIT; }

exit $RC
