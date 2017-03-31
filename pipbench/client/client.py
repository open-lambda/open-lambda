import argparse
import requests
import json
import os
import tarfile
import shutil
import numpy
import math

'''
PipBench client
Triggers the backend to create a package with the provided parameters, which is added to a local
PyPi compatible cache, and retrieves the name of the created package. It also generates a handler
that uses the packages, and uploads it to the store. 
'''

def generate_packages(distributions):
    packages = []
    num_imports = distributions['numImports'].sample()
    for i in range(0, num_imports):
        num_files = distributions['dataFiles']['num'].sample()
        data_files = []
        for j in range(0, num_files):
            data_files.append(distributions['dataFiles']['size'].sample()) # in KB
        install_cpu = distributions['install']['cpu'].sample()
        install_mem = distributions['install']['mem'].sample()
        import_cpu = distributions['import']['cpu'].sample()
        import_mem = distributions['import']['mem'].sample()

        new_package_name = generate_package(data_files, install_cpu, install_mem, import_cpu, import_mem)
        packages.append(new_package_name)
    return packages

def generate_package(data_files, install_cpu, install_mem, import_cpu, import_mem):
    headers = {'Content-Type': 'application/json'}
    r = requests.post('http://localhost:9198/package', headers=headers, data=json.dumps({
        "dataFiles": data_files,
        "installCpu": install_cpu,
        "installMem": install_mem,
        "importCpu": import_cpu,
        "importMem": import_mem
    }))
    if r.status_code != 200:
        raise Exception('PipBench server failed to create package ' + r.json())
    else:
        package_name = r.json()['packageName']
        return package_name

def generate_handler_file(handler_name, packages):
    handler_contents = ''
    for p in packages:
        handler_contents += 'import ' + p + '\n'
    f = open(handler_name + '/lambda_func.py', 'w')
    f.write(handler_contents)
    f.close()

def generate_requirements(handler_name, packages):
    handler_contents = ''
    for p in packages:
        handler_contents += p + '\n'
    f = open(handler_name + '/' + 'requirements.txt', 'w')
    f.write(handler_contents)
    f.close()

def zip(handler_name):
    tar = tarfile.open(handler_name + ".tar.gz", "w:gz")
    tar.add(handler_name)
    tar.close()

def cleanup(handler_name):
    shutil.rmtree(handler_name)

def parse_config(config_file_name):
    if config_file_name is None:
        return {
            "numImports": {
                "dist": "normal",
                "loc": 5.0,
                "scale": 2.0
            },
            "dataFiles": {
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
                    "loc": 1000000.0,
                    "scale": 10000.0
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
                    "loc": 1000000.0,
                    "scale": 10000.0
                }
            }
        }
    else:
        f = open(config_file_name, 'r')
        config = json.load(f)
        return config

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

def create_distributions(config):
    distributions = {}

    distributions['numImports'] = distribution_factory(config['numImports'])
    distributions['dataFiles'] = {}
    distributions['dataFiles']['num'] = distribution_factory(config['dataFiles']['num'])
    distributions['dataFiles']['size'] = distribution_factory(config['dataFiles']['size'])
    distributions['install'] = {}
    distributions['install']['cpu'] = distribution_factory(config['install']['cpu'])
    distributions['install']['mem'] = distribution_factory(config['install']['mem'])
    distributions['import'] = {}
    distributions['import']['cpu'] = distribution_factory(config['import']['cpu'])
    distributions['import']['mem'] = distribution_factory(config['import']['mem'])

    return distributions

parser = argparse.ArgumentParser(description='Generate a handler')
parser.add_argument('-config', default=None)
parser.add_argument('-handler-name', default='pip_bench_handler')
args = parser.parse_args()

if not os.path.exists(args.handler_name):
    os.makedirs(args.handler_name)

config = parse_config(args.config)
distributions = create_distributions(config)
packages = generate_packages(distributions)
generate_handler_file(args.handler_name, packages)
generate_requirements(args.handler_name, packages)
zip(args.handler_name)
cleanup(args.handler_name)
# todo upload handler to store if flag present
print('Created handler zip ' + args.handler_name + ' successfully')

