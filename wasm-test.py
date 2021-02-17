#!/usr/bin/env python3

import argparse
import os, sys, json, time, requests, copy, traceback, tempfile, threading, subprocess
from collections import OrderedDict
from subprocess import check_output
from multiprocessing import Pool
from contextlib import contextmanager

# These will be set by argparse in main()
TEST_FILTER = []

results = OrderedDict({"runs": []})
curr_conf = None

''' Issues a post request to the OL worker '''
def post(path, data=None):
    return requests.post('http://localhost:5000/'+path, json.dumps(data))


def raise_for_status(r):
    if r.status_code != 200:
        raise Exception("STATUS %d: %s" % (r.status_code, r.text))

def test_in_filter(name):
    if len(TEST_FILTER) == 0:
        return True

    return name in TEST_FILTER

def get_mem_stat_mb(stat):
    with open('/proc/meminfo') as f:
        for l in f:
            if l.startswith(stat+":"):
                parts = l.strip().split()
                assert(parts[-1] == 'kB')
                return int(parts[1]) / 1024
    raise Exception('could not get stat')

def ol_oom_killer():
    while True:
        if get_mem_stat_mb('MemAvailable') < 128:
            print("out of memory, trying to kill OL")
            os.system('pkill ol')
        time.sleep(1)

def test(fn):
    def wrapper(*args, **kwargs):
        if len(args):
            raise Exception("positional args not supported for tests")

        name = fn.__name__

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
        result["conf"] = curr_conf
        result["seconds"] = None
        result["total_seconds"] = None
        result["stats"] = None
        result["ol-stats"] = None
        result["errors"] = []
        result["worker_tail"] = None

        total_t0 = time.time()
        worker = None

        try:
            # setup worker
            worker = Popen(['./ol-wasm'])

            # run test/benchmark
            test_t0 = time.time()
            rv = fn(**kwargs)
            test_t1 = time.time()
            result["seconds"] = test_t1 - test_t0

            result["pass"] = True
        except Exception:
            rv = None
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
        result["total_seconds"] = total_t1-total_t0
        result["stats"] = rv
        results["runs"].append(result)

        print(json.dumps(result, indent=2))
        return rv

    return wrapper


def put_conf(conf):
    pass

''' Loads a config and overwrites certain fields with what is set in **keywords '''
@contextmanager

@test
def wasm_numpy_test():
    r = post("run/numpy", [1, 2])
    if r.status_code != 200:
        raise Exception("STATUS %d: %s" % (r.status_code, r.text))
    j = r.json()
    assert j['result'] == 3

    r = post("run/numpy", [[1, 2], [3, 4]])
    if r.status_code != 200:
        raise Exception("STATUS %d: %s" % (r.status_code, r.text))
    j = r.json()
    assert j['result'] == 10

    r = post("run/numpy", [[[1, 2], [3, 4]], [[1, 2], [3, 4]]])
    if r.status_code != 200:
        raise Exception("STATUS %d: %s" % (r.status_code, r.text))
    j = r.json()
    assert j['result'] == 20

@test
def ping_test():
    pings = 1000
    t0 = time.time()
    for i in range(pings):
        r = requests.get("http://localhost:5000/status")
        raise_for_status(r)
    seconds = time.time() - t0
    return {"pings_per_sec": pings/seconds}

def run_tests():
    test_reg = os.path.abspath("test-registry")

    print("Testing WASM")

    ping_test()
    wasm_numpy_test()

def main():
    global TEST_FILTER

    parser = argparse.ArgumentParser(description='Run tests for OpenLambda')
    parser.add_argument('--test_filter', type=str, default="")

    args = parser.parse_args()

    TEST_FILTER = [name for name in args.test_filter.split(",") if name != '']

    print("Test filter is '%s'" % TEST_FILTER)

    t0 = time.time()

    # so our test script doesn't hang if we have a memory leak
    timerThread = threading.Thread(target=ol_oom_killer, daemon=True)
    timerThread.start()

    # run tests with various configs
    run_tests()

    # save test results
    passed = len([t for t in results["runs"] if t["pass"]])
    failed = len([t for t in results["runs"] if not t["pass"]])
    results["passed"] = passed
    results["failed"] = failed
    results["seconds"] = time.time() - t0
    print("PASSED: %d, FAILED: %d" % (passed, failed))

    with open("test.json", "w") as f:
        json.dump(results, f, indent=2)

    sys.exit(failed)


if __name__ == '__main__':
    main()
