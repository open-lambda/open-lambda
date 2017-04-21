import os
import shutil
import json
from helper_modules.handler import Handler
import argparse
import re
import numpy
import operator

CWD = os.path.dirname(os.path.realpath(__file__))


def populate_graph(graph):
    # sort by level
    pkgs = sorted(graph, key=lambda x: int(re.match(r'level(\d+)', x).group(1)))
    for pkg in pkgs:
        if len(graph[pkg]) == 0:
            continue
        graph[pkg] |= set.union(*[graph[dep] for dep in graph[pkg]])


def write_handlers(handlers_dir, graph, pop, refs, duplicates, num_pkgs):
    if num_pkgs < 0:
        num_pkgs = len(graph)
    count = 0
    pkgs = set()
    pkgs_indexed = [pkg for pkg in graph]

    num_pkgs = len(graph)
    total_pop = 0
    for pkg in pop:
        total_pop += pop[pkg]
    for pkg in graph:
        if count >= num_pkgs:
            break
        # TODO: memory is hard-coded to 1MB
        mem = 1024
        if True:
            num_pkgs = len(graph)
            while True:
                i = numpy.random.randint(0, num_pkgs)
                if pop[pkgs_indexed[i]] / total_pop > numpy.random.random():
                    imps = set([pkgs_indexed[i]])
                    refs[pkgs_indexed[i]] += 1
                    break
        else:
            # TODO: for now a handler only imports one package
            imps = set([pkg])
        deps = set.union(imps, *[graph[imp] for imp in imps])
        for idx in range(duplicates):
            handler_name = '%shdl%d' % (pkg, idx)
            handler = Handler(handler_name, imps, deps, mem)
            write_handler(handlers_dir, handler)
        pkgs |= deps
        count += 1

    return pkgs


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
    parser.add_argument('-n', '--num-pkgs', type=int, default=-1, help='number of packages to used, sorted by level')
    args = parser.parse_args()

    with open(args.spec_file) as spec_file:
        spec = json.load(spec_file)
    graph = {entry['name']: set(entry['deps']) for entry in spec}
    pop = {entry['name']: entry['handler_popularity'] for entry in spec}
    refs ={entry['name']: 0  for entry in spec}

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
    pkgs = write_handlers(handlers_dir, graph, pop, refs, args.duplicates, args.num_pkgs)
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
