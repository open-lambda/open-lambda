''' Utility functions for all test scripts '''

# pylint: disable=global-statement,relative-beyond-top-level,too-many-statements,broad-except

from time import time
from collections import OrderedDict
from threading import Thread

import sys
import json
import traceback

from . import ol_oom_killer, mounts, get_ol_stats, get_current_config, get_worker_output

TEST_FILTER = []
TEST_BLOCKLIST = []
WORKER_TYPE = []
RESULTS = OrderedDict({"runs": []})
START_TIME = None

def set_worker_type(new_val):
    ''' Setup up the worker type for all following tests '''
    global WORKER_TYPE
    WORKER_TYPE = new_val

def set_test_filter(new_val):
    ''' Sets up the filter for all following tests '''

    global TEST_FILTER
    TEST_FILTER = new_val

def set_test_blocklist(new_val):
    ''' Sets up the blocklist for all following tests '''

    global TEST_BLOCKLIST
    TEST_BLOCKLIST = new_val

def start_tests():
    ''' Starts the background logic for a test run '''

    # so our test script does not hang if we have a memory leak
    timer_thread = Thread(target=ol_oom_killer, daemon=True)
    timer_thread.start()

    global START_TIME
    START_TIME = time()

def check_test_results():
    ''' Store the test results in a file an terminates the program '''
    results = RESULTS
    passed = len([t for t in results["runs"] if t["pass"]])
    failed = len([t for t in results["runs"] if not t["pass"]])
    elapsed = time() - START_TIME

    results["passed"] = passed
    results["failed"] = failed
    results["seconds"] = elapsed

    if failed:
        failed_names = ", ".join([t["test"] for t in results["runs"] if not t["pass"]])
        print("Failing tests: " + failed_names)
    print(f"PASSED: {passed}, FAILED: {failed}, ELAPSED: {elapsed}")

    with open("test.json", "w", encoding='utf-8') as resultsfile:
        json.dump(results, resultsfile, indent=2)

    sys.exit(failed)

def _test_in_filter(name):
    if name in TEST_BLOCKLIST:
        return False

    if len(TEST_FILTER) == 0:
        return True

    return name in TEST_FILTER

def test(func):
    ''' Boilerplate code for tests '''

    def _wrapper(*args, **kwargs):
        if len(args) > 0:
            raise RuntimeError("positional args not supported for tests")

        name = func.__name__

        if not _test_in_filter(name):
            print(f'Skipping test "{name}"')
            return None

        print('='*40)
        if len(kwargs) > 0:
            print(name, kwargs)
        else:
            print(name)
        print('='*40)
        result = OrderedDict()
        result["test"] = name
        result["params"] = kwargs
        result["pass"] = None
        result["conf"] = get_current_config()
        result["test_seconds"] = None
        result["total_seconds"] = None
        result["stats"] = None
        result["ol-stats"] = None
        result["errors"] = []
        result["worker_tail"] = None

        total_t0 = time()
        mounts0 = mounts()
        worker = None
        return_val = None

        worker = WORKER_TYPE()
        assert worker
        print("Worker started")

        if worker:
            try:
                # run test/benchmark
                test_t0 = time()
                return_val = func(**kwargs)
                test_t1 = time()
                result["test_seconds"] = test_t1 - test_t0
                result["pass"] = True
            except Exception as err:
                print(f"Failed to run test: {err}")
                result["pass"] = False
                result["errors"].append(traceback.format_exc().split("\n"))

            worker.stop()

        mounts1 = mounts()
        if len(mounts0) != len(mounts1):
            result["pass"] = False
            result["errors"].append([
                f"mounts are leaking ({len(mounts0)} before, "
                f"{len(mounts1)} after), leaked: {mounts1 - mounts0}"
            ])

        # get internal stats from OL
        result["ol-stats"] = get_ol_stats()

        total_t1 = time()
        result["total_seconds"] = total_t1-total_t0
        result["stats"] = return_val

        if result["pass"]:
            # truncate because we probably won't use it for debugging
            result["worker_tail"] = get_worker_output()[-10:]
        else:
            result["worker_tail"] = get_worker_output()

        RESULTS["runs"].append(result)

        if result["pass"]:
            subset = {
                "test": result["test"],
                "params": result["params"],
                "pass": result["pass"],
                "test_seconds": result["test_seconds"],
                "stats": result["stats"],
            }
            print(json.dumps(subset, indent=2))
        else:
            print(json.dumps(result, indent=2))

        return return_val

    return _wrapper
