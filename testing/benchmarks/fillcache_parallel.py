#!/usr/bin/python
import sys
from subprocess import check_output
import multiprocessing

INDEX_HOST = '128.104.222.169'
INDEX_PORT = '9199'

CORES = 40
PKG_DIR = '/ol/open-lambda/pipbench/packages'
LIMIT = 10 * (1024**3) # 10 GB

def worker(packages):
    for pkg in packages:
        path = PKG_DIR + '/' + pkg + '-0.1.tar.gz'
        print path
        check_output(['pip', 'install', '-t', 'packages/%s' % pkg, '--no-deps', '-q', path])
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

