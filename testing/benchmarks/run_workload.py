from time import time, sleep
import collections, os, sys, math, json, subprocess, shutil

TRACE_RUN = False
SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))

def run(cmd, quiet=False):
    if TRACE_RUN:
        print 'EXEC ' + cmd
    if quiet:
        err = open('/dev/null')
        rv = subprocess.check_output(cmd, shell=True, stderr=err)
        err.close()
    else:
        rv = subprocess.check_output(cmd, shell=True)
    return rv

def debug_clean():
    try:
        run('sudo kill `sudo lsof -t -i:8080`', quiet=True)
    except Exception:
        pass
    try:
        run('sudo docker unpause $(docker ps -a -q)', quiet=True)
    except Exception:
        pass
    try:
        run('sudo docker kill $(docker ps -a -q)', quiet=True)
    except Exception:
        pass
    try:
        run('sudo docker rm $(docker ps -a -q)', quiet=True)
    except Exception:
        pass

def clean_for_test(cluster_name):
    try:
        run('sudo %s/../../bin/admin kill -cluster=%s/%s' % (SCRIPT_DIR, SCRIPT_DIR, cluster_name), quiet=True)
    except Exception:
        pass
    try:
        run('sudo rm -rf %s/%s' % (SCRIPT_DIR, cluster_name), quiet=True)
    except Exception:
        pass

def get_default_config(cluster_name):
    return {
        "registry": "local",
        "sandbox": "docker",
        "reg_dir": "../registry",
        "cluster_name": "%s" % cluster_name,
        "worker_dir": "workers/default",
        "benchmark_log": "./perf/%s.perf" % cluster_name,
        "sandbox_buffer": 0,
        "num_forkservers": 1
    }

def add_interpreter_pool(config, num_forkservers, pool, pool_dir):
    config['num_forkservers'] = num_forkservers
    config['pool'] = pool
    config['pool_dir'] = pool_dir
    return config

def add_container_pool(config, sandbox_buffer):
    config['sandbox_buffer'] = sandbox_buffer
    return config

def get_time_millis():
    return int(round(time() * 1000))

def setup_cluster(cluster_name, config):
    # Create cluster
    run('sudo %s/../../bin/admin new -cluster %s/%s' % (SCRIPT_DIR, SCRIPT_DIR, cluster_name))
    # Write worker config
    worker_template_f = open('%s/%s/config/template.json' % (SCRIPT_DIR, cluster_name), 'w')
    json.dump(config, worker_template_f)
    worker_template_f.close()
    # Start worker
    run('sudo %s/../../bin/admin workers -cluster=%s/%s' % (SCRIPT_DIR, SCRIPT_DIR, cluster_name))

def copy_handlers(cluster_name):
    shutil.rmtree('%s/%s/registry' % (SCRIPT_DIR, cluster_name))
    shutil.copytree(SCRIPT_DIR + '/handlers', '%s/%s/registry' % (SCRIPT_DIR, cluster_name))

def run_lambda(which):
    # If this throws an error it is most likely a race condition where the worker has not fully started yet
    if which == 'hello':
        run('curl -X POST localhost:8080/runLambda/hello -d \'{"name": "Alice"}\'', quiet=True)
    elif which == 'numpy':
        run('curl -X POST localhost:8080/runLambda/numpy -d \'{"name": "Alice"}\'', quiet=True)

def benchmark(cluster_name, config, which_lambda, iterations):
    clean_for_test(cluster_name)
    #debug_clean()
    setup_cluster(cluster_name, config)
    copy_handlers(cluster_name)
    sleep(1)

    for i in range(0, iterations):
        print('try req for '+ cluster_name)
        run_lambda(which_lambda)

    sleep(1)
    clean_for_test(cluster_name)
    #debug_clean()

if not os.path.exists('perf'):
    os.makedirs('perf')

config = get_default_config('ninc')
benchmark('ninc', config, 'hello', 5)

config = get_default_config('inc')
config = add_interpreter_pool(config, num_forkservers=1, pool='basic', pool_dir='inc/pool')
benchmark('inc', config, 'hello', 5)

config = get_default_config('nic')
config = add_container_pool(config, 5)
benchmark('nic', config, 'hello', 5)

config = get_default_config('ic')
config = add_interpreter_pool(config, num_forkservers=1, pool='basic', pool_dir='ic/pool')
config = add_container_pool(config, 5)
benchmark('ic', config, 'hello', 5)