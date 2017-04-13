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
curl -X POST localhost:8080/runLambda/call_perf -d '{}'
