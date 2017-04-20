import hashlib

class Package:
    def __init__(self,
            name,
            deps=None,
            popularity=None,
            compressed=None,
            uncompressed=None,
            install_cpu=None,
            import_cpu=None,
            import_mem=None,
            subfiles=None):
        self.name = name
        self.deps = deps
        self.popularity = popularity
        self.compressed = compressed
        self.uncompressed = uncompressed
        self.install_cpu = install_cpu
        self.import_cpu = import_cpu
        self.import_mem = import_mem


    def setup_code(self):
        deps = ','.join("'%s'" % dep for dep in self.deps)
        return '''
import load_simulator
load_simulator.simulate_load({cpu}, 0)

from setuptools import setup
setup(
    name = '{name}',
    version = '0.1',
    packages=['{name}'],
    package_dir={{'{name}': '{name}'}},
    package_data={{'{name}': ['load_simulator.so', 'data/*.dat']}},
    install_requires=[{deps}],
)
'''.format(name=self.name, cpu=self.install_cpu, deps=deps)


    def init_code(self):
        imps = '\n'.join('import %s' % dep for dep in self.deps)
        return '''
{imps}
import {name}.load_simulator
p = load_simulator.simulate_load({cpu}, {mem})
'''.format(name=self.name, imps=imps, cpu=self.import_cpu, mem=self.import_mem)


    def get_dir(self):
        hsh = hashlib.sha256(self.name.encode('utf-8')).hexdigest()
        return '%s/%s/%s/%s' % (hsh[:2], hsh[2:4], hsh[4:], self.name)
