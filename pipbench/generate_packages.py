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

def create_setup(package_name, cpu, mem, deps):
    dir = packages_dir + '/' + package_name + '/'
    # currently data files are in install directory, but not imported
    load_simulation = get_load_simulation_code_setup(cpu, mem)
    deps_str = ''
    for d in deps:
        deps_str += "        \'" + d + "\',\n"
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

def create_init(package_name, cpu, mem, deps):
    dir = packages_dir + '/' + package_name + '/' + package_name + '/'
    imports_str = ''
    for d in deps:
        imports_str += 'import %s\n' % d
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

def add_dependencies(config, packages):
    package_imports = {}
    package_imports_nums = {}
    for p in packages:
        package_imports[p] = []
        package_imports_nums[p] = config['package_popularity'].sample()

    most_used_pkgs_first = sorted(package_imports_nums, key=package_imports_nums.get, reverse=True)

    for i in range(0, len(packages)):
        num = package_imports_nums[most_used_pkgs_first[i]]
        print('num ' + str(num))
        used_by = 0
        while used_by < num and used_by < len(packages) - i - 1:
            pi = numpy.random.randint(i + 1, len(packages))
            if most_used_pkgs_first[i] not in package_imports[most_used_pkgs_first[pi]]:
                package_imports[most_used_pkgs_first[pi]].append(most_used_pkgs_first[i])
                used_by += 1
                print('added to ' + most_used_pkgs_first[pi])

    return package_imports

def generate_packages(config):
    packages = []
    num_packages = config['num_packages']
    for i in range(0, num_packages):
        packages.append(get_package_name())
    package_deps = add_dependencies(config, packages)

    k = 0
    for p in packages:
        k += 1
        print('Creating package %d: %s' % (k, p))
        package_spec = {}
        num_files = config['data_files']['num'].sample()
        package_spec["data_files"] = []
        for j in range(0, num_files):
            package_spec["data_files"].append(config['data_files']['size'].sample()) # in KB
        package_spec["install_cpu"] = config['install']['cpu'].sample()
        package_spec["install_mem"] = config['install']['mem'].sample()
        package_spec["import_cpu"] = config['import']['cpu'].sample()
        package_spec["import_mem"] = config['import']['mem'].sample()
        package_spec["dependencies"] = package_deps[p]
        create_package(package_spec, p)
    return packages

def create_package(package_spec, name):
    os.makedirs(packages_dir + '/' + name)
    os.makedirs(packages_dir + '/' + name + '/' + name)
    create_data_files(name, package_spec['data_files'])
    create_setup(name, package_spec['install_cpu'], package_spec['install_mem'], package_spec['dependencies'])
    create_init(name, package_spec['import_cpu'], package_spec['import_mem'], package_spec['dependencies'])
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
            "package_popularity": {
                "dist": "zipf",
                "a": 2
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

def parse_distributions(config):
    config['package_popularity'] = distribution_factory(config['package_popularity'])
    config['data_files']['num'] = distribution_factory(config['data_files']['num'])
    config['data_files']['size'] = distribution_factory(config['data_files']['size'])
    config['install']['cpu'] = distribution_factory(config['install']['cpu'])
    config['install']['mem'] = distribution_factory(config['install']['mem'])
    config['import']['cpu'] = distribution_factory(config['import']['cpu'])
    config['import']['mem'] = distribution_factory(config['import']['mem'])
    return config

if __name__ == '__main__':
    os.system('gcc -shared -I/usr/include/python2.7 -lpython2.7  load_simulator.c -o load_simulator.so')

    packages_dir = 'packages'

    config = parse_config(None)
    config = parse_distributions(config)
    generate_packages(config)

    # create mirror dir if not found
    if not os.path.exists(packages_dir):
        os.makedirs(packages_dir)

