#!/usr/bin/python
import sys
from subprocess import check_output
import multiprocessing

INDEX_HOST = 'node-1.sosp.openlambda-pg0.wisc.cloudlab.us'
INDEX_PORT = '9199'

CORES = 40
PKG_DIR = '/ol/open-lambda/pipbench/packages'
LIMIT = 10 * (1024**3) # 10 GB

def worker(packages):
    for pkg in packages:
        cmd = ['pip', 'install', '-i', 'http://' + INDEX_HOST + ':' + INDEX_PORT + '/simple', '--trusted-host', INDEX_HOST, '-t', 'packages/%s' % pkg, '--no-deps', '-q', pkg]
        print(' '.join(cmd))
        check_output(cmd)

    return len(packages)

def main():
    curr = 0.0

    # shard work
    shards = [[] for i in range(CORES)]
    with open('top_packages.txt', 'r') as fd:
        for i,line in enumerate(fd):
            pkg = line.strip()
            shards[i%CORES].append(pkg)

    # unpack in parallel
    pool = multiprocessing.Pool(CORES)
    print pool.map(worker, shards)

if __name__ == '__main__':
    main()

