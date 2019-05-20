#!/bin/bash

GO_FILES=$(cd worker && find . -type f ! -path './vendor/*' -name '*.go')

# format test
if [[ $(cd worker && gofmt -d $GO_FILES) ]]; then
	cat <<EOF
Error: format check failed
Please format "worker" directory with the following command and commit again:

    gofmt -w -l \$(find $PWD/worker ! -path '*/vendor/*' -name '*.go')
EOF
	exit 1
fi

# build test
make
