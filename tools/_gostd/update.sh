#!/bin/bash
GOENV="$(go env GOARCH)-$(go env GOOS)"

git pull && go clean && go vet && go build && ./test.sh --on-error : && echo "No changes to $GOENV test results." && exit 0

git diff

read -p "Accept and update $GOENV test results? " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]
then
    git commit -a -m "Update $GOENV tests" && git push
fi
