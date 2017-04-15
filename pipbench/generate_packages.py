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

def get_load_simulation_code_setup(cpu, mem):
    return str.format('''
import load_simulator
load_simulator.simulate_install({0}, {1}, False)
''',  cpu, 0)


def get_load_simulation_code_init(name, cpu, mem):
    return str.format('''
import {2}.load_simulator
p = load_simulator.simulate_load({0}, {1}, True)
''',  cpu, mem, name)


def copy_load_simulator_so(packages_dir, package_name):
    # the first is used in setup.py
    shutil.copyfile('load_simulator.so', packages_dir + '/' + package_name + '/load_simulator.so')
    # the second is use in __init__.py
    shutil.copyfile('load_simulator.so', packages_dir + '/' + package_name + '/' + package_name + '/load_simulator.so')


def create_data_files(packages_dir, package_name, file_sizes, compression_ratio_int):
    dir = packages_dir + '/' + package_name + '/' + package_name + '/data/'
    os.makedirs(dir)
    compressable_size = 0
    for i in range(0, len(file_sizes)):
        compressable_size += file_sizes[i]
        f = open(dir + 'data_' + str(i) + '.dat', 'w')
        random_char = random.choice(string.ascii_letters + string.digits)
        compressable_str = ''
        for j in range(0, file_sizes[i] * 1024):
            compressable_str += random_char
        f.write(compressable_str)
        f.close()
    compression_ratio = float(compression_ratio_int) / float(100)
    if compressable_size > 0 and compression_ratio > 0:
        uncompressable_size = int(((1 - compression_ratio) * compressable_size) / compression_ratio)
        uncompressable_str = ''
        for j in range(0, uncompressable_size * 1024):
            uncompressable_str += random.choice(string.ascii_letters + string.digits)
        f = open(dir + 'uncompressable.dat', 'w')
        f.write(uncompressable_str)
        f.close()


def create_setup(packages_dir, package_name, cpu, mem, deps):
    dir = packages_dir + '/' + package_name + '/'
    # currently data files are in install directory, but not imported
    load_simulation = get_load_simulation_code_setup(cpu, mem)
    deps_str = ''
    for d in deps:
        deps_str += "        \'" + d.get_name() + "\',\n"
    setup = str.format('''
from setuptools import setup
setup(
    name = '{0}',
    version = '0.1',
    packages=['{0}'],
    package_dir={{'{0}': '{0}'}},
    package_data={{'{0}': ['load_simulator.so', 'data/*.dat']}},
    install_requires=[
{1}
    ],
)
''', package_name, deps_str)
    f = open(dir + 'setup.py', 'w')
    f.write(load_simulation + setup)
    f.close()


def create_init(packages_dir, package_name, cpu, mem, deps):
    dir = packages_dir + '/' + package_name + '/' + package_name + '/'
    imports_str = ''
    for d in deps:
        imports_str += 'import %s\n' % d.get_name()
    setup_contents = get_load_simulation_code_init(package_name, cpu, mem)
    f = open(dir + '__init__.py', 'w')
    f.write(imports_str + setup_contents)
    f.close()


def get_package_name():
    # ensure no conflicts
    while True:
        name =  ''.join(random.choice(string.ascii_lowercase) for _ in range(10))
        if not os.path.exists('packages/' + name):
            return name


def dep_safety_rec(package, potential_dependency):
    if package == potential_dependency:
        return False
    existing_deps = package.get_dependencies()
    for dep in existing_deps:
        if not is_safe_to_add_dependency(dep, potential_dependency):
            return False
    return True


def is_safe_to_add_dependency(package, potential_dependency):
    try:
        return dep_safety_rec(package, potential_dependency)
    except RecursionError:
        return False


def get_total_popularity(packages):
    total_popularity = 0
    for p in packages:
        total_popularity += p.get_popularity()
    return total_popularity


def create_dependency_tree(packages):
    num_packages = len(packages)
    total_popularity = get_total_popularity(packages)

    for p in packages:
        tries = 0
        while p.should_add_more_dependencies() and tries < 10 * num_packages:
            tries += 1
            # get a dependency to try
            i = numpy.random.randint(0, num_packages)
            dep = packages[i]
            # add with probability proportional to the popularity
            if dep.get_popularity() / total_popularity > numpy.random.random() and is_safe_to_add_dependency(p, dep):
                p.add_dependency(dep)
                dep.add_reference()
        #print('real: %s target: %s' % (len(p.get_dependencies()), p.get_dependencies_target()))


def generate_packages(config):
    packages = []
    num_packages = config['num_packages']
    for i in range(0, num_packages):
        name = get_package_name()
        num_files = config['data_files']['num'].sample()
        data_file_sizes = []
        for j in range(0, num_files):
            data_file_sizes.append(config['data_files']['size'].sample()) # in KB
        compression_ratio = config['data_files']['compression_ratio'].sample()
        install_cpu_time = config['install']['cpu'].sample()
        install_mem = config['install']['mem'].sample()
        import_cpu_time = config['import']['cpu'].sample()
        import_cpu_mem = config['import']['mem'].sample()
        num_dependencies = config['num_dependencies'].sample()
        popularity = config['popularity'].sample()
        new_package = Package(name, popularity, num_dependencies, data_file_sizes, compression_ratio, install_cpu_time, install_mem,
                              import_cpu_time, import_cpu_mem)
        packages.append(new_package)
    return packages


def write_packages(packages_dir, packages):
    for p in packages:
        write_package(packages_dir, p)


def write_package(packages_dir, package):
    # create package directories
    os.makedirs('%s/%s' % (packages_dir, package.get_name()))
    os.makedirs('%s/%s/%s' % (packages_dir, package.get_name(), package.get_name()))
    # create contents
    create_data_files(packages_dir, package.get_name(), package.get_data_file_sizes(), package.get_compression_ratio())
    create_setup(packages_dir, package.get_name(), package.get_install_cpu_time(), package.get_install_mem(),
                 package.get_dependencies())
    create_init(packages_dir, package.get_name(), package.get_import_cpu_time(), package.get_import_mem(),
                package.get_dependencies())
    copy_load_simulator_so(packages_dir, package.get_name())
    # tarball and zip
    tar = tarfile.open(packages_dir + '/' + package.get_name() + "-0.1.tar.gz", "w:gz")
    os.chdir(packages_dir)
    tar.add(package.get_name())
    tar.close()
    # remove unzipped package directory
    shutil.rmtree(package.get_name())
    os.chdir('..')


def parse_config(config_file_name):
    if config_file_name is None:
        config = {
            "num_packages": 1000,
            "popularity": {
                "dist": "zipf",
                "a": 2
            },
            "num_dependencies": {
                "dist": "normal",
                "loc": 0.0,
                "scale": 0.5
            },
            "data_files": {
                "num": {
                    "dist": "normal",
                    "loc": 10.0,
                    "scale": 5.0
                },
                "size": {
                    "dist": "normal",
                    "loc": 10.0,
                    "scale": 5.0
                },
                "compression_ratio": {
                    "dist": "exact_value",
                    "value": 50
                }
            },
            "install": {
                "cpu": {
                    "dist": "normal",
                    "loc": 100000000.0,
                    "scale": 100000000.0
                },
                "mem": {
                    "dist": "normal",
                    "loc": 10000.0,
                    "scale": 10.0
                }
            },
            "import": {
                "cpu": {
                    "dist": "normal",
                    "loc": 100000000.0,
                    "scale": 100000000.0
                },
                "mem": {
                    "dist": "normal",
                    "loc": 10000.0,
                    "scale": 10.0
                }
            }
        }
    else:
        f = open(config_file_name, 'r')
        config = json.load(f)

    # configure distributions
    config['popularity'] = distribution_factory(config['popularity'])
    config['num_dependencies'] = distribution_factory(config['num_dependencies'])
    config['data_files']['num'] = distribution_factory(config['data_files']['num'])
    config['data_files']['size'] = distribution_factory(config['data_files']['size'])
    config['data_files']['compression_ratio'] = distribution_factory(config['data_files']['compression_ratio'])
    config['install']['cpu'] = distribution_factory(config['install']['cpu'])
    config['install']['mem'] = distribution_factory(config['install']['mem'])
    config['import']['cpu'] = distribution_factory(config['import']['cpu'])
    config['import']['mem'] = distribution_factory(config['import']['mem'])
    return config


def write_popularity_distribution_target(packages):
    contents = ''
    for p in packages:
        contents += '%s,%d\n' % (p.get_name(), p.get_popularity())
    f = open('package_popularity_target.csv', 'w')
    f.write(contents)
    f.close()

def write_popularity_distribution_real(packages):
    contents = ''
    for p in packages:
        contents += '%s,%d\n' % (p.get_name(), p.get_reference_count())
    f = open('package_popularity_real.csv', 'w')
    f.write(contents)
    f.close()

def main():
    packages_dir = 'packages'

    # create mirror dir if not found
    if not os.path.exists(packages_dir):
        os.makedirs(packages_dir)

    # ensure we have the load simulator binary
    os.system('gcc -fPIC -shared -I/usr/include/python2.7 -lpython2.7  load_simulator.c -o load_simulator.so')


    parser = argparse.ArgumentParser(description='Start a cluster')
    parser.add_argument('-config', default=None)
    args = parser.parse_args()

    config = parse_config(args.config)
    print('Creating packages...')
    packages = generate_packages(config)
    print('Generating dependency tree...')
    create_dependency_tree(packages)
    print('Writing out packages....')
    write_packages(packages_dir, packages)
    print('Writing out popularity distribution target...')
    write_popularity_distribution_target(packages)
    print('Writing out popularity distribution real...')
    write_popularity_distribution_real(packages)
    print('Done')


if __name__ == '__main__':
    main()
