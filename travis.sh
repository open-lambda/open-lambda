#!/bin/bash

unset GOPATH
unset GOROOT

GO_FILES=$(cd ol && find . -type f ! -path './vendor/*' -name '*.go')

# format test
if [[ $(cd ol && gofmt -d $GO_FILES) ]]; then
	cat <<EOF
Error: format check failed
Please format "ol" directory with the following command and commit again:

    gofmt -w -l \$(find $PWD/ol ! -path '*/vendor/*' -name '*.go')
EOF
	exit 1
fi

# build test
make bin/ol
