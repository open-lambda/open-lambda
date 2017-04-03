from time import time, sleep
import collections, os, sys, math, json, subprocess

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

def debug_clean(cluster_name):
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


def get_time_millis():
    return int(round(time() * 1000))

def setup_cluster(cluster_name):
    run('sudo %s/../../bin/admin new -cluster %s/%s' % (SCRIPT_DIR, SCRIPT_DIR, cluster_name))
    run('sudo cp %s/%s_template.json %s/%s/config/template.json' % (SCRIPT_DIR, cluster_name, SCRIPT_DIR, cluster_name))
    run('sudo %s/../../bin/admin workers -cluster=%s/%s' % (SCRIPT_DIR, SCRIPT_DIR, cluster_name))

    run('sudo cp -r %s/../../quickstart/handlers/hello %s/%s/registry/hello' % (SCRIPT_DIR, SCRIPT_DIR, cluster_name))
    run('sudo cp -r %s/../../quickstart/handlers/hello %s/%s/registry/numpy' % (SCRIPT_DIR, SCRIPT_DIR, cluster_name))


def run_lambda(which):
    # If this throws an error it is most likely a race condition where the worker has not fully started yet
    if which == 'hello':
        run('curl -X POST localhost:8080/runLambda/hello -d \'{"name": "Alice"}\'', quiet=True)
    elif which == 'numpy':
        run('curl -X POST localhost:8080/runLambda/numpy -d \'{"name": "Alice"}\'', quiet=True)

def interpreters_no_containers():
    pass

def benchmark(type, which_lambda, iterations):
    clean_for_test(type)
    #debug_clean(type)
    setup_cluster(type)
    sleep(1)

    for i in range(0, iterations):
        print('try req for '+ type)
        run_lambda(which_lambda)

    sleep(1)
    clean_for_test(type)
    #debug_clean(type)


# do no change these unless you also change config file cluster_name and their file names
NO_INTERPRETERS_NO_CONTAINERS = 'ninc'
INTERPRETERS_NO_CONTAINERS = 'inc'
NO_INTERPRETERS_CONTAINERS = 'nic'
INTERPRETERS_CONTAINERS = 'ic'

ITERATIONS = 5

try:
    run('sudo rm -rf perf')
except Exception:
    pass

run('sudo mkdir perf')

# No container pool and no interpreter pool
benchmark(NO_INTERPRETERS_NO_CONTAINERS, 'hello', ITERATIONS)

# No container pool and interpreter pool
benchmark(INTERPRETERS_NO_CONTAINERS, 'hello', ITERATIONS)

# container pool and no interpreter pool
benchmark(NO_INTERPRETERS_CONTAINERS, 'hello', ITERATIONS)

# container pool and interpreter pool
benchmark(INTERPRETERS_CONTAINERS, 'hello', ITERATIONS)
