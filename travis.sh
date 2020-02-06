#!/bin/bash

unset GOPATH
unset GOROOT

GO_FILES=$(cd src && find . -type f ! -path './vendor/*' -name '*.go')

# format test
if [[ $(cd src && gofmt -d $GO_FILES) ]]; then
	cat <<EOF
Error: format check failed
Please format "src" directory with the following command and commit again:

    gofmt -w -l \$(find $PWD/src ! -path '*/vendor/*' -name '*.go')
EOF
	exit 1
fi

# build test
make ./ol
