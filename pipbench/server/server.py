import falcon
import json
import os
import random
import string
import tarfile
import shutil


def get_load_simulation_code_setup(cpu, mem):
    return str.format('''
import load_simulator


load_simulator.simulate({0}, {1})
''',  cpu, mem)

def get_load_simulation_code_init(name, cpu, mem):
    return str.format('''
import {2}.load_simulator


load_simulator.simulate({0}, {1})
''',  cpu, mem, name)

def copy_load_simulator_so(name):
    # the first is used in setup.py
    shutil.copyfile('load_simulator.so', packages_dir + '/' + name + '/load_simulator.so')
    # the second is use in __init__.py
    shutil.copyfile('load_simulator.so', packages_dir + '/' + name + '/' + name + '/load_simulator.so')

def build_load_simulator():
    os.system('gcc -shared -I/usr/include/python2.7 -lpython2.7  load_simulator.c -o load_simulator.so')

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
    return ''.join(random.choice(string.ascii_lowercase) for _ in range(10))

class PackageResource:
    def on_post(self, req, res):
        print('hit')
        body = req.stream.read(req.content_length or 0)
        if body == 0:
            res.body = 'No body provided'
            res.status = falcon.HTTP_400
            return
        package_spec = json.loads(body.decode("utf-8"))
        print(package_spec)
        name = get_package_name()
        # create package
        print('creating package with name ' + name)
        try:
            os.makedirs(packages_dir + '/' + name)
            os.makedirs(packages_dir + '/' + name + '/' + name)
            create_data_files(name, package_spec['dataFiles'])
            create_setup(name, package_spec['importCpu'], package_spec['importMem'])
            create_init(name, package_spec['installCpu'], package_spec['installMem'])
            copy_load_simulator_so(name)
            tar = tarfile.open(packages_dir + '/' + name + "-0.1.tar.gz", "w:gz")
            os.chdir(packages_dir)
            tar.add( name)
            tar.close()
            shutil.rmtree(name)
            os.chdir('..')
            print('package created')
        except Exception as e:
            print(e)
            res.status = falcon.HTTP_500
            return
        res.body = json.dumps({'packageName': name})
        res.status = falcon.HTTP_200

packages_dir = 'packages'

# create mirror dir if not found
if not os.path.exists(packages_dir):
    os.makedirs(packages_dir)

# make sure load simulator shared library exists
build_load_simulator()

# setup server endpoint
package_resource = PackageResource()
app = falcon.API()
app.add_route('/package', package_resource)