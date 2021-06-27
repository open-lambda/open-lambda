#!/usr/bin/env python3

# pylint: disable=global-statement, too-many-statements, fixme

import argparse
import os
import sys
import json
import time
import requests
import copy
import traceback
import tempfile
import threading
import subprocess

from collections import OrderedDict
from subprocess import check_output
from multiprocessing import Pool
from contextlib import contextmanager

# These will be set by argparse in main()
OLDIR = ''
TEST_FILTER = []

results = OrderedDict({"runs": []})
CURR_CONF = None

''' Issues a post request to the OL worker '''
def post(path, data=None):
    return requests.post('http://localhost:5000/'+path, json.dumps(data))

def raise_for_status(req):
    if req.status_code != 200:
        raise Exception("STATUS %d: %s" % (req.status_code, req.text))

def test_in_filter(name):
    if len(TEST_FILTER) == 0:
        return True

    return name in TEST_FILTER

def get_mem_stat_mb(stat):
    with open('/proc/meminfo') as memfile:
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
            print("Skipping test '%s'" % name)
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
        result["conf"] = CURR_CONF
        result["seconds"] = None
        result["total_seconds"] = None
        result["stats"] = None
        result["ol-stats"] = None
        result["errors"] = []
        result["worker_tail"] = None

        total_t0 = time.time()
        mounts0 = mounts()
        try:
            # setup worker
            run(['./ol', 'worker', '-p='+OLDIR, '--detach'])

            # run test/benchmark
            test_t0 = time.time()
            return_val = func(**kwargs)
            test_t1 = time.time()
            result["seconds"] = test_t1 - test_t0

            result["pass"] = True
        except:
            return_val = None
            result["pass"] = False
            result["errors"].append(traceback.format_exc().split("\n"))

        # cleanup worker
        try:
            run(['./ol', 'kill', '-p='+OLDIR])
        except:
            result["pass"] = False
            result["errors"].append(traceback.format_exc().split("\n"))
        mounts1 = mounts()
        if len(mounts0) != len(mounts1):
            result["pass"] = False
            result["errors"].append(["mounts are leaking (%d before, %d after), leaked: %s"
                                     % (len(mounts0), len(mounts1), str(mounts1 - mounts0))])

        # get internal stats from OL
        if os.path.exists(OLDIR+"/worker/stats.json"):
            with open(OLDIR+"/worker/stats.json") as statsfile:
                olstats = json.load(statsfile)
                result["ol-stats"] = OrderedDict(sorted(list(olstats.items())))

        total_t1 = time.time()
        result["total_seconds"] = total_t1-total_t0
        result["stats"] = return_val

        with open(os.path.join(OLDIR, "worker.out")) as workerfile:
            result["worker_tail"] = workerfile.read().split("\n")
            if result["pass"]:
                # truncate because we probably won't use it for debugging
                result["worker_tail"] = result["worker_tail"][-10:]

        results["runs"].append(result)
        print(json.dumps(result, indent=2))
        return return_val

    return wrapper


def put_conf(conf):
    global CURR_CONF
    with open(os.path.join(OLDIR, "config.json"), "w") as configfile:
        json.dump(conf, configfile, indent=2)
    CURR_CONF = conf


def mounts():
    output = check_output(["mount"])
    output = str(output, "utf-8")
    output = output.split("\n")
    return set(output)

# Loads a config and overwrites certain fields with what is set in **keywords
@contextmanager
def test_conf(**keywords):
    with open(os.path.join(OLDIR, "config.json")) as conffile:
        orig = json.load(conffile)
    new = copy.deepcopy(orig)
    for key in keywords:
        if not key in new:
            raise Exception("unknown config param: %s" % key)
        if isinstance(keywords[key], dict):
            for key2 in keywords[key]:
                new[key][key2] = keywords[key][key2]
        else:
            new[key] = keywords[key]

    # setup
    print("PUSH conf:", keywords)
    put_conf(new)

    yield new

    # cleanup
    print("POP conf:", keywords)
    put_conf(orig)


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
        raise Exception("command (%s) failed: %s"  % (" ".join(cmd), out))


@test
def install_tests():
    # we want to make sure we see the expected number of pip installs,
    # so we don't want installs lying around from before
    return_code = os.system('rm -rf test-dir/lambda/packages/*')
    assert return_code == 0

    # try something that doesn't install anything
    msg = 'hello world'
    req = post("run/echo", msg)
    raise_for_status(req)
    if req.json() != msg:
        raise Exception("found %s but expected %s" % (req.json(), msg))
    req = post("stats", None)
    raise_for_status(req)
    installs = req.json().get('pull-package.cnt', 0)
    assert installs == 0

    for pos in range(3):
        name = "install"
        if pos != 0:
            name += str(pos+1)
        req = post("run/"+name, {})
        raise_for_status(req)
        assert req.json() == "imported"

        req = post("stats", None)
        raise_for_status(req)
        installs = req.json()['pull-package.cnt']
        if pos < 2:
            # with deps, requests should give us these:
            # certifi, chardet, idna, requests, urllib3
            assert installs == 5
        else:
            assert installs == 6

@test
def hello_rust():
    req = post("run/rust-hello", [])
    if req.status_code != 200:
        raise Exception("STATUS %d: %s" % (req.status_code, req.text))

@test
def internal_call():
    req = post("run/rust-internal_call", {"count": 5})
    if req.status_code != 200:
        raise Exception("STATUS %d: %s" % (req.status_code, req.text))

@test
def numpy_test():
    # try adding the nums in a few different matrixes.  Also make sure
    # we can have two different numpy versions co-existing.
    req = post("run/numpy19", [1, 2])
    if req.status_code != 200:
        raise Exception("STATUS %d: %s" % (req.status_code, req.text))
    j = req.json()
    assert j['result'] == 3
    assert j['version'].startswith('1.19')

    req = post("run/numpy20", [[1, 2], [3, 4]])
    if req.status_code != 200:
        raise Exception("STATUS %d: %s" % (req.status_code, req.text))
    j = req.json()
    assert j['result'] == 10
    assert j['version'].startswith('1.20')

    req = post("run/numpy19", [[[1, 2], [3, 4]], [[1, 2], [3, 4]]])
    if req.status_code != 200:
        raise Exception("STATUS %d: %s" % (req.status_code, req.text))
    j = req.json()
    assert j['result'] == 20
    assert j['version'].startswith('1.19')

    # use rust binary
    req = post("run/rust-numpy", [1, 2])
    if req.status_code != 200:
        raise Exception("STATUS %d: %s" % (req.status_code, req.text))
    try:
        j = req.json()
    except:
        raise Exception("Failed to decode json for request %s" % req.text)

    assert j['result'] == 3

    req = post("run/pandas", [[0, 1, 2], [3, 4, 5]])
    if req.status_code != 200:
        raise Exception("STATUS %d: %s" % (req.status_code, req.text))
    j = req.json()
    print(j)
    assert j['result'] == 15
    assert float(".".join(j['version'].split('.')[:2])) >= 1.19

    req = post("run/pandas18", [[1, 2, 3],[1, 2, 3]])
    if req.status_code != 200:
        raise Exception("STATUS %d: %s" % (req.status_code, req.text))
    j = req.json()
    assert j['result'] == 12
    assert j['version'].startswith('1.18')

def stress_one_lambda_task(args):
    start, seconds = args
    pos = 0
    while time.time() < start + seconds:
        req = post("run/echo", pos)
        raise_for_status(req)
        assert req.text == str(pos)
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
    # TODO: do in parallel
    start = time.time()
    for pos in range(lambda_count):
        req = post("run/L%d"%pos, {"alloc_mb": alloc_mb})
        raise_for_status(req)
        assert req.text == str(pos)
    seconds = time.time() - start

    return {"reqs_per_sec": lambda_count/seconds}

def call_each_once(lambda_count, alloc_mb=0):
    with tempfile.TemporaryDirectory() as reg_dir:
        # create dummy lambdas
        for pos in range(lambda_count):
            with open(os.path.join(reg_dir, "L%d.py"%pos), "w") as code:
                code.write("def f(event):\n")
                code.write("    global s\n")
                code.write("    s = '*' * %d * 1024**2\n" % alloc_mb)
                code.write("    return %d\n" % pos)

        with test_conf(registry=reg_dir):
            call_each_once_exec(lambda_count=lambda_count, alloc_mb=alloc_mb)

@test
def fork_bomb():
    limit = CURR_CONF["limits"]["procs"]
    req = post("run/fbomb", {"times": limit*2})
    raise_for_status(req)
    # the function returns the number of children that we were able to fork
    actual = int(req.text)
    assert 1 <= actual <= limit

@test
def max_mem_alloc():
    limit = CURR_CONF["limits"]["mem_mb"]
    req = post("run/max_mem_alloc", None)
    raise_for_status(req)
    # the function returns the MB that was able to be allocated
    actual = int(req.text)
    assert limit-16 <= actual <= limit

@test
def ping_test():
    pings = 1000
    start = time.time()
    for _ in range(pings):
        req = requests.get("http://localhost:5000/status")
        raise_for_status(req)
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
    req = post("run/rust-hashing", {"num_hashes": 100, "input_len": 1024})
    if req.status_code != 200:
        raise Exception("STATUS %d: %s" % (req.status_code, req.text))

@test
def update_code():
    reg_dir = CURR_CONF['registry']
    cache_seconds = CURR_CONF['registry_cache_ms'] / 1000

    for pos in range(3):
        # update function code
        with open(os.path.join(reg_dir, "version.py"), "w") as code:
            code.write("def f(event):\n")
            code.write("    return %d\n" % pos)

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
    parent = ""
    for _ in range(depth):
        req = post("create", {"code": "", "leaf": False, "parent": parent})
        raise_for_status(req)
        if parent:
            # don't need this parent any more, so pause it to get
            # memory back (so we can run this test with low memory)
            post("pause/"+parent)
        parent = req.text.strip()

    req = post("destroy/1", None)
    raise_for_status(req)
    req = post("stats", None)
    raise_for_status(req)
    destroys = req.json()['Destroy():ms.cnt']
    assert destroys == depth


def run_tests(sandboxes):
    test_reg = os.path.abspath("test-registry")

    for sandbox in sandboxes:
        print("Testing backend '%s'" % sandbox)

        with test_conf(sandbox=sandbox):
            with test_conf(registry=test_reg):
                ping_test()

                # test very basic rust program
                hello_rust()

                # run some more computation in rust
                rust_hashing()

                # do smoke tests under various configs
                with test_conf(features={"import_cache": False}):
                    install_tests()
                with test_conf(mem_pool_mb=500):
                    install_tests()
                with test_conf(sandbox="docker", features={"import_cache": False}):
                    install_tests()

                # test resource limits
                fork_bomb()
                max_mem_alloc()

                # numpy pip install needs a larger mem cap
                with test_conf(mem_pool_mb=500):
                    numpy_test()

            # make sure code updates get pulled within the cache time
            with tempfile.TemporaryDirectory() as reg_dir:
                with test_conf(registry=reg_dir, registry_cache_ms=3000):
                    update_code()

            # test heavy load
            with test_conf(registry=test_reg):
                stress_one_lambda(procs=1, seconds=15)
                stress_one_lambda(procs=2, seconds=15)
                stress_one_lambda(procs=8, seconds=15)

            with test_conf(features={"reuse_cgroups": True}):
                call_each_once(lambda_count=100, alloc_mb=1)
                call_each_once(lambda_count=1000, alloc_mb=10)

    if "sock" in sandboxes:
        print("Testing SOCK directly (without lambdas)")

        with test_conf(server_mode="sock", mem_pool_mb=500):
            sock_churn(baseline=0, procs=1, seconds=5, fork=False)
            sock_churn(baseline=0, procs=1, seconds=10, fork=True)
            sock_churn(baseline=0, procs=15, seconds=10, fork=True)
            sock_churn(baseline=32, procs=1, seconds=10, fork=True)
            sock_churn(baseline=32, procs=15, seconds=10, fork=True)

def main():
    global OLDIR
    global TEST_FILTER

    parser = argparse.ArgumentParser(description='Run tests for OpenLambda')
    parser.add_argument('--reuse_config', action="store_true")
    parser.add_argument('--sandboxes', type=str, default="docker,sock")
    parser.add_argument('--test_filter', type=str, default="")
    parser.add_argument('--ol_dir', type=str, default="test-dir")

    args = parser.parse_args()

    TEST_FILTER = [name for name in args.test_filter.split(",") if name != '']
    OLDIR = args.ol_dir

    print("Test filter is '%s' and OL directory is '%s'" % (TEST_FILTER, OLDIR))

    sandboxes = [name for name in args.sandboxes.split(",") if name != '']

    if len(sandboxes) == 0:
        raise RuntimeError("No server modes specified")

    start = time.time()

    # so our test script doesn't hang if we have a memory leak
    timer_thread = threading.Thread(target=ol_oom_killer, daemon=True)
    timer_thread.start()

    if os.path.exists(OLDIR):
        try:
            run(['./ol', 'kill', '-p='+OLDIR])
            print("stopped existing worker")
        except Exception as err:
            print('could not kill existing worker: %s' % str(err))

    # general setup
    if not args.reuse_config:
        if os.path.exists(OLDIR):
            run(['rm', '-rf', OLDIR])

        run(['./ol', 'new', '-p='+OLDIR])
    else:
        if os.path.exists(OLDIR):
            # Make sure the pid file is gone even if the previous worker crashed
            try:
                run(['rm', '-rf', '%s/worker' % OLDIR])
            except:
                pass

    # run tests with various configs
    with test_conf(limits={"installer_mem_mb": 250}):
        run_tests(sandboxes)

    # save test results
    passed = len([t for t in results["runs"] if t["pass"]])
    failed = len([t for t in results["runs"] if not t["pass"]])
    results["passed"] = passed
    results["failed"] = failed
    results["seconds"] = time.time() - start
    print("PASSED: %d, FAILED: %d" % (passed, failed))

    with open("test.json", "w") as resultsfile:
        json.dump(results, resultsfile, indent=2)

    sys.exit(failed)


if __name__ == '__main__':
    main()
