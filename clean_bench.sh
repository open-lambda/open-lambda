#!/bin/bash

# Script to clean up env after ./run_bench.sh.
# See run_bench.sh for setup explanation.

set -x

./bin/admin kill -cluster=bench_call
./bin/admin kill -cluster=bench_resp
rm -rf ./bench_call/workers/*
rm -rf ./bench_resp/workers/*
