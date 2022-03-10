#!/usr/bin/env python3

''' Various integration tests for the open lambda framwork '''

# pylint: disable=global-statement, too-many-statements, fixme, broad-except, too-many-locals, missing-function-docstring

import argparse
import os
import sys
import json
import time
import requests
import traceback
import tempfile
import threading
import subprocess

from collections import OrderedDict
from subprocess import check_output
from multiprocessing import Pool

from helper import ContainerWorker, WasmWorker, prepare_open_lambda, setup_config, get_ol_stats, get_worker_output, get_current_config, TestConfContext

from api import OpenLambda

# These will be set by argparse in main()
TEST_FILTER = []
WORKER_TYPE = None

results = OrderedDict({"runs": []})

''' Issues a post request to the OL worker '''
def post(path, data=None):
    return requests.post('http://localhost:5000/'+path, json.dumps(data))

def raise_for_status(req):
    if req.status_code != 200:
        raise Exception(f"STATUS {req.status_code}: {req.text}")

def test_in_filter(name):
    if len(TEST_FILTER) == 0:
        return True

    return name in TEST_FILTER

def get_mem_stat_mb(stat):
    with open('/proc/meminfo', 'r', encoding='utf-8') as memfile:
        for line in memfile:
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

        total_t0 = time.time()
        mounts0 = mounts()
        worker = None

        try:
            worker = WORKER_TYPE()
            print("Worker started")

            # run test/benchmark
            test_t0 = time.time()
            return_val = func(**kwargs)
            test_t1 = time.time()
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

        total_t1 = time.time()
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

def mounts():
    output = check_output(["mount"])
    output = str(output, "utf-8")
    output = output.split("\n")
    return set(output)

def run(cmd):
    print("RUN", " ".join(cmd))
    try:
        out = check_output(cmd, stderr=subprocess.STDOUT)
        fail = False
    except subprocess.CalledProcessError as err:
        out = err.output
        fail = True

    out = str(out, 'utf-8')
    if len(out) > 500:
        out = out[:500] + "..."

    if fail:
        raise Exception(f'command ({" ".join(cmd)}) failed: {out}')

@test
def install_tests():
    # we want to make sure we see the expected number of pip installs,
    # so we don't want installs lying around from before
    return_code = os.system('rm -rf test-dir/lambda/packages/*')
    assert return_code == 0

    open_lambda = OpenLambda()

    # try something that doesn't install anything
    msg = 'hello world'
    jdata = open_lambda.run("echo", msg)
    if jdata != msg:
        raise Exception(f"found {jdata} but expected {msg}")

    jdata = open_lambda.get_statistics()
    installs = jdata.get('pull-package.cnt', 0)
    assert installs == 0

    for pos in range(3):
        name = "install"
        if pos != 0:
            name += str(pos+1)
        result = open_lambda.run(name, {})
        assert result == "imported"

        result = open_lambda.get_statistics()
        installs = result['pull-package.cnt']
        if pos < 2:
            # with deps, requests should give us these:
            # certifi, chardet, idna, requests, urllib3
            assert installs == 5
        else:
            assert installs == 6

def check_status_code(req):
    if req.status_code != 200:
        raise Exception(f"STATUS {req.status_code}: {req.text}")

@test
def hello_rust():
    open_lambda = OpenLambda()
    open_lambda.run("hello", [])

@test
def internal_call():
    open_lambda = OpenLambda()
    open_lambda.run("run/internal_call", {"count": 5})

@test
def numpy_test():
    open_lambda = OpenLambda()

    # try adding the nums in a few different matrixes.  Also make sure
    # we can have two different numpy versions co-existing.
    result = open_lambda.run("numpy19", [1, 2])
    assert result['result'] == 3
    assert result['version'].startswith('1.19')

    result = open_lambda.run("numpy20", [[1, 2], [3, 4]])
    assert result['result'] == 10
    assert result['version'].startswith('1.20')

    result = open_lambda.run("numpy19", [[[1, 2], [3, 4]], [[1, 2], [3, 4]]])
    assert result['result'] == 20
    assert result['version'].startswith('1.19')

    result = open_lambda.run("pandas", [[0, 1, 2], [3, 4, 5]])
    assert result['result'] == 15
    assert float(".".join(result['version'].split('.')[:2])) >= 1.19

    result = open_lambda.run("pandas18", [[1, 2, 3],[1, 2, 3]])
    assert result['result'] == 12
    assert result['version'].startswith('1.18')

def stress_one_lambda_task(args):
    open_lambda = OpenLambda()

    start, seconds = args
    pos = 0
    while time.time() < start + seconds:
        result = open_lambda.run("echo", pos, json=False)
        assert result.text == str(pos)
        pos += 1
    return pos

@test
def stress_one_lambda(procs, seconds):
    start = time.time()

    with Pool(procs) as pool:
        reqs = sum(pool.map(stress_one_lambda_task, [(start, seconds)] * procs, chunksize=1))

    return {"reqs_per_sec": reqs/seconds}

@test
def call_each_once_exec(lambda_count, alloc_mb):
    open_lambda = OpenLambda()

    # TODO: do in parallel
    start = time.time()
    for pos in range(lambda_count):
        result = open_lambda.run(f"L{pos}", {"alloc_mb": alloc_mb}, json=False)
        assert result == str(pos)
    seconds = time.time() - start

    return {"reqs_per_sec": lambda_count/seconds}

def call_each_once(lambda_count, alloc_mb=0):
    with tempfile.TemporaryDirectory() as reg_dir:
        # create dummy lambdas
        for pos in range(lambda_count):
            with open(os.path.join(reg_dir, f"L{pos}.py"), "w", encoding='utf-8') as code:
                code.write( "def f(event):\n")
                code.write( "    global s\n")
                code.write(f"    s = '*' * {alloc_mb} * 1024**2\n")
                code.write(f"    return {pos}\n")

        with TestConfContext(registry=reg_dir):
            call_each_once_exec(lambda_count=lambda_count, alloc_mb=alloc_mb)

@test
def fork_bomb():
    open_lambda = OpenLambda()

    limit = get_current_config()["limits"]["procs"]
    result = open_lambda.run("fbomb", {"times": limit*2}, json=False)

    # the function returns the number of children that we were able to fork
    assert 1 <= int(result) <= limit

@test
def max_mem_alloc():
    open_lambda = OpenLambda()

    limit = get_current_config()["limits"]["mem_mb"]
    result = open_lambda.run("max_mem_alloc", None)

    # the function returns the MB that was able to be allocated
    assert limit-16 <= int(result) <= limit

@test
def ping_test():
    open_lambda = OpenLambda()

    pings = 1000
    start = time.time()
    for _ in range(pings):
        open_lambda.check_status()

    seconds = time.time() - start
    return {"pings_per_sec": pings/seconds}

def sock_churn_task(args):
    echo_path, parent, start, seconds = args
    count = 0
    while time.time() < start + seconds:
        args = {"code": echo_path, "leaf": True, "parent": parent}
        req = post("create", args)
        raise_for_status(req)
        sandbox_id = req.text.strip()
        req = post("destroy/"+sandbox_id, {})
        raise_for_status(req)
        count += 1
    return count


@test
def sock_churn(baseline, procs, seconds, fork):
    # baseline: how many sandboxes are sitting idly throughout the experiment
    # procs: how many procs are concurrently creating and deleting other sandboxes

    echo_path = os.path.abspath("test-registry/echo")

    if fork:
        req = post("create", {"code": "", "leaf": False})
        raise_for_status(req)
        parent = req.text.strip()
    else:
        parent = ""

    for _ in range(baseline):
        req = post("create", {"code": echo_path, "leaf": True, "parent": parent})
        raise_for_status(req)
        sandbox_id = req.text.strip()
        req = post("pause/"+sandbox_id)
        raise_for_status(req)

    start = time.time()
    with Pool(procs) as pool:
        reqs = sum(pool.map(sock_churn_task, [(echo_path, parent, start, seconds)] * procs,
            chunksize=1))

    return {"sandboxes_per_sec": reqs/seconds}

@test
def rust_hashing():
    req = post("run/hashing", {"num_hashes": 100, "input_len": 1024})
    check_status_code(req)

@test
def update_code():
    curr_conf = get_current_config()
    reg_dir = curr_conf['registry']
    cache_seconds = curr_conf['registry_cache_ms'] / 1000

    for pos in range(3):
        # update function code
        with open(os.path.join(reg_dir, "version.py"), "w", encoding='utf-8') as code:
            code.write( "def f(event):\n")
            code.write(f"    return {pos}\n")

        # how long does it take for us to start seeing the latest code?
        start = time.time()
        while True:
            req = post("run/version", None)
            raise_for_status(req)
            num = int(req.text)
            assert num >= pos-1
            end = time.time()

            # make sure the time to grab new code is about the time
            # specified for the registry cache (within ~1 second)
            assert end - start <= cache_seconds + 1
            if num == pos:
                if pos > 0:
                    assert end - start >= cache_seconds - 1
                break

@test
def recursive_kill(depth):
    open_lambda = OpenLambda()

    parent = ""
    for _ in range(depth):
        open_lambda.create({"code": "", "leaf": False, "parent": parent})
        if parent:
            # don't need this parent any more, so pause it to get
            # memory back (so we can run this test with low memory)
            open_lambda.pause(parent)
        parent = req.text.strip()

    req = post("destroy/1", None)
    raise_for_status(req)
    req = post("stats", None)
    raise_for_status(req)
    destroys = req.json()['Destroy():ms.cnt']
    assert destroys == depth

@test
def increment():
    req = post("run/increment", {})
    raise_for_status(req)

def run_tests():
    ping_test()

    # test very basic rust program
    # hello_rust()

    # run some more computation in rust
    # rust_hashing()

    # internal_call()

     # increment()

    # do smoke tests under various configs
    with TestConfContext(features={"import_cache": False}):
        install_tests()
    with TestConfContext(mem_pool_mb=500):
        install_tests()
    with TestConfContext(sandbox="docker", features={"import_cache": False}):
        install_tests()

    # test resource limits
    fork_bomb()
    max_mem_alloc()

    # numpy pip install needs a larger mem cap
    with TestConfContext(mem_pool_mb=500):
        numpy_test()

#TODO # make sure code updates get pulled within the cache time
#    with tempfile.TemporaryDirectory() as reg_dir:
#        with TestConfContext(registry=reg_dir, registry_cache_ms=3000):
#            update_code()
#
#    # test heavy load
#    with TestConfContext(registry=test_reg):
#        stress_one_lambda(procs=1, seconds=15)
#        stress_one_lambda(procs=2, seconds=15)
#        stress_one_lambda(procs=8, seconds=15)
#
#    with TestConfContext(features={"reuse_cgroups": True}):
#        call_each_once(lambda_count=100, alloc_mb=1)
#        call_each_once(lambda_count=1000, alloc_mb=10)


# TODO move sock-specific tests somewhere lse
#    if "sock" in sandboxes:
#        print("Testing SOCK directly (without lambdas)")
#
#        with TestConfContext(server_mode="sock", mem_pool_mb=500):
#            sock_churn(baseline=0, procs=1, seconds=5, fork=False)
#            sock_churn(baseline=0, procs=1, seconds=10, fork=True)
#            sock_churn(baseline=0, procs=15, seconds=10, fork=True)
#            sock_churn(baseline=32, procs=1, seconds=10, fork=True)
#            sock_churn(baseline=32, procs=15, seconds=10, fork=True)
#

def main():
    global TEST_FILTER
    global WORKER_TYPE

    parser = argparse.ArgumentParser(description='Run tests for OpenLambda')
    parser.add_argument('--reuse_config', action="store_true")
    parser.add_argument('--worker_type', type=str, default="container")
    parser.add_argument('--test_filter', type=str, default="")
    parser.add_argument('--ol_dir', type=str, default="test-dir")

    args = parser.parse_args()

    TEST_FILTER = [name for name in args.test_filter.split(",") if name != '']
    setup_config(args.ol_dir, "test-registry")
    prepare_open_lambda()

    print(f'Test filter is "{TEST_FILTER}" and OL directory is "{args.ol_dir}"')

    if args.worker_type == 'container':
        WORKER_TYPE = ContainerWorker
    elif args.worker_type == 'wasm':
        WORKER_TYPE = WasmWorker
    else:
        raise RuntimeError(f"Invalid worker type {args.worker_type}")

    start = time.time()

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
    results["seconds"] = time.time() - start
    print(f"PASSED: {passed}, FAILED: {failed}")

    with open("test.json", "w", encoding='utf-8') as resultsfile:
        json.dump(results, resultsfile, indent=2)

    sys.exit(failed)

if __name__ == '__main__':
    main()
