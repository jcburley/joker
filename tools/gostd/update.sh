#!/bin/bash
GOENV="$(go env GOARCH)-$(go env GOOS)"

git pull && go clean && ./build.sh && echo "No changes to $GOENV test results." && exit 0

git diff _tests

read -p "Accept and update $GOENV test results (only)? " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]
then
    git commit -m "Update $GOENV tests" _tests && git push
fi
