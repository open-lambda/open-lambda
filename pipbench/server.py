import falcon
import json
import os
import random
import string


def create_assets(package_name, num_files, file_size):
    dir = packages_dir + '/' + package_name + '/'
    for i in range(0, num_files):
        f = open(dir + 'asset_' + str(i), 'w')
        for j in range(0, file_size * 1000):
            f.write("\0")
        f.close()


def create_setup(package_name, cpu, mem):
    dir = packages_dir + '/' + package_name + '/'
    # todo use c code
    setup_contents = ''
    for i in range(0, cpu):
        setup_contents += '''
if True == True:
    pass
'''
    f = open(dir + 'setup.py', 'w')
    f.write(setup_contents)
    f.close()

def create_init(package_name, cpu, mem):
    dir = packages_dir + '/' + package_name + '/'
    # todo use c code
    setup_contents = ''
    for i in range(0, cpu):
        setup_contents += '''
if True == True:
    pass
'''
    f = open(dir + '__init__.py', 'w')
    f.write(setup_contents)
    f.close()

def does_package_exist(name):
    return os.path.exists(packages_dir + '/' + name)

def get_package_name(package_spec):
    return str(package_spec['numAssets']) + 'x' + str(package_spec['assetSize']) + 'x' + str(package_spec['installCpu']) + 'x' \
           + str(package_spec['installMem']) + 'x' + str(package_spec['importCpu']) + 'x' + str(package_spec['importMem'])

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
        print(type(package_spec))
        name = get_package_name(package_spec)
        if does_package_exist(name):
            print('package already exists with name ' + name)
            res.body = json.dumps({'packageName': name})
            res.status = falcon.HTTP_200
            return
        # create package since it doesn't exist
        print('creating package with name ' + name)
        try:
            os.makedirs(packages_dir + '/' + name)
            create_assets(name, package_spec['numAssets'], package_spec['assetSize'])
            create_setup(name, package_spec['importCpu'], package_spec['importMem'])
            create_init(name, package_spec['installCpu'], package_spec['installMem'])
        except Exception as e:
            print(e)
            res.status = falcon.HTTP_500
            return
        res.body = json.dumps({'packageName': name})
        res.status = falcon.HTTP_200

packages_dir = 'mirror_dir'

# create mirror dir if not found
if not os.path.exists(packages_dir):
    os.makedirs(packages_dir)

# setup server endpoint
package_resource = PackageResource()
app = falcon.API()
app.add_route('/package', package_resource)