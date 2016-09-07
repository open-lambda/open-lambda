#!/usr/bin/env bash

set -e -x

echo run tests
apt-get install -y git
git clone https://github.com/open-lambda/open-lambda
cd open-lambda
bash ./quickstart/deps.sh
bash make
bash ./bin/worker quickstart/quickstart.json &
bash sleep 2
bash curl -X POST localhost:8080/runLambda/hello -d '{"name":"test"}'
