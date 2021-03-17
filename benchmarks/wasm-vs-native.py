#! /bin/env python3

import argparse

from time import time
from helper import *

OUTFILE=None
NUM_RUNS=3

def benchmark(fn):
    def wrapper(*args, **kwargs):
        global OUTFILE

        if len(args):
            raise Exception("positional args not supported")

        name = fn.__name__

        if not bench_in_filter(name):
            print("Skipping test '%s'" % name)
            return None

        for Worker in [ContainerWorker, WasmWorker]:
            worker = Worker()
            fargs = [worker]+list(args)

            print("Running benchmark `%s`" % name)
            start = time()
            fn(*fargs)
            end = time()

            elapsed = (end - start) * 1000.0
            print("Done. (Elapsed time %fms)", elapsed)

            OUTFILE.write("%s, %s, %f\n" % (name, worker.name(), elapsed))

            worker.stop()

    return wrapper

@benchmark
def hello(worker):
    worker.run('hello', [])

def main():
    global BENCH_FILTER
    global OUTFILE
    global NUM_RUNS

    parser = argparse.ArgumentParser(description='Run benchmarks between native containers and WebAssembly')
    parser.add_argument('--bench_filter', type=str, default="")
    parser.add_argument('--num_runs', type=int, default=3)
    parser.add_argument('--reuse_config', action='store_true')

    args = parser.parse_args()
    BENCH_FILTER = [name for name in args.bench_filter.split(",") if name != '']
    NUM_RUNS=args.num_runs

    OUTFILE = open("./bench-results.csv", 'w')
    OUTFILE.write("bench_name, worker_type, elapsed\n")

    prepare_open_lambda(reuse_config=args.reuse_config)

    hello()

if __name__ == '__main__':
    main()
