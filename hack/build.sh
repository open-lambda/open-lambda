#!/bin/bash

# This script builds the project, without the need to understand how to setup a GOPATH
# inspired from docker's script: https://github.com/docker/docker/blob/master/hack/make.sh

export CODE_BASE='../worker'

export LAMBDA_PACKAGE='src/github.com/tylerharter/open-lambda'
export GOPATH='.gopath'

export WORKER='worker/'
export CLIENT='worker/prof/client'

export WD=$(pwd)

mkdir -p ${GOPATH}/${LAMBDA_PACKAGE}
ln -sf ${WD}/${CODE_BASE} ${GOPATH}/${LAMBDA_PACKAGE}

# now that gopath exists, we re-export as absolute, required by go
export GOPATH=${WD}/${GOPATH}

cd ${GOPATH}/${LAMBDA_PACKAGE}/${WORKER} && go get && go build
cd ${GOPATH}/${LAMBDA_PACKAGE}/${CLIENT} && go get && go build

