#!/usr/bin/env python3

''' Integration test for open lambda's native and WebAssembly runtimes '''

# pylint: disable=missing-function-docstring, consider-using-with

import argparse
import os

from time import time

from open_lambda import OpenLambda

from helper import DockerWorker, WasmWorker, SockWorker, TestConfContext
from helper import prepare_open_lambda, setup_config

from helper.test import set_test_filter, start_tests, check_test_results, set_worker_type, test

def get_mem_stat_mb(stat):
    with open('/proc/meminfo', 'r', encoding='utf-8') as file:
        for line in file:
            if line.startswith(stat+":"):
                parts = line.strip().split()
                assert parts[-1] == 'kB'
                return int(parts[1]) / 1024
    raise Exception('could not get stat')

@test
def ping():
    open_lambda = OpenLambda()

    pings = 1000
    t_start = time()
    for _ in range(pings):
        open_lambda.check_status()
    seconds = time() - t_start
    return {"pings_per_sec": pings/seconds}

@test
def noop():
    open_lambda = OpenLambda()
    open_lambda.run("noop", args=[], json=False)

@test
def hashing():
    open_lambda = OpenLambda()
    open_lambda.run("hashing", args={"num_hashes": 100, "input_len": 1024}, json=False)

def run_tests():
    ''' Runs all tests '''

    ping()
    noop()
    hashing()

def _main():
    parser = argparse.ArgumentParser(description='Run tests for OpenLambda')
    parser.add_argument('--test_filter', type=str, default="")
    parser.add_argument('--worker_type', type=str, default="sock")
    parser.add_argument('--ol_dir', type=str, default="test-dir")
    parser.add_argument('--registry', type=str, default="test-registry")

    args = parser.parse_args()

    set_test_filter([name for name in args.test_filter.split(",") if name != ''])
    wasm = False

    if args.worker_type == 'docker':
        set_worker_type(DockerWorker)
    elif args.worker_type == 'sock':
        set_worker_type(SockWorker)
    elif args.worker_type in ["webassembly", "wasm"]:
        set_worker_type(WasmWorker)
        wasm = True
    else:
        raise RuntimeError(f"Invalid worker type {args.worker_type}")

    if wasm:
        start_tests()
        run_tests()
    else:
        setup_config(args.ol_dir)
        prepare_open_lambda(args.ol_dir)

        start_tests()

        registry = os.path.abspath(args.registry)
        with TestConfContext(registry=registry):
            run_tests()

    check_test_results()

if __name__ == '__main__':
    _main()
