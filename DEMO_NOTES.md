# OpenLambda Demo (Codespaces)

## Build
make ol imgs/ol-min

## Init + Run worker
sudo ./ol worker init -p myworker -i ol-min
sudo ./ol worker up -p myworker

## Health check
curl -s http://localhost:5000/status

## Install + Run hello lambda
sudo ./ol admin install -n hello -p myworker examples/hello
curl -s -X POST http://localhost:5000/run/hello -d '{}'
