#! /bin/env python3

'''
SOCK-specific tests
'''

#pylint: disable=global-statement,too-many-statements,missing-function-docstring,wrong-import-position

import argparse
import os
import sys
import subprocess

from time import time

from multiprocessing import Pool

from helper import SockWorker, prepare_open_lambda, setup_config, TestConfContext
from helper.test import set_test_filter, start_tests, check_test_results, set_worker_type, test

# You can either install the OpenLambda Python bindings
# or run the test from the project's root folder
sys.path.append('python/src')
from open_lambda import OpenLambda

@test
def install_examples_to_worker_registry():
    """Install all lambda functions from examples directory to worker registry using admin install"""
    examples_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "examples")
    
    if not os.path.exists(examples_dir):
        print(f"Examples directory not found at {examples_dir}")
        return
    
    # Get all directories in examples
    example_functions = []
    for item in os.listdir(examples_dir):
        item_path = os.path.join(examples_dir, item)
        if os.path.isdir(item_path):
            # Check if it has f.py (required for lambda functions)
            if os.path.exists(os.path.join(item_path, "f.py")):
                example_functions.append(item_path)
    
    print(f"Found {len(example_functions)} lambda functions in examples directory")
    
    # Install each function using admin install command
    # Find the ol binary - it should be in the project root
    project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    ol_binary = os.path.join(project_root, "ol")
    
    if not os.path.exists(ol_binary):
        print(f"✗ OL binary not found at {ol_binary}")
        return
    
    # Get OL_DIR from the global args - for sock_test we'll use test-dir
    ol_dir = "test-dir"
    
    for func_dir in example_functions:
        func_name = os.path.basename(func_dir)
        print(f"Installing {func_name} from {func_dir}")
        
        try:
            # Run ol admin install -p <worker_path> <function_directory>
            result = subprocess.run([ol_binary, "admin", "install", f"-p={ol_dir}", func_dir], 
                                  capture_output=True, text=True, cwd=project_root)
            
            if result.returncode == 0:
                print(f"✓ Successfully installed {func_name}")
            else:
                print(f"✗ Failed to install {func_name}: {result.stderr}")
                
        except Exception as e:
            print(f"✗ Error installing {func_name}: {e}")
    
    print("Finished installing example functions")

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

    echo_path = os.path.abspath("registry/echo.tar.gz")
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
        install_examples_to_worker_registry()
        sock_churn(baseline=0, procs=1, seconds=5, fork=False)
        sock_churn(baseline=0, procs=1, seconds=10, fork=True)
        sock_churn(baseline=0, procs=15, seconds=10, fork=True)
        sock_churn(baseline=32, procs=1, seconds=10, fork=True)
        sock_churn(baseline=32, procs=15, seconds=10, fork=True)

def main():
    parser = argparse.ArgumentParser(description='Run SOCK-specific tests for OpenLambda')
    parser.add_argument('--test_filter', type=str, default="")
    parser.add_argument('--ol_dir', type=str, default="test-dir")
    parser.add_argument('--registry', type=str, default="registry")

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
