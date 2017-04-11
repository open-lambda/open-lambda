import os
import random
import string
import tarfile
import shutil
import numpy
import math

def get_load_simulation_code_setup(cpu, mem):
    return str.format('''
import load_simulator


load_simulator.simulate_install({0}, {1})
''',  cpu, mem)

def get_load_simulation_code_init(name, cpu, mem):
    return str.format('''
import {2}.load_simulator


p = load_simulator.simulate_import({0}, {1})
''',  cpu, mem, name)

def copy_load_simulator_so(name):
    # the first is used in setup.py
    shutil.copyfile('load_simulator.so', packages_dir + '/' + name + '/load_simulator.so')
    # the second is use in __init__.py
    shutil.copyfile('load_simulator.so', packages_dir + '/' + name + '/' + name + '/load_simulator.so')

def create_data_files(package_name, file_sizes):
    dir = packages_dir + '/' + package_name + '/' + package_name + '/data/'
    os.makedirs(dir)
    for i in range(0, len(file_sizes)):
        f = open(dir + 'data_' + str(i) + '.dat', 'w')
        for j in range(0, file_sizes[i] * 1024):
            f.write(random.choice(string.ascii_letters + string.digits))
        f.close()

def create_setup(package_name, cpu, mem):
    dir = packages_dir + '/' + package_name + '/'
    # currently data files are in install directory, but not imported
    load_simulation = get_load_simulation_code_setup(cpu, mem)
    setup = str.format('''
from setuptools import setup

setup(
    name = '{0}',
    version = '0.1',
    packages=['{0}'],
    package_dir={{'{0}': '{0}'}},
    package_data={{'{0}': ['load_simulator.so', 'data/*.dat']}}
)
''', package_name)
    f = open(dir + 'setup.py', 'w')
    f.write(load_simulation + setup)
    f.close()

def create_init(package_name, cpu, mem):
    dir = packages_dir + '/' + package_name + '/' + package_name + '/'
    setup_contents = get_load_simulation_code_init(package_name, cpu, mem)
    f = open(dir + '__init__.py', 'w')
    f.write(setup_contents)
    f.close()

def get_package_name():
    # ensure no conflicts
    while True:
        name =  ''.join(random.choice(string.ascii_lowercase) for _ in range(10))
        if not os.path.exists('packages/' + name):
            return name

def generate_packages(distributions):
    packages = []
    package_spec = {}
    num_imports = distributions['num_packages']
    for i in range(0, num_imports):
        print('Creating package ', i + 1)
        num_files = distributions['data_files']['num'].sample()
        package_spec["data_files"] = []
        for j in range(0, num_files):
            package_spec["data_files"].append(distributions['data_files']['size'].sample()) # in KB
        package_spec["install_cpu"] = distributions['install']['cpu'].sample()
        package_spec["install_mem"] = distributions['install']['mem'].sample()
        package_spec["import_cpu"] = distributions['import']['cpu'].sample()
        package_spec["import_mem"] = distributions['import']['mem'].sample()

        new_package_name = create_package(package_spec)
        packages.append(new_package_name)
    return packages

def create_package(package_spec):
    name = get_package_name()
    os.makedirs(packages_dir + '/' + name)
    os.makedirs(packages_dir + '/' + name + '/' + name)
    create_data_files(name, package_spec['data_files'])
    create_setup(name, package_spec['install_cpu'], package_spec['install_mem'])
    create_init(name, package_spec['import_cpu'], package_spec['import_mem'])
    copy_load_simulator_so(name)
    tar = tarfile.open(packages_dir + '/' + name + "-0.1.tar.gz", "w:gz")
    os.chdir(packages_dir)
    tar.add( name)
    tar.close()
    shutil.rmtree(name)
    os.chdir('..')

class Distribution:
    def __init__(self, dist, dist_args):
        self.dist = dist
        self.dist_args = dist_args

    def sample(self):
        dist = getattr(numpy.random, self.dist)
        return abs(math.ceil(dist(**self.dist_args)))

def distribution_factory(dist_spec):
    dist = dist_spec['dist']
    dist_spec.pop('dist')
    return Distribution(dist, dist_spec)

def parse_config(config_file_name):
    if config_file_name is None:
        return {
            "num_packages": 1000,
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
        return config

def create_distributions(config):
    distributions = {}

    distributions['num_packages'] = config['num_packages']
    distributions['data_files'] = {}
    distributions['data_files']['num'] = distribution_factory(config['data_files']['num'])
    distributions['data_files']['size'] = distribution_factory(config['data_files']['size'])
    distributions['install'] = {}
    distributions['install']['cpu'] = distribution_factory(config['install']['cpu'])
    distributions['install']['mem'] = distribution_factory(config['install']['mem'])
    distributions['import'] = {}
    distributions['import']['cpu'] = distribution_factory(config['import']['cpu'])
    distributions['import']['mem'] = distribution_factory(config['import']['mem'])

    return distributions

if __name__ == '__main__':
    packages_dir = 'packages'

    config = parse_config(None)
    distributions = create_distributions(config)
    generate_packages(distributions)

    # create mirror dir if not found
    if not os.path.exists(packages_dir):
        os.makedirs(packages_dir)

