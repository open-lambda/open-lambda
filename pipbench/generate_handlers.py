import os
import numpy
import math
import numpy

def write_lambda_func(handler_dir, handler_name, packages):
    handler_contents = ''
    for p in packages:
        handler_contents += 'import ' + p + '\n'
    handler_contents += str.format('''
def handler(conn, event):
    try:
        return "Hello from {0}"
    except Exception as e:
        return {{'error': str(e)}}
''', handler_name)
    os.makedirs('%s/%s' % (handlers_dir, handler_name))
    f = open('%s/%s/lambda_func.py' % (handler_dir, handler_name), 'w')
    f.write(handler_contents)
    f.close()

def write_packages_txt(handler_dir, handler_name, packages):
    handler_contents = ''
    for p in packages:
        handler_contents += '%s:%s\n' % (p, p)
    f = open('%s/%s/packages.txt' % (handler_dir, handler_name), 'w')
    f.write(handler_contents)
    f.close()

def get_next_package(total_counts, targets, packages):
    while True:
        i = numpy.random.random_integers(0, len(packages) - 1)
        p = packages[i]
        if total_counts[p] - targets[p] < 0:
            total_counts[p] += 1
            return p

def get_packages_for_handler(num_handlers, handler_num, total_counts, targets, target_total, packages):
    used_packages = []
    num_packages_to_use = math.floor(target_total / num_handlers)
    extra_num_needed = math.fmod(target_total, num_handlers)
    extra_indicator = math.floor(num_handlers / extra_num_needed)
    if math.fmod(handler_num, extra_indicator) == 0.0:
        num_packages_to_use += 1
    for i in range(0, num_packages_to_use):
        p = get_next_package(total_counts, targets, packages)
        used_packages.append(p)
    return used_packages

def match_packages_and_handlers(handlers_to_packages, total_counts, targets, target_total, all_handlers, all_packages):
    num_handlers = len(all_handlers)
    num_packages = len(all_packages)
    ordered_package_names = sorted(targets, key=targets.get)
    for i in range(0, target_total):
        p = None
        for package in ordered_package_names:
            if total_counts[package] < targets[package]:
                p = package
                break

        while True:
            j = numpy.random.random_integers(0, num_handlers - 1)
            handler = all_handlers[j]

            if total_counts[package] < targets[package] and package not in handlers_to_packages[handler]:
                handlers_to_packages[handler].append(package)
                total_counts[package] += 1
                break


def generate_handlers(config, handler_dir, total_counts, targets, target_total, all_packages):
    name = 'a'
    handlers_to_packages = {}
    handlers = []
    for i in range(0, config['num_handlers']):
        handlers_to_packages[name + str(i)] = []
        handlers.append(name + str(i))
    match_packages_and_handlers(handlers_to_packages, total_counts, targets, target_total, handlers, all_packages)
    for handler_name, packages in handlers_to_packages.items():
        write_lambda_func(handler_dir, handler_name, packages)
        write_packages_txt(handlers_dir, handler_name, packages)

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
    config['load']['cpu'] = distribution_factory(config['load']['cpu'])
    config['load']['mem'] = distribution_factory(config['load']['mem'])
    config['package_popularity'] = distribution_factory(config['package_popularity'])

    return config

def parse_config(config_file_name):
    if config_file_name is None:
        return {
            "num_handlers": 1000,
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
            }
        }
    else:
        f = open(config_file_name, 'r')
        config = json.load(f)
        return config

def get_list_of_packages(packages_dir, total_counts):
    all_packages_tars = os.listdir('%s' % packages_dir)
    all_packages_names = []
    for p in all_packages_tars:
        name = p.split('-')[0]
        total_counts[name] = 0
        all_packages_names.append(name)
    return all_packages_names

def create_target_counts(config, total_counts, targets):
    sum = 0
    for package_name in total_counts:
        target = config['package_popularity'].sample()
        sum += target
        targets[package_name] = target
    return sum

def write_actual_package_frequencies(total_counts):
    frequencies = ''
    for p in total_counts:
        frequencies += '%s,%d\n' % (p, total_counts[p])

    f = open('import_distribution.csv', 'w')
    f.write(frequencies)
    f.close()

if __name__ == '__main__':
    handlers_dir = 'handlers'
    packages_dir = 'packages'
    if not os.path.exists(handlers_dir):
        os.makedirs(handlers_dir)

    if not os.path.exists(packages_dir):
        print('packages directory does not exist')
        exit()

    config = parse_config(None)
    config = create_distributions(config)
    total_counts = {}
    targets = {}
    all_packages = get_list_of_packages(packages_dir, total_counts)
    target_total = create_target_counts(config, total_counts, targets)
    print('Distributing %d imports of %d packages across %d handlers' %(target_total, len(all_packages), 1000))
    generate_handlers(config, handlers_dir, total_counts, targets, target_total, all_packages)
    '''bad_total = 0
    for p in total_counts:
        bad_total += math.fabs(total_counts[p] - targets[p])
    print('bad match by %d packages from a target of %d' % (bad_total, target_total))'''
    write_actual_package_frequencies(total_counts)