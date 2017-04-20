import sys
import os
import random
import string
import tarfile
import shutil
import numpy
from helper_modules.distribution import distribution_factory
import json
from helper_modules.package import Package
import argparse
from subprocess import check_output
import multiprocessing


CORES = 40
RANDOM = None
CWD = os.path.dirname(os.path.realpath(__file__))


def worker(packages):
    for p in packages:
        write_package('%s/web' % CWD, p)


def write_packages(pkgs_dir, packages):
    pkg_shards = [[] for i in range(CORES)]
    for i in range(len(packages)):
        pkg_shards[i%CORES].append(packages[i])
    pool = multiprocessing.Pool(CORES)
    pool.map(worker, pkg_shards)


def create_tar(pkg_dir, pkg):
    data_dir = '%s/%s/data' % (pkg_dir, pkg.name)
    os.makedirs(data_dir)
    if pkg.uncompressed > 3300:
        with open('%s/data.dat' % data_dir, 'w') as data:
            random_char = chr(random.randint(0, 255))
            data.write(random_char * (pkg.uncompressed - 3300))
    with tarfile.open(pkg_dir + '-0.1.tar.gz', 'w:gz') as tar:
        tar.add(pkg_dir, arcname=pkg.name)
    size = os.stat(pkg_dir + '-0.1.tar.gz').st_size
    if size < pkg.compressed:
        with open('%s/ballast.dat' % pkg_dir, 'wb') as ballast:
            rand_len = pkg.compressed - size
            ballast.write(RANDOM[:rand_len])
    with tarfile.open(pkg_dir + '-0.1.tar.gz', 'w:gz') as tar:
        tar.add(pkg_dir, arcname=pkg.name)


def write_pkg_simple(simple_dir, pkg):
    os.makedirs('%s/%s' % (simple_dir, pkg.name))
    with open('%s/%s/index.html' % (simple_dir, pkg.name), 'w') as index:
        index.write('''
<body><a href="../../packages/{pkg_dir}-0.1.tar.gz">{pkg_tar}</a></body>
'''.format(pkg_dir=pkg.get_dir(), pkg_tar=pkg.name + '-0.1.tar.gz'))


def write_package(mirror_dir, pkg):
    pkg_dir = '%s/packages/%s' % (mirror_dir, pkg.get_dir())
    # create package directories
    os.makedirs('%s/%s' % (pkg_dir, pkg.name))
    # create contents
    with open('%s/setup.py' % pkg_dir, 'w') as f:
        f.write(pkg.setup_code())
    with open('%s/%s/__init__.py' % (pkg_dir, pkg.name), 'w') as f:
        f.write(pkg.init_code())
    shutil.copyfile('load_simulator.so', '%s/load_simulator.so' % (pkg_dir))
    shutil.copyfile('load_simulator.so', '%s/%s/load_simulator.so' % (pkg_dir, pkg.name))
    create_tar(pkg_dir, pkg)
    shutil.rmtree(pkg_dir)

    write_pkg_simple('%s/simple' % mirror_dir, pkg)


def write_simple(mirror_dir, pkgs):
    with open('%s/simple/index.html' % mirror_dir, 'w') as index:
        index.write('<body>')
        for pkg in pkgs:
            index.write('<a href="{pkg_name}">{pkg_name}</a>'.format(pkg_name=pkg.name))
        index.write('</body>')


def main():
    if len(sys.argv) != 3:
        print('usage: python %s <spec_file> <random_file>')
        return

    with open(sys.argv[1]) as spec_file:
        spec = json.load(spec_file)

    max_size = max(pkg['compressed'] for pkg in spec)

    global RANDOM
    if len(sys.argv) >= 3:
        with open(sys.argv[2], 'rb') as f:
            RANDOM = f.read()
        if len(RANDOM) < max_size:
            print('random file is not large enough: %d < %d' % (len(RANDOM), max_size))
            return

    mirror_dir = '%s/web' % CWD

    # create mirror dir if not found
    if not os.path.exists(mirror_dir):
        os.makedirs(mirror_dir)

    # ensure we have the load simulator binary
    os.system('gcc -fPIC -shared -I/usr/include/python2.7 -lpython2.7  load_simulator.c -o load_simulator.so')

    #parser = argparse.ArgumentParser(description='Create pipbench packages')
    #parser.add_argument('-config', default=None)
    #args = parser.parse_args()

    #config = parse_config(args.config)
    print('Creating packages...')
    #packages = generate_packages(spec)
    packages = [Package(**entry) for entry in spec]
    #print('Generating dependency tree...')
    #create_dependency_tree(packages)
    print('Writing out packages....')
    write_packages(mirror_dir, packages)
    #print('Writing out popularity distribution target...')
    #write_popularity_distribution_target(packages)
    #print('Writing out popularity distribution real...')
    #write_popularity_distribution_real(packages)
    #print('Writing out direct dependencies lists...')
    #write_out_package_dependencies(packages)
    #print('Writing out package size file...')
    #write_out_package_sizes(packages)
    print('Writing out index.html')
    write_simple(mirror_dir, packages)
    print('Done')


if __name__ == '__main__':
    main()
