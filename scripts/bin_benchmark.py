#! /bin/env python3

''' Benchmarks web assembly against SOCK and Docker '''

# pylint: disable=global-statement,missing-function-docstring,consider-using-with

import sys
import argparse

from multiprocessing import Process
from time import time

from helper import DockerWorker, SockWorker, WasmWorker, prepare_open_lambda, setup_config

OUTFILE = None
NUM_WARMUPS = None
NUM_RUNS = None
WORKER_TYPES = []
BENCH_FILTER = []

def bench_in_filter(name, bench_filter):
    if len(bench_filter) == 0:
        return True

    for fname in bench_filter:
        if fname in name:
            return True

    return False

def benchmark(num_threads=1):
    def inner_function(func):
        def wrapper(*args, **_kwargs):
            if len(args) > 0:
                raise Exception("positional args not supported")

            name = func.__name__

            if not bench_in_filter(name, BENCH_FILTER):
                print(f'Skipping test f"{name}"')
                return

            for worker_type in WORKER_TYPES:
                worker = worker_type()
                fargs = [worker_type]+list(args)
                print("Worker started")

                for _ in range(NUM_WARMUPS):
                    sys.stdout.write(
                        f'Running benchmark "{name}" with '
                        f'backend "{worker_type.name()}" (warmup) ...'
                    )

                    func(*fargs)
                    print("Done.")

                for _ in range(NUM_RUNS):
                    sys.stdout.write(
                        f'Running benchmark "{name}" with '
                        f'backend "{worker_type.name()}...'
                    )

                    start = time()
                    tasks = []
                    for _ in range(num_threads):
                        proc = Process(target=func, args=fargs)
                        proc.start()
                        tasks.append(proc)

                    for task in tasks:
                        task.join()
                    end = time()

                    elapsed = (end - start) * 1000.0
                    print(f"Done. (Elapsed time {elapsed}ms)")

                    OUTFILE.write(f"{name}, {worker_type.name()}, {elapsed}\n")

                worker.stop()

        return wrapper
    return inner_function

@benchmark()
def hello(worker):
    worker.run('hello', [])

@benchmark()
def get_put1(worker):
    worker.run('get_put',
               {"num_gets":1, "num_puts":1, "num_deletes":1, "entry_size": 1000*1000}
    )

@benchmark()
def get_put100(worker):
    worker.run('get_put',
               {"num_gets":100, "num_puts":100, "num_deletes":100, "entry_size": 10*1000}
    )

@benchmark(num_threads=10)
def concurrent_get_put100(worker):
    worker.run('get_put', {"num_puts":100, "num_gets": 0, "num_deletes": 0, "entry_size": 10*1000})

@benchmark()
def hash100(worker):
    worker.run('hashing', {"num_hashes": 100, "input_len": 1024})

@benchmark()
def hash10000(worker):
    worker.run('hashing', {"num_hashes": 10*1000, "input_len": 1024})

@benchmark()
def hash100000(worker):
    worker.run('hashing', {"num_hashes": 100*1000, "input_len": 1024})

def main():
    global BENCH_FILTER
    global OUTFILE
    global NUM_RUNS
    global NUM_WARMUPS
    global WORKER_TYPES

    parser = argparse.ArgumentParser(
        description='Run benchmarks between native containers and WebAssembly')
    parser.add_argument('--bench_filter', type=str, default="")
    parser.add_argument('--num_warmups', type=int, default=3)
    parser.add_argument('--num_runs', type=int, default=20)
    parser.add_argument('--reuse_config', action='store_true')
    parser.add_argument('--ol_dir', type=str, default='bench-dir')
    parser.add_argument('--worker_types', type=str, default="datastore,container,wasm")

    args = parser.parse_args()

    setup_config(args.ol_dir)

    BENCH_FILTER = [name for name in args.bench_filter.split(",") if name != '']
    NUM_WARMUPS = args.num_warmups
    NUM_RUNS = args.num_runs
    WORKER_TYPES = []

    worker_names = [name for name in args.worker_types.split(",") if name != '']

    if 'sock' in worker_names:
        WORKER_TYPES.append(SockWorker)
    if 'wasm' in worker_names:
        WORKER_TYPES.append(WasmWorker)
    if 'docker' in worker_names:
        WORKER_TYPES.append(DockerWorker)

    OUTFILE = open("./bench-results.csv", 'w', encoding='utf-8')
    OUTFILE.write("bench_name, worker_type, elapsed\n")

    prepare_open_lambda(args.ol_dir, reuse_config=args.reuse_config)

    # pylint: disable=no-value-for-parameter
    hello()
    hash100()
    hash10000()
    hash100000()
    # pylint: enable=no-value-for-parameter

if __name__ == '__main__':
    main()
