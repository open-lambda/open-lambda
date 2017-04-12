import os
import numpy
import numpy
import json
from helper_modules.distribution import distribution_factory
from helper_modules.handler import Handler
from helper_modules.package import Package


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
    for p in packages:
        handler_contents += '%s:%s\n' % (p.get_name(), p.get_name())
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
        while h.should_add_more_dependencies() and tries < 10 * num_packages:
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


def get_list_of_packages():
    packages = []
    with open('package_popularity_real.csv', 'r') as f:
        for line in f:
            line_l = line.split(',')
            # name,popularity
            name = line_l[0]
            popularity = int(line_l[1])
            packages.append(Package(name, popularity))
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

    config = parse_config(None)
    print('Reading in package popularity distribution...')
    packages = get_list_of_packages()
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
