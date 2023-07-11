#!/usr/bin/env python3

''' Various integration tests for the open lambda framwork '''

# pylint: disable=global-statement, too-many-statements, fixme, broad-except, too-many-locals, missing-function-docstring

import argparse
import os

from helper import DockerWorker, SockWorker, prepare_open_lambda, setup_config
from helper import get_current_config, TestConfContext, assert_eq

from helper.test import set_test_filter, start_tests, check_test_results, set_worker_type, test

from open_lambda import OpenLambda

# These will be set by argparse in main()
OL_DIR = None

@test
def syslog_test():
    open_lambda = OpenLambda()
    result = open_lambda.run("syslog", []) # this should fail

    enable_seccomp = get_current_config()["features"]["enable_seccomp"]

    assert_eq(result, -1 if enable_seccomp else 0)

def seccomp_tests():
    # syslog is allowed in docker, but not in ol seccomp filtering
    with TestConfContext(features={"enable_seccomp": True}):
        syslog_test() # syslog should fail
    with TestConfContext(features={"enable_seccomp": False}):
        syslog_test() # syslog should run


def main():
    global OL_DIR

    parser = argparse.ArgumentParser(description='Run tests for OpenLambda')
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
        seccomp_tests()

    check_test_results()

if __name__ == '__main__':
    main()
