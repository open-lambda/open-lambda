import os
import numpy
import numpy
import json
from helper_modules.distribution import distribution_factory
from helper_modules.handler import Handler
from helper_modules.package import Package
import argparse

def write_lambda_func(handlers_dir, handler_name, packages):
    handler_contents = ''
    for p in packages:
        handler_contents += 'import ' + p.get_name() + '\n'
    handler_contents += str.format('''
def handler(conn, event):
    try:
        return "Hello from {0}"
    except Exception as e:
        return {{'error': str(e)}}
''', handler_name)
    os.makedirs('%s/%s' % (handlers_dir, handler_name))
    f = open('%s/%s/lambda_func.py' % (handlers_dir, handler_name), 'w')
    f.write(handler_contents)
    f.close()


def write_packages_txt(handler_dir, handler_name, packages):
    handler_contents = ''
    deps_list = get_all_dependencies_in_tree(packages)
    for d in deps_list:
        handler_contents += '%s:%s\n' % (d.get_name(), d.get_name())
    f = open('%s/%s/packages.txt' % (handler_dir, handler_name), 'w')
    f.write(handler_contents)
    f.close()


def get_total_popularity(packages):
    total_popularity = 0
    for p in packages:
        total_popularity += p.get_popularity()
    return total_popularity


def match_packages_and_handlers(handlers, packages):
    num_packages = len(packages)
    total_popularity = get_total_popularity(packages)

    for h in handlers:
        tries = 0
        while h.should_add_more_dependencies():
            tries += 1
            # find a package
            i = numpy.random.randint(0, num_packages)
            dep = packages[i]
            # add with probability proportional to the popularity
            if dep.get_popularity() / total_popularity:
                h.add_dependency(dep)
                dep.add_reference()


def generate_handlers(config):
    handlers = []
    for i in range(0, config['num_handlers']):
        num_deps = config["num_dependencies"].sample()
        name = 'a%d' % i
        handlers.append(Handler(name, num_deps))
    return handlers


def write_handlers(handlers_dir, handlers):
    for h in handlers:
        write_handler(handlers_dir, h)


def write_handler(handlers_dir, handler):
    write_lambda_func(handlers_dir, handler.get_name(), handler.get_dependencies())
    write_packages_txt(handlers_dir, handler.get_name(), handler.get_dependencies())


def parse_config(config_file_name):
    if config_file_name is None:
        config = {
            "num_handlers": 4000,
            "load": {
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
            "package_popularity": {
                "dist": "zipf",
                "a": 2
            },
            "num_dependencies": {
                "dist": "normal",
                "loc": 2.0,
                "scale": 2.0
            }
        }
    else:
        f = open(config_file_name, 'r')
        config = json.load(f)

    config['load']['cpu'] = distribution_factory(config['load']['cpu'])
    config['load']['mem'] = distribution_factory(config['load']['mem'])
    config['package_popularity'] = distribution_factory(config['package_popularity'])
    config['num_dependencies'] = distribution_factory(config['num_dependencies'])

    return config


def add_popularity_to_packages(csv_file_name, packages):
    with open(csv_file_name, 'r') as f:
        for line in f:
            line_l = line.split(',')
            # name,popularity
            name = line_l[0]
            popularity = int(line_l[1])
            for p in packages:
                if p.get_name() == name:
                    p.set_popularity(popularity)
                    break
    return packages


def get_all_dependencies_in_tree(packages):
    dep_list = []
    for p in packages:
        get_all_depdencies_rec_helper(p, dep_list)
    return dep_list


def get_all_depdencies_rec_helper(package, dep_list):
    dep_list.append(package)
    for d in package.get_dependencies():
        get_all_depdencies_rec_helper(d, dep_list)


def add_dependencies_to_packages(deps_json_file_name, packages):
    with open(deps_json_file_name, 'r') as f:
        package_dependencies = json.load(f)
    for name, deps in package_dependencies.items():
        for dep_name in deps:
            for p in packages:
                if p.get_name() == name:
                    for d in packages:
                        if d.get_name() == dep_name:
                            p.add_dependency(d)
                            break
                    break

def get_packages():
    with open('package_dependencies.json', 'r') as f:
        package_dependencies = json.load(f)
    packages = []
    for name in package_dependencies:
        packages.append(Package(name))
    return packages


def write_handler_import_distribution(packages):
    frequencies = ''
    for p in packages:
        frequencies += '%s,%d\n' % (p.get_name(), p.get_reference_count())

    f = open('handler_import_distribution.csv', 'w')
    f.write(frequencies)
    f.close()


def main():
    handlers_dir = 'handlers'
    packages_dir = 'packages'

    os.system('gcc -fPIC -shared -I/usr/include/python2.7 -lpython2.7  load_simulator.c -o load_simulator.so')

    if not os.path.exists(handlers_dir):
        os.makedirs(handlers_dir)

    if not os.path.exists(packages_dir):
        print('packages directory does not exist')
        exit()

    parser = argparse.ArgumentParser(description='Start a cluster')
    parser.add_argument('-config', default=None)
    parser.add_argument('-package-popularity-csv', default='package_popularity_real.csv')
    parser.add_argument('-package-dependencies-json', default='package_dependencies.json')
    args = parser.parse_args()

    config = parse_config(args.config)
    print('Reading in package popularity distribution...')
    packages = get_packages()
    add_dependencies_to_packages(args.package_dependencies_json, packages)
    add_popularity_to_packages(args.package_popularity_csv, packages)
    print('Creating handlers...')
    handlers = generate_handlers(config)
    print('Adding dependencies to handlers...')
    match_packages_and_handlers(handlers, packages)
    print('Writing out handlers...')
    write_handlers(handlers_dir, handlers)
    print('Writing out handler import distribution...')
    write_handler_import_distribution(packages)
    print('Done')


if __name__ == '__main__':
    main()
