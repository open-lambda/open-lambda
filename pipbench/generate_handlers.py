import os
import shutil
import json
from helper_modules.handler import Handler
import argparse
import re


CWD = os.path.dirname(os.path.realpath(__file__))


def populate_graph(graph):
    # sort by level
    pkgs = sorted(graph, key=lambda x: int(re.match(r'level(\d+)', x).group(1)))
    for pkg in pkgs:
        if len(graph[pkg]) == 0:
            continue
        graph[pkg] |= set.union(*[graph[dep] for dep in graph[pkg]])


def write_handlers(handlers_dir, graph, duplicates):
    for pkg in graph:
        for idx in range(duplicates):
            handler_name = '%shdl%d' % (pkg, idx)
            # TODO: memory is hard-coded to 1MB
            mem = 1024
            # TODO: for now a handler only imports one package
            imps = set([pkg])
            handler = Handler(handler_name, imps, set.union(imps, *[graph[imp] for imp in imps]), mem)
            write_handler(handlers_dir, handler)


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
    args = parser.parse_args()

    with open(args.spec_file) as spec_file:
        spec = json.load(spec_file)
    graph = {entry['name']: set(entry['deps']) for entry in spec}

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
    write_handlers(handlers_dir, graph, args.duplicates)
    print('Done')


if __name__ == '__main__':
    main()
