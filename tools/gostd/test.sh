#!/bin/bash

dircat() {
    find "$1" -type f | sort | while read f; do printf -- "-------- BEGIN %s:\n" "$f"; cat "$f"; printf -- "-------- END %s.\n" "$f"; done
}

[ ! -x gostd ] && echo >&2 "No executable to test." && exit 99

export GOENV="_tests/gold/$(go env GOARCH)-$(go env GOOS)"
OUTDIR="$GOENV/joker"

rm -fr "$OUTDIR" core-apis.dat
mkdir -p "$OUTDIR"/{core/data,std}

git show gostd:../../core/go_templates/g_goswitch.gotemplate > "$OUTDIR/core/g_goswitch.go"
git show gostd:../../core/go_templates/g_customlibs.joketemplate > "$OUTDIR/core/data/g_customlibs.joke"

RC=0

{ ./gostd --no-timestamp --verbose --joker ../.. --replace --output "$OUTDIR" 2>&1 | grep -v '^Default context:'; dircat "$OUTDIR"; } | sed 's:/usr/local/go1.18beta1:/usr/local/go:g' > "$GOENV/gosrc.gold"
git diff --quiet -u "$GOENV/gosrc.gold" || { echo >&2 "FAILED: gosrc test"; RC=1; $EXIT; }

exit $RC
