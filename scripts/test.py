#!/usr/bin/env python3

''' Various integration tests for the open lambda framwork '''

# pylint: disable=global-statement,too-many-statements,fixme
# pylint: disable=broad-except,too-many-locals
# pylint: disable=missing-function-docstring,wrong-import-position

import argparse
import os
import sys
import tempfile
import tarfile
import subprocess

from time import time
from subprocess import call
from multiprocessing import Pool

import requests

from helper import DockerWorker, SockWorker, prepare_open_lambda, setup_config
from helper import get_current_config, TestConfContext, assert_true, assert_eq

from helper.test import (
    set_test_filter,
    set_test_blocklist,
    start_tests,
    check_test_results,
    set_worker_type,
    get_worker_type,
    test,
)

# You can either install the OpenLambda Python bindings
# or run the test from the project's root folder
sys.path.append('python/src')
from open_lambda import OpenLambda

# These will be set by argparse in main()
OL_DIR = None

def install_examples_to_worker_registry():
    """Install all lambda functions from examples directory to
    worker registry using admin install"""
    examples_dir = os.path.join(
        os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "examples"
    )
    if not os.path.exists(examples_dir):
        print(f"Examples directory not found at {examples_dir}")
        return
    # Get all directories in examples (each directory is a lambda function)
    example_functions = []
    for item in os.listdir(examples_dir):
        item_path = os.path.join(examples_dir, item)
        if os.path.isdir(item_path):
            example_functions.append(item_path)
    print(f"Found {len(example_functions)} lambda functions in examples directory")
    # Install each function using admin install command
    # Find the ol binary - it should be in the project root
    project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    ol_binary = os.path.join(project_root, "ol")
    if not os.path.exists(ol_binary):
        print(f"✗ OL binary not found at {ol_binary}")
        return
    for func_dir in example_functions:
        func_name = os.path.basename(func_dir)
        print(f"Installing {func_name} from {func_dir}")
        try:
            # Run ol admin install -p <worker_path> <function_directory>
            result = subprocess.run([ol_binary, "admin", "install", f"-p={OL_DIR}", func_dir],
                                capture_output=True, text=True, cwd=project_root)
            if result.returncode == 0:
                print(f"✓ Successfully installed {func_name}")
            else:
                print(f"✗ Failed to install {func_name}: {result.stderr}")
                raise Exception(f"install failed for {func_name}")
        except Exception as e:
            print(f"✗ Error installing {func_name}: {e}")
            raise e
    print("Finished installing example functions")


@test
def install_tests():
    # we want to make sure we see the expected number of pip installs,
    # so we don't want installs lying around from before
    return_code = call(['rm', '-rf', f'{OL_DIR}/lambda/packages/*'])
    assert_eq(return_code, 0)

    open_lambda = OpenLambda()

    # try something that doesn't install anything
    msg = 'hello world'
    jdata = open_lambda.run("echo", msg)
    if jdata != msg:
        raise ValueError(f"found {jdata} but expected {msg}")

    jdata = open_lambda.get_statistics()
    installs = jdata.get('pull-package.cnt', 0)
    assert_eq(installs, 0)

    for pos in range(3):
        name = f"install{pos+1}"
        result = open_lambda.run(name, {})
        assert_eq(result, "imported")

        result = open_lambda.get_statistics()

        installs = result['pull-package.cnt']
        if pos < 2:
            # with deps, requests should give us these:
            # certifi, charset-normalizer, idna, requests, urllib3
            assert_eq(installs, 5)
        else:
            # requests (and deps) + simplejson
            assert_eq(installs, 6)


def check_status_code(req):
    if req.status_code != 200:
        raise requests.HTTPError(f"STATUS {req.status_code}: {req.text}")


@test
def numpy_test():
    open_lambda = OpenLambda()

    # try adding the nums in a few different matrixes.  Also make sure
    # we can have two different numpy versions co-existing.
    result = open_lambda.run("numpy21", [1, 2])
    assert_eq(result['result'], 3)
    assert_true(result['numpy-version'].startswith('2.1'))

    result = open_lambda.run("numpy22", [[1, 2], [3, 4]])
    assert_eq(result['result'], 10)
    assert_true(result['numpy-version'].startswith('2.2'))

    result = open_lambda.run("numpy22", [[[1, 2], [3, 4]], [[1, 2], [3, 4]]])
    assert_eq(result['result'], 20)
    assert_true(result['numpy-version'].startswith('2.2'))

    result = open_lambda.run("pandas", [[0, 1, 2], [3, 4, 5]])
    assert_eq(result['result'], 15)
    assert_true(float(".".join(result['numpy-version'].split('.')[:2])) >= 2.2)

    result = open_lambda.run("pandas-v1", [[1, 2, 3], [1, 2, 3]])
    assert_eq(result['result'], 12)
    assert_true(result['numpy-version'].startswith('1.26'))

def stress_one_lambda_task(args):
    open_lambda = OpenLambda()

    start, seconds = args
    pos = 0
    while time() < start + seconds:
        result = open_lambda.run("echo", pos, json=False)
        assert_eq(result, str(pos))
        pos += 1
    return pos

@test
def stress_one_lambda(procs, seconds):
    start = time()

    with Pool(procs) as pool:
        reqs = sum(pool.map(stress_one_lambda_task, [(start, seconds)] * procs, chunksize=1))

    return {"reqs_per_sec": reqs/seconds}

@test
def call_each_once_exec(lambda_count, alloc_mb, zygote_provider):
    with TestConfContext(features={"import_cache": zygote_provider}):
        open_lambda = OpenLambda()

        # TODO: do in parallel
        start = time()
        for pos in range(lambda_count):
            result = open_lambda.run(f"L{pos}", {"alloc_mb": alloc_mb}, json=False)
            assert_eq(result, str(pos))
            seconds = time() - start

            return {"reqs_per_sec": lambda_count/seconds}

def call_each_once(lambda_count, alloc_mb=0, zygote_provider="tree"):
    with tempfile.TemporaryDirectory() as reg_dir:
        # create dummy lambdas as tar.gz files
        for pos in range(lambda_count):
            # Create temporary directory for lambda contents
            with tempfile.TemporaryDirectory() as lambda_dir:
                # Write f.py file
                with open(os.path.join(lambda_dir, "f.py"), "w", encoding='utf-8') as code:
                    code.write("def f(event):\n")
                    code.write("    global s\n")
                    code.write(f"    s = '*' * {alloc_mb} * 1024**2\n")
                    code.write(f"    return {pos}\n")
                # Create tar.gz file
                tar_path = os.path.join(reg_dir, f"L{pos}.tar.gz")
                with tarfile.open(tar_path, "w:gz") as tar:
                    tar.add(os.path.join(lambda_dir, "f.py"), arcname="f.py")

        with TestConfContext(registry=reg_dir):
            call_each_once_exec(lambda_count=lambda_count, alloc_mb=alloc_mb,
                                zygote_provider=zygote_provider)

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
    start = time()
    for _ in range(pings):
        open_lambda.check_status()

    seconds = time() - start
    return {"pings_per_sec": pings/seconds}

@test
def update_code():
    curr_conf = get_current_config()
    reg_dir = curr_conf['registry']
    cache_seconds = curr_conf['registry_cache_ms'] / 1000

    open_lambda = OpenLambda()

    for pos in range(3):
        # update function code in tar.gz format
        with tempfile.TemporaryDirectory() as lambda_dir:
            # Write f.py file
            with open(os.path.join(lambda_dir, "f.py"), "w", encoding='utf-8') as code:
                code.write("def f(event):\n")
                code.write(f"    return {pos}\n")
            # Create tar.gz file
            tar_path = os.path.join(reg_dir, "version.tar.gz")
            with tarfile.open(tar_path, "w:gz") as tar:
                tar.add(os.path.join(lambda_dir, "f.py"), arcname="f.py")

        # how long does it take for us to start seeing the latest code?
        start = time()
        while True:
            text = open_lambda.run("version", None)
            num = int(text)
            assert num >= pos-1
            end = time()

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
        result = open_lambda.create({"code": "", "leaf": False, "parent": parent})
        if parent:
            # don't need this parent any more, so pause it to get
            # memory back (so we can run this test with low memory)
            open_lambda.pause(parent)
        parent = result.strip()

    open_lambda.destroy("1")

    stats = open_lambda.get_statistics()
    destroys = stats['Destroy():ms.cnt']
    assert_eq(destroys, depth)

@test
def flask_test():
    url = 'http://localhost:5000/run/flask-test'
    print("URL", url)
    r = requests.get(url)
    print("RESPONSE", r)

    # flask apps should have control of status code, headers, and response body
    if r.status_code != 418:
        raise ValueError(f"expected status code 418, but got {r.status_code}")
    if not "A" in r.headers:
        raise ValueError(f"'A' not found in headers, as expected: {r.headers}")
    if r.headers["A"] != "B":
        raise ValueError(f"headers['A'] should be 'B', not {r.headers['A']}")
    if r.text != "hi\n":
        raise ValueError(f"r.text should be 'hi\n', not {repr(r.text)}")

@test
def flask_entry_test():
    """Test OL_ENTRY_FILE feature with a Flask app using app.py instead of f.py"""
    # Test the index route
    url = 'http://localhost:5000/run/flask-entry-test'
    print("URL", url)
    r = requests.get(url)
    print("RESPONSE", r)

    if r.status_code != 200:
        raise ValueError(f"expected status code 200, but got {r.status_code}")
    if r.text != "Hello from app.py!\n":
        raise ValueError(f"r.text should be 'Hello from app.py!\\n', not {repr(r.text)}")

    # Test the info route
    url_info = 'http://localhost:5000/run/flask-entry-test/info'
    print("URL", url_info)
    r = requests.get(url_info)
    print("RESPONSE", r)

    if r.status_code != 200:
        raise ValueError(f"expected status code 200, but got {r.status_code}")
    data = r.json()
    if data.get("entry_file") != "app.py":
        raise ValueError(f"expected entry_file='app.py', got {data}")

@test
def test_http_method_restrictions():
    url = 'http://localhost:5000/run/lambda-config-test'
    print("URL", url)
    print("Testing POST request...")
    r = requests.post(url)

    if r.status_code != 418:
        raise ValueError(f"expected status code 418, but got {r.status_code}")
    if not "A" in r.headers:
        raise ValueError(f"'A' not found in headers, as expected: {r.headers}")
    if r.headers["A"] != "B":
        raise ValueError(f"headers['A'] should be 'B', not {r.headers['A']}")
    if r.text != "hi\n":
        raise ValueError(f"r.text should be 'hi\n', not {repr(r.text)}")

    # Test PUT request
    print("Testing PUT request...")
    r = requests.put(url)

    # Verify response for PUT request
    if r.status_code != 405:
        raise ValueError(f"Expected status code 405 for PUT, but got {r.status_code}")
    if r.text != "HTTP method not allowed. Sent: PUT, Allowed: [GET POST]\n":
        raise ValueError(
            f"r.text should be 'HTTP method not allowed. Sent: PUT, Allowed: [GET POST]\n' "
            f"for PUT, not {repr(r.text)}"
        )

@test
def env_test():
    """Test that environment variables from ol.yaml are properly loaded"""
    open_lambda = OpenLambda()

    # Call the env-test function
    result = open_lambda.run("env-test", {})

    # Verify that all configured environment variables are present
    expected_vars = {
        "MY_ENV_VAR": "Hello from environment",
        "DATABASE_URL": "postgresql://user:pass@localhost/db", 
        "DEBUG_MODE": "true",
        "API_KEY": "secret-key-789",
        "CUSTOM_PATH": "/usr/local/bin"
    }

    # Check that the configured_env_vars match what we expect
    if "configured_env_vars" not in result:
        raise ValueError(f"configured_env_vars not found in response: {result}")

    configured = result["configured_env_vars"]

    for key, expected_value in expected_vars.items():
        if key not in configured:
            raise ValueError(f"Environment variable {key} not found in response")
        if configured[key] != expected_value:
            raise ValueError(
                f"Environment variable {key}={configured[key]} but expected {expected_value}")

    print(f"✓ All {len(expected_vars)} environment variables loaded correctly")

    # Verify DEBUG_MODE enabled all env vars to be returned
    if "all_env_vars" not in result:
        raise ValueError("DEBUG_MODE=true but all_env_vars not returned")

    return {"env_vars_tested": len(expected_vars)}


def run_tests():
    worker_type = get_worker_type()
    worker = worker_type()
    assert worker
    print("Worker started")
    install_examples_to_worker_registry()
    print("Examples installed")
    worker.stop()

    ping_test()

    # do smoke tests under various configs
    with TestConfContext(features={"import_cache": ""}):
        install_tests()
    with TestConfContext(mem_pool_mb=1000):
        install_tests()

    # test resource limits
    fork_bomb()
    max_mem_alloc()

    # numpy pip install needs a larger memory cap.
    # numpy also spawns threads using OpenBLAS, so a higher
    # process limit is needed.
    with TestConfContext(mem_pool_mb=1000, limits={'procs': 32}, trace={"cgroups": True}):
        numpy_test()

    # make sure we can use WSGI apps based on frameworks like Flask
    flask_test()
    flask_entry_test()
    test_http_method_restrictions()

    # test environment variables from ol.yaml
    env_test()

    # make sure code updates get pulled within the cache time
    with tempfile.TemporaryDirectory() as reg_dir:
        with TestConfContext(registry=reg_dir, registry_cache_ms=3000):
            update_code()

    # test heavy load
    with TestConfContext():
        stress_one_lambda(procs=1, seconds=15)
        stress_one_lambda(procs=2, seconds=15)
        stress_one_lambda(procs=8, seconds=15)

    with TestConfContext():
        call_each_once(lambda_count=10, alloc_mb=1, zygote_provider="tree")
        call_each_once(lambda_count=100, alloc_mb=10, zygote_provider="")
        call_each_once(lambda_count=100, alloc_mb=10, zygote_provider="tree")
        call_each_once(lambda_count=100, alloc_mb=10, zygote_provider="multitree")

def main():
    global OL_DIR

    parser = argparse.ArgumentParser(description='Run tests for OpenLambda')
    parser.add_argument('--worker_type', type=str, default="sock")
    parser.add_argument('--test_filter', type=str, default="")
    parser.add_argument('--test_blocklist', type=str, default="")
    parser.add_argument('--registry', type=str, default="registry")
    parser.add_argument('--ol_dir', type=str, default="test-dir")
    parser.add_argument('--image', type=str, default="ol-wasm")

    args = parser.parse_args()

    if args.test_filter and args.test_blocklist:
        raise RuntimeError("--test_filter and --test_blocklist cannot be used together")
    if args.test_filter:
        set_test_filter([name for name in args.test_filter.split(",") if name != ''])
    elif args.test_blocklist:
        set_test_blocklist([name for name in args.test_blocklist.split(",") if name != ''])

    OL_DIR = args.ol_dir

    # Clean up any existing test directory
    if os.path.exists(args.ol_dir):
        call(['rm', '-rf', args.ol_dir])

    setup_config(args.ol_dir)
    prepare_open_lambda(args.ol_dir, args.image)

    # Use worker registry directory from config
    registry_path = "file://" + os.path.abspath(args.registry)

    trace_config = {
        "cgroups": True,
        "memory": True,
        "evictor": True,
        "package": True,
    }

    with TestConfContext(registry=registry_path, trace=trace_config):
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
