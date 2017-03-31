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
    shutil.copyfile('load_simulator.so', packages_dir + '/' + name + '/load_simulator.so')
    shutil.copyfile('load_simulator.so', packages_dir + '/' + name + '/' + name + '/load_simulator.so')


def build_load_simulator():
    os.system('gcc -shared -I/usr/include/python2.7 -lpython2.7  load_simulator.c -o load_simulator.so')

def create_assets(package_name, num_files, file_size):
    dir = packages_dir + '/' + package_name + '/assets/'
    os.makedirs(dir)
    for i in range(0, num_files):
        f = open(dir + 'asset_' + str(i), 'w')
        for j in range(0, file_size * 1024):
            f.write("\0")
        f.close()

def create_setup(package_name, cpu, mem):
    dir = packages_dir + '/' + package_name + '/'
    # todo add data files to setup? looks like numpy doesn't use this in setup
    load_simulation = get_load_simulation_code_setup(cpu, mem)
    setup = str.format('''
from setuptools import setup

setup(
    name = '{0}',
    version = '0.1',
    packages=['{0}'],
    package_dir={{'{0}': '{0}'}},
    py_modules= ['{0}.load_simulator']
)
''', package_name)
    f = open(dir + 'setup.py', 'w')
    f.write(load_simulation + setup)
    f.close()

def create_init(package_name, cpu, mem):
    dir = packages_dir + '/' + package_name + '/' + package_name + '/'
    os.makedirs(dir)
    setup_contents = get_load_simulation_code_init(package_name, cpu, mem)
    f = open(dir + '__init__.py', 'w')
    f.write(setup_contents)
    f.close()

def does_package_exist(name):
    return os.path.exists(packages_dir + '/' + name + '-0.1.tar.gz')

def get_package_name(package_spec):
    return ''.join(random.choice(string.ascii_uppercase) for _ in range(10))

class MockedPackageResource:
    def on_post(self, req, res):
        print('hit')
        res.body = json.dumps({'packageName': ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(10)) })


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
        name = get_package_name(package_spec)
        if does_package_exist(name):
            print('package already exists with name ' + name)
            res.body = json.dumps({'packageName': name})
            res.status = falcon.HTTP_200
            return
        # create package
        print('creating package with name ' + name)
        try:
            os.makedirs(packages_dir + '/' + name)
            create_assets(name, package_spec['numAssets'], package_spec['assetSize'])
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