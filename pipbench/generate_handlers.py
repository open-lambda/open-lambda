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


def write_handlers(handlers_dir, graph, pop, num_imports, duplicates, num_handlers):
    pkgs = set()
    refs = {pkg_name: 0  for pkg_name in graph}
    num_pkgs = len(graph)
    pkgs_pops_dist = []
    pkg_by_ind = list(pop.keys())

    i = 0
    for pkg_name in pkg_by_ind:
        i += 1
        pkg_pop = pop[pkg_name]
        for j in range(pkg_pop):
            pkgs_pops_dist.append(i)

    for h_i in range(num_handlers):
        # TODO: memory is hard-coded to 1MB
        mem = 1024
        dep_names = []
        for i in range(num_imports[numpy.random.randint(0, len(num_imports))]):
            pkgidx = pkgs_pops_dist[numpy.random.randint(0, num_pkgs)]
            pkg = pkg_by_ind[pkgidx]
            dep_names.append(pkg)
            refs[pkg] += 1
        imps = set(dep_names)
        deps = set.union(imps, *[graph[imp] for imp in imps])
        for idx in range(duplicates):
            handler_name = '%shdl%d' % (h_i, idx)
            handler = Handler(handler_name, imps, deps, mem)
            write_handler(handlers_dir, handler)
        pkgs |= deps

    return pkgs, refs


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
    parser.add_argument('-d', '--duplicates', type=int, default=10, help='number of duplicate handlers for each package')
    parser.add_argument('-n', '--num-handlers', type=int, default=1, help='number of handlers to create')
    parser.add_argument('zipf_arg', type=float, help='argument to the zipfian distribution')
    parser.add_argument('rand_seed', type=int, help='random number generator seed')
    parser.add_argument('dep_dist', help='number of imports per handler dist')
    args = parser.parse_args()

    numpy.random.seed(args.rand_seed)
    random.seed(args.rand_seed)

    with open(args.spec_file) as spec_file:
        spec = json.load(spec_file)
    graph = {entry['name']: set(entry['deps']) for entry in spec}
    pop = {entry['name']: numpy.random.zipf(args.zipf_arg) for entry in spec}
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
    pkgs, refs = write_handlers(handlers_dir, graph, pop, num_imports, args.duplicates, args.num_handlers)
    print('Writing out packages used...')
    spec = {entry['name']: entry for entry in spec}
    with open('packages_and_size.txt', 'w') as f:
        for pkg in pkgs:
            f.write('%s:%d\n' % (pkg, spec[pkg]['uncompressed']))
    print('Writing out package handler reference counts')
    with open('packages_handler_refcounts.txt', 'w') as f:
        refs_list = sorted(refs.items(), key=operator.itemgetter(1))
        for rc in refs_list:
            f.write('%s %d\n' %(rc[0], rc[1]))
    print('Done')


if __name__ == '__main__':
    main()
