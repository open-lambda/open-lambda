#!/usr/bin/env python3

'''Many popular PyPI packages (e.g., pandas, scikit-learn, etc) have
extensive tests.  We want those packages to work on OL, so we run
these inside the SOCK containers.  Some of these suites take quite
long (e.g., 20+ minutes), so we set large timeouts.  Very high memory
limits (>1GB) are also necessary.
'''

import argparse
import os

from helper import DockerWorker, SockWorker, prepare_open_lambda, setup_config
from helper import TestConfContext, assert_eq

from helper.test import set_test_filter, start_tests, check_test_results, set_worker_type, test

from open_lambda import OpenLambda

# These will be set by argparse in main()
OL_DIR = None

@test
def pandas_test():
    open_lambda = OpenLambda()
    result = open_lambda.run("pandas-tests", None)
    assert_eq(result, True)

def run_tests():
    with TestConfContext(mem_pool_mb=6000, limits={"mem_mb": 2000, "max_runtime_default": 3600}):
        pandas_test()

def main():
    global OL_DIR

    parser = argparse.ArgumentParser(description='Run tests for OpenLambda')
    parser.add_argument('--reuse_config', action="store_true")
    parser.add_argument('--worker_type', type=str, default="sock")
    parser.add_argument('--test_filter', type=str, default="")
    parser.add_argument('--registry', type=str, default="test-registry")
    parser.add_argument('--ol_dir', type=str, default="test-dir")

    args = parser.parse_args()

    set_test_filter([name for name in args.test_filter.split(",") if name != ''])
    OL_DIR = args.ol_dir

    setup_config(args.ol_dir)
    prepare_open_lambda(args.ol_dir)

    trace_config = {
        "cgroups": True,
        "memory": True,
        "evictor": True,
        "package": True,
    }
    with TestConfContext(registry=os.path.abspath(args.registry), trace=trace_config):
        if args.worker_type == 'docker':
            set_worker_type(DockerWorker)
        elif args.worker_type == 'sock':
            set_worker_type(SockWorker)
        else:
            raise RuntimeError(f"Invalid worker type {args.worker_type}")

        start_tests()
        run_tests()

    check_test_results()

if __name__ == '__main__':
    main()
