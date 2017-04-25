#!/bin/bash

# Run the simple inter-lambda call benchmark.
# This assumes the clusters, named bench_resp and bench_call,
# have already been created with ./bin/admin new, and their
# registries have already been populated with the lambdas
# from mm_lambdas. 
# Clean up afterwards with ./clean_bench.sh.

set -x

# Assume nobody running, clean workers dir
./bin/admin workers -cluster=bench_resp -p=8081
./bin/admin workers -cluster=bench_call
# Give it a sec to catch up
sleep 3
#curl -X POST localhost:8080/runLambda/call_curl -d '{}'
#curl -X POST localhost:8080/runLambda/call_perf -d '{}'
#curl -X POST localhost:8080/runLambda/ipc_test -d '{}'
#curl -X POST localhost:8080/runLambda/fork_test -d '{}'
#curl -X POST localhost:8080/runLambda/pickle_test -d '{}'
#curl -X POST localhost:8080/runLambda/proto_test -d '{}'

# IPC bench
#curl -X POST localhost:8081/runLambda/ipc_resp -d '{}' & # setup resp
#curl -X POST localhost:8080/runLambda/ipc_call -d '{}' # execute call

# Serial test: pickle
#curl -X POST localhost:8081/runLambda/pkl_resp -d '{"num_keys":"1", "depth" : "1", "value_len" : "1"}' & # setup resp
#curl -X POST localhost:8080/runLambda/pkl_call -d '{"num_keys":"1", "depth" : "1", "value_len" : "1"}' # execute call

# Serial test: json (still uses pkl_resp)
curl -X POST localhost:8081/runLambda/pkl_resp -d '{"num_keys":"1", "depth" : "1", "value_len" : "1"}' & # setup resp
curl -X POST localhost:8080/runLambda/json_call -d '{"num_keys":"1", "depth" : "1", "value_len" : "1"}' # execute call
