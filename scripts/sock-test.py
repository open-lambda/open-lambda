#! /bin/env python3

'''
SOCK-specific tests
'''

#pylint: disable=global-statement,too-many-statements

import argparse
import traceback
import json
import os
import threading
import sys

from time import time
from collections import OrderedDict

from multiprocessing import Pool

from helper import SockWorker, prepare_open_lambda, setup_config, get_current_config, mounts
from helper import get_worker_output, get_ol_stats, TestConfContext, ol_oom_killer

from open_lambda import OpenLambda

results = OrderedDict({"runs": []})

# These will be set by argparse in main()
TEST_FILTER = []
WORKER_TYPE = None
OL_DIR = None

def test_in_filter(name):
    if len(TEST_FILTER) == 0:
        return True

    return name in TEST_FILTER

def test(func):
    def wrapper(*args, **kwargs):
        if len(args) > 0:
            raise Exception("positional args not supported for tests")

        name = func.__name__

        if not test_in_filter(name):
            print(f'Skipping test "{name}"')
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
        result["conf"] = get_current_config()
        result["seconds"] = None
        result["total_seconds"] = None
        result["stats"] = None
        result["ol-stats"] = None
        result["errors"] = []
        result["worker_tail"] = None

        total_t0 = time()
        mounts0 = mounts()
        worker = None

        try:
            worker = SockWorker()
            print("Worker started")

            # run test/benchmark
            test_t0 = time()
            return_val = func(**kwargs)
            test_t1 = time()
            result["seconds"] = test_t1 - test_t0

            result["pass"] = True
        except Exception as err:
            print(f"Failed to start worker: {err}")
            return_val = None
            result["pass"] = False
            result["errors"].append(traceback.format_exc().split("\n"))

        if worker:
            worker.stop()

        mounts1 = mounts()
        if len(mounts0) != len(mounts1):
            result["pass"] = False
            result["errors"].append([f"mounts are leaking ({len(mounts0)} before, {len(mounts1)} after), leaked: {mounts1 - mounts0}"])

        # get internal stats from OL
        result["ol-stats"] = get_ol_stats()

        total_t1 = time()
        result["total_seconds"] = total_t1-total_t0
        result["stats"] = return_val

        result["worker_tail"] = get_worker_output()
        if result["pass"]:
            # truncate because we probably won't use it for debugging
            result["worker_tail"] = result["worker_tail"][-10:]

        results["runs"].append(result)
        print(json.dumps(result, indent=2))
        return return_val

    return wrapper


def sock_churn_task(args):
    open_lambda = OpenLambda()

    echo_path, parent, start, seconds = args
    count = 0
    while time() < start + seconds:
        sandbox_id = open_lambda.create({"code": echo_path, "leaf": True, "parent": parent})
        open_lambda.destroy(sandbox_id)
        count += 1
    return count

@test
def sock_churn(baseline, procs, seconds, fork):
    # baseline: how many sandboxes are sitting idly throughout the experiment
    # procs: how many procs are concurrently creating and deleting other sandboxes

    echo_path = os.path.abspath("test-registry/echo")
    open_lambda = OpenLambda()

    if fork:
        parent = open_lambda.create({"code": "", "leaf": False})
    else:
        parent = ""

    for _ in range(baseline):
        sandbox_id = open_lambda.create({"code": echo_path, "leaf": True, "parent": parent})
        open_lambda.pause(sandbox_id)

    start = time()
    with Pool(procs) as pool:
        reqs = sum(pool.map(sock_churn_task, [(echo_path, parent, start, seconds)] * procs,
            chunksize=1))

    return {"sandboxes_per_sec": reqs/seconds}

def run_tests():
    print("Testing SOCK directly (without lambdas)")

    with TestConfContext(server_mode="sock", mem_pool_mb=500):
        sock_churn(baseline=0, procs=1, seconds=5, fork=False)
        sock_churn(baseline=0, procs=1, seconds=10, fork=True)
        sock_churn(baseline=0, procs=15, seconds=10, fork=True)
        sock_churn(baseline=32, procs=1, seconds=10, fork=True)
        sock_churn(baseline=32, procs=15, seconds=10, fork=True)

def main():
    global TEST_FILTER
    global OL_DIR

    parser = argparse.ArgumentParser(description='Run SOCK-specific tests for OpenLambda')
    parser.add_argument('--reuse_config', action="store_true")
    parser.add_argument('--test_filter', type=str, default="")
    parser.add_argument('--ol_dir', type=str, default="test-dir")

    args = parser.parse_args()

    TEST_FILTER = [name for name in args.test_filter.split(",") if name != '']
    OL_DIR = args.ol_dir

    setup_config(args.ol_dir, "test-registry")
    prepare_open_lambda()

    print(f'Test filter is "{TEST_FILTER}" and OL directory is "{args.ol_dir}"')

    start = time()

    # so our test script doesn't hang if we have a memory leak
    timer_thread = threading.Thread(target=ol_oom_killer, daemon=True)
    timer_thread.start()

    # run tests with various configs
    with TestConfContext(limits={"installer_mem_mb": 250}):
        run_tests()

    # save test results
    passed = len([t for t in results["runs"] if t["pass"]])
    failed = len([t for t in results["runs"] if not t["pass"]])
    results["passed"] = passed
    results["failed"] = failed
    results["seconds"] = time() - start
    print(f"PASSED: {passed}, FAILED: {failed}")

    with open("test.json", "w", encoding='utf-8') as resultsfile:
        json.dump(results, resultsfile, indent=2)

    sys.exit(failed)

if __name__ == '__main__':
    main()
