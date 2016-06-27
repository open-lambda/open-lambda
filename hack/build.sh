#!/bin/bash

# This script builds the project, without the need to understand how
# to setup a GOPATH inspired from docker's script:
# https://github.com/docker/docker/blob/master/hack/make.sh

set -e

export WD=$(pwd)
export GOPATH=${WD}'/go'
export CODE_BASE='../worker'
export LAMBDA_PACKAGE=${GOPATH}'/src/github.com/open-lambda/open-lambda'
export WORKER=${LAMBDA_PACKAGE}'/worker/'
export CLIENT=${LAMBDA_PACKAGE}'/worker/prof/client'

# setup commit hooks
ln -s ${WD}/pre-commit ${WD}/../.git/hooks/ >/dev/null 2>/dev/null || true

# build
if [[ $1 == "test" ]]
then
    set -e -x
    cd ${WORKER}; go test ${@:2}
else
    set -e -x
    # just compile
    cd ${WORKER}; go get; go build
    cd ${CLIENT}; go get; go build
fi
