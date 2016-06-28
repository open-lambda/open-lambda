#!/usr/bin/env bash

set -e -x

echo run tests
git clone https://github.com/open-lambda/open-lambda
cd open-lambda
bash ./tools/quickstart/startup.sh
make test
