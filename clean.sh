#!/usr/bin/env bash

# Undo "side-effects" of running tools/gostd/gostd.

set -e  # Exit on error.

rm -fr docs/go.std.* core/a_*_data.go
rm -f g_* core/g_* core/data/g_*
rm -fr std/go*

# Restore original versions of generated files so vanilla Joker can build.
cp core/go_templates/g_goswitch.gotemplate core/g_goswitch.go
cp core/go_templates/g_customlibs.joketemplate core/data/g_customlibs.joke

# Ok if failure here.
(cd docs; ../joker generate-docs.joke --no-go || ../joker-good generate-docs.joke --no-go || :)

# Delete regenerated file that is not in the repo (in this fork/branch).
rm -fr docs/index.html
