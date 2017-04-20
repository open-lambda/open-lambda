import collections, os, sys, math, json, subprocess, shutil, argparse, time

TRACE_RUN = False
SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))

def run(cmd, quiet=False):
    if TRACE_RUN:
        print('EXEC ' + cmd)
    if quiet:
        err = open('/dev/null')
        rv = subprocess.check_output(cmd, shell=True, stderr=err)
        err.close()
    else:
        rv = subprocess.check_output(cmd, shell=True)
    return rv

def debug_clean():
    try:
        run('kill `sudo lsof -t -i:8080`', quiet=True)
    except Exception:
        pass
    try:
        run('docker unpause $(docker ps -a -q)', quiet=True)
    except Exception:
        pass
    try:
        run('docker kill $(docker ps -a -q)', quiet=True)
    except Exception:
        pass
    try:
        run('docker rm $(docker ps -a -q)', quiet=True)
    except Exception:
        pass

def kill_for_test(cluster_name):
    try:
        run('%s/../../bin/admin kill -cluster=%s/%s' % (SCRIPT_DIR, SCRIPT_DIR, cluster_name), quiet=True)
    except Exception:
        pass

def clean_for_test(cluster_name):
    try:
        run('rm -rf %s/%s' % (SCRIPT_DIR, cluster_name), quiet=True)
    except Exception:
        pass

def get_default_config(cluster_name):
    return {
        "registry": "local",
        "sandbox": "docker",
        "reg_dir": "%s/registry" % (SCRIPT_DIR),
        "cluster_name": "%s" % cluster_name,
        "worker_dir": "workers/default",
        "benchmark_log": "%s/perf/%s.perf" % (SCRIPT_DIR, cluster_name),
        "sandbox_buffer": 25,
        "sandbox_config": {},
        "pip_mirror": "172.17.0.1",
        "pkgs_dir": "%s/empty" % SCRIPT_DIR,
        "import_cache_size": 0,
        "handler_cache_size": 0,
    }

def add_handler_cache(config):
    config['handler_cache_size'] = 1000000
    return config

def add_install_cache(config, d):
    config['pkgs_dir'] = d
    return config

def add_import_cache(config, d, size):
    config['import_cache_size'] = size
    config['import_cache_dir'] = d
    return config

def setup_cluster(cluster_name, config):
    # Create cluster
    run('%s/../../bin/admin new -cluster %s/%s' % (SCRIPT_DIR, SCRIPT_DIR, cluster_name))
    # Write worker config
    worker_template_f = open('%s/%s/config/template.json' % (SCRIPT_DIR, cluster_name), 'w')
    json.dump(config, worker_template_f, indent=4, separators=(',', ': '))
    worker_template_f.close()
    # Start worker
    run('%s/../../bin/admin workers -cluster=%s/%s' % (SCRIPT_DIR, SCRIPT_DIR, cluster_name))

def copy_handlers(cluster_name):
    shutil.rmtree('%s/%s/registry' % (SCRIPT_DIR, cluster_name))
    shutil.copytree(SCRIPT_DIR + '/../handlers', '%s/%s/registry' % (SCRIPT_DIR, cluster_name))


if not os.path.exists('perf'):
    os.makedirs('perf')

parser = argparse.ArgumentParser(description='Start a cluster')
parser.add_argument('-cluster', default='test')
parser.add_argument('--stop', action='store_true')
parser.add_argument('--remove', action='store_true')
parser.add_argument('--stop-all', action='store_true')
parser.add_argument('--start', action='store_true')
parser.add_argument('--handler-cache', action='store_true')
parser.add_argument('--import-cache', default=5242880)
parser.add_argument('--install-cache', action='store_true')
parser.add_argument('--pipbench', action='store_true')
args = parser.parse_args()

if args.stop_all:
    debug_clean()
    print('All Docker containers and workers stopped')
    exit()

if args.cluster is None:
    print('Specify a cluster name')
    exit()

if args.stop:
    kill_for_test(args.cluster)
    print('Cluster %s stopped' % args.cluster)
    if not args.remove:
        exit()

if args.remove:
    clean_for_test(args.cluster)
    print('Cluster %s removed' % args.cluster)
    exit()


if args.start:
    config = get_default_config(args.cluster)
    if args.handler_cache:
        config = add_handler_cache(config)
    if args.import_cache:
        d = '%s/%s/import-cache' % (SCRIPT_DIR, args.cluster)
        config = add_import_cache(config, d, int(args.import_cache))
    if args.install_cache:
        d = '%s/install-cache' % (SCRIPT_DIR)
        config = add_install_cache(config, d)
    if args.pipbench:
        config = add_pipbench(config)
    setup_cluster(args.cluster, config)
    print('Cluster %s started' % args.cluster)
else :
    print('Specify a command')
