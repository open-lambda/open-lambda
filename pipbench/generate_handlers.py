import os
import shutil
import json
from helper_modules.handler import Handler
import argparse
import re
import numpy
import operator
import random


CWD = os.path.dirname(os.path.realpath(__file__))


def populate_graph(graph):
    # sort by level
    pkgs = sorted(graph, key=lambda x: int(re.match(r'level(\d+)', x).group(1)))
    for pkg in pkgs:
        if len(graph[pkg]) == 0:
            continue
        graph[pkg] |= set.union(*[graph[dep] for dep in graph[pkg]])


def pick_imports(pkgs, zipf, dep_dist):
    pkg_dist = numpy.random.zipf(zipf, len(pkgs))
    tot_zipf = sum(pkg_dist)
    pkg_dist = [float(dist) / tot_zipf for dist in pkg_dist]

    while True:
        num_deps = numpy.random.choice(dep_dist)
        yield numpy.random.choice(pkgs, num_deps, False, pkg_dist)

def write_handlers(handlers_dir, graph, zipf, dep_dist, duplicates, num_handlers):
    all_deps = set()
    refs = {pkg_name: 0  for pkg_name in graph}

    # sort keys for deterministic result
    imps_gen = pick_imports(sorted(graph), zipf, dep_dist)

    for i in range(num_handlers):
        # TODO: memory is hard-coded to 1MB
        mem = 1024
        imps = set(next(imps_gen))
        for imp in imps:
            refs[imp] += 1
        deps = set.union(imps, *[graph[imp] for imp in imps])
        for j in range(duplicates):
            handler_name = 'hdl%d_%d' % (i, j)
            handler = Handler(handler_name, imps, deps, mem)
            write_handler(handlers_dir, handler)
        all_deps |= deps

    return all_deps, refs


def write_handler(handlers_dir, handler):
    handler_dir = '%s/%s' % (handlers_dir, handler.name)
    os.makedirs(handler_dir)

    lambda_func = handler.get_lambda_func()
    with open('%s/lambda_func.py' % handler_dir, 'w') as f:
        f.write(lambda_func)

    packages_txt = handler.get_packages_txt()
    with open('%s/packages.txt' % handler_dir, 'w') as f:
        f.write(packages_txt)

    shutil.copyfile('load_simulator.so', '%s/load_simulator.so' % handler_dir)


def main():
    parser = argparse.ArgumentParser(description='Generate pipbench handlers')
    parser.add_argument('spec_file', help='json specification file of the pipbench mirror')
    parser.add_argument('dep_dist', help='number of imports per handler dist')
    parser.add_argument('-d', '--duplicates', type=int, default=1, help='number of duplicate handlers for each package')
    parser.add_argument('-n', '--num-handlers', type=int, default=100, help='number of handlers to create')
    parser.add_argument('-z', '--zipf_arg', type=float, default=1.4, help='argument to the zipfian distribution')
    parser.add_argument('-s', '--seed', type=int, default=1, help='random number generator seed')
    args = parser.parse_args()

    numpy.random.seed(args.seed)

    with open(args.spec_file) as spec_file:
        spec = json.load(spec_file)
    graph = {entry['name']: set(entry['deps']) for entry in spec}

    with open(args.dep_dist) as f:
        num_imports = list(map(int, f.read().split()))

    handlers_dir = '%s/handlers' % CWD

    os.system('gcc -fPIC -shared -I/usr/include/python2.7 -lpython2.7  load_simulator.c -o load_simulator.so')

    if not os.path.exists(handlers_dir):
        os.makedirs(handlers_dir)
    elif len(os.listdir(handlers_dir)) != 0:
        print('handlers directory is not empty')
        return

    print('Populating indirect dependencies...')
    populate_graph(graph)
    print('Writing out handlers...')
    all_deps, refs = write_handlers(handlers_dir, graph, args.zipf_arg, num_imports, args.duplicates, args.num_handlers)
    print('Writing out packages used...')
    spec = {entry['name']: entry for entry in spec}
    with open('packages_and_size.txt', 'w') as f:
        for pkg in all_deps:
            f.write('%s:%d\n' % (pkg, spec[pkg]['uncompressed']))
    print('Writing out package handler reference counts')
    with open('packages_handler_refcounts.txt', 'w') as f:
        for pkg in sorted(refs, key=refs.get):
            f.write('%s %d\n' %(pkg, refs[pkg]))
    print('Done')


if __name__ == '__main__':
    main()
