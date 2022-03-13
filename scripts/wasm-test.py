#!/usr/bin/env python3

''' Integration test for open lambda's WebAssembly runtime '''

# pylint: disable=global-statement, missing-function-docstring, broad-except, invalid-name, consider-using-with

import argparse
import os
import sys
import json
import time
import traceback
import threading

from time import sleep
from collections import OrderedDict
from subprocess import Popen
from contextlib import contextmanager

from api import OpenLambda
from helper import Datastore
import lambdastore

# These will be set by argparse in main()
TEST_FILTER = []

results = OrderedDict({"runs": []})

def test_in_filter(name):
    if len(TEST_FILTER) == 0:
        return True

    return name in TEST_FILTER

def get_mem_stat_mb(stat):
    with open('/proc/meminfo', 'r', encoding='utf-8') as file:
        for line in file:
            if line.startswith(stat+":"):
                parts = line.strip().split()
                assert parts[-1] == 'kB'
                return int(parts[1]) / 1024
    raise Exception('could not get stat')

def ol_oom_killer():
    while True:
        if get_mem_stat_mb('MemAvailable') < 128:
            print("out of memory, trying to kill OL")
            os.system('pkill ol')
        time.sleep(1)

def test(func):
    def wrapper(*args, **kwargs):
        if len(args) > 0:
            raise Exception("positional args not supported for tests")

        name = func.__name__

        if not test_in_filter(name):
            print(f'"Skipping test "{name}"')
            return None

        print('='*40)
        if len(kwargs):
            print(name, kwargs)
        else:
            print(name)
        print('='*40)
        result = OrderedDict()
        result["test"] = name
        result["params"] = kwargs
        result["pass"] = None
        result["conf"] = None
        result["seconds"] = None
        result["total_seconds"] = None
        result["stats"] = None
        result["ol-stats"] = None
        result["errors"] = []
        result["worker_tail"] = None

        total_t0 = time.time()
        datastore = Datastore()
        worker = None

        try:
            # setup worker
            worker = Popen(['./ol-wasm'])
            sleep(0.1)

            # wait for worker to be ready
            while True:
                try:
                    open_lambda = OpenLambda()
                    open_lambda.check_status()
                    break
                except:
                    # wait some more...
                    sleep(0.1)

            # run test/benchmark
            test_t0 = time.time()
            ret_val = func(**kwargs)
            test_t1 = time.time()
            result["seconds"] = test_t1 - test_t0

            result["pass"] = True
        except Exception:
            ret_val = None
            result["pass"] = False
            result["errors"].append(traceback.format_exc().split("\n"))

        # cleanup worker
        try:
            if worker:
                worker.kill()
        except Exception:
            result["pass"] = False
            result["errors"].append(traceback.format_exc().split("\n"))

        total_t1 = time.time()
        result["stats"] = ret_val
        result["total_seconds"] = total_t1-total_t0
        results["runs"].append(result)

        datastore.stop()

        print(json.dumps(result, indent=2))
        return ret_val

    return wrapper

# Loads a config and overwrites certain fields with what is set in **keywords
@contextmanager

@test
def wasm_numpy_test():
    open_lambda = OpenLambda()
    lstore = lambdastore.create_client('localhost')

    obj = lstore.create_object('test')
    oid = obj.get_identifier().to_hex_string()

    jdata = open_lambda.run_on(oid, 'numpy', [1,2])
    assert jdata['result'] == 3

    jdata = open_lambda.run_on(oid, "numpy", [[1, 2], [3, 4]])
    assert jdata['result'] == 10

    jdata = open_lambda.run_on(oid, "numpy", [[[1, 2], [3, 4]], [[1, 2], [3, 4]]])
    assert jdata['result'] == 20

@test
def ping_test():
    open_lambda = OpenLambda()

    pings = 1000
    t_start = time.time()
    for _ in range(pings):
        open_lambda.check_status()
    seconds = time.time() - t_start
    return {"pings_per_sec": pings/seconds}

def run_tests():
    ''' Runs all tests '''

    print("Testing WASM")

    ping_test()
    wasm_numpy_test()

def _main():
    global TEST_FILTER

    parser = argparse.ArgumentParser(description='Run tests for OpenLambda')
    parser.add_argument('--test_filter', type=str, default="")

    args = parser.parse_args()

    TEST_FILTER = [name for name in args.test_filter.split(",") if name != '']

    print(f'Test filter is "{TEST_FILTER}"')

    t_start = time.time()

    # so our test script doesn't hang if we have a memory leak
    timer_thread = threading.Thread(target=ol_oom_killer, daemon=True)
    timer_thread.start()

    # run tests with various configs
    run_tests()

    # save test results
    passed = len([t for t in results["runs"] if t["pass"]])
    failed = len([t for t in results["runs"] if not t["pass"]])
    results["passed"] = passed
    results["failed"] = failed
    results["seconds"] = time.time() - t_start
    print(f"PASSED: {passed}, FAILED: {failed}")

    with open("test.json", "w", encoding='utf-8') as file:
        json.dump(results, file, indent=2)

    sys.exit(failed)

if __name__ == '__main__':
    _main()
