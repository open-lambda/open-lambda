#!/bin/bash

# This script builds the project, without the need to understand how
# to setup a GOPATH inspired from docker's script:
# https://github.com/docker/docker/blob/master/hack/make.sh

set -e -x

export WD=$(pwd)
export GOPATH=${WD}'/.gopath'
export CODE_BASE='../worker'
export LAMBDA_PACKAGE=${GOPATH}'/src/github.com/tylerharter/open-lambda'
export WORKER=${LAMBDA_PACKAGE}'/worker/'
export CLIENT=${LAMBDA_PACKAGE}'/worker/prof/client'

# init gopath
mkdir -p ${LAMBDA_PACKAGE}
ln -sf ${WD}/${CODE_BASE} ${LAMBDA_PACKAGE}

# build
cd ${WORKER}; go get; go build
cd ${CLIENT}; go get; go build

# setup commit hooks
ln -s ${WD}/pre-commit ${WD}/../.git/hooks/ >/dev/null 2>/dev/null || true
