#! /bin/env python3

'''
SOCK-specific tests
'''

#pylint: disable=global-statement,too-many-statements,missing-function-docstring

import argparse
import os

from time import time

from multiprocessing import Pool

from helper import SockWorker, prepare_open_lambda, setup_config, TestConfContext
from helper.test import set_test_filter, start_tests, check_test_results, set_worker_type, test

from open_lambda import OpenLambda

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
    parser = argparse.ArgumentParser(description='Run SOCK-specific tests for OpenLambda')
    parser.add_argument('--reuse_config', action="store_true")
    parser.add_argument('--test_filter', type=str, default="")
    parser.add_argument('--ol_dir', type=str, default="test-dir")
    parser.add_argument('--registry', type=str, default="test-registry")

    args = parser.parse_args()

    set_test_filter([name for name in args.test_filter.split(",") if name != ''])
    set_worker_type(SockWorker)

    setup_config(args.ol_dir)
    prepare_open_lambda(args.ol_dir)

    start_tests()
    with TestConfContext(registry=os.path.abspath(args.registry), limits={"installer_mem_mb": 250}):
        run_tests()
    check_test_results()

if __name__ == '__main__':
    main()
