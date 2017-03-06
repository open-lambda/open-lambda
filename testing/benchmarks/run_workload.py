from time import time, sleep
import collections, os, sys, math, json, subprocess

TRACE_RUN = False

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

def clean_for_test(cluster_name):
    try:
        run('sudo ./bin/admin kill -cluster=' + cluster_name, quiet=True)
    except Exception:
        pass
    try:
        run('sudo rm -rf ' + cluster_name, quiet=True)
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

def get_time_millis():
    return int(round(time() * 1000))

def setup_cluster(cluster_name):
    try:
        run('sudo ./bin/admin new -cluster ' + cluster_name)
    except Exception:
        pass
    run('sudo cp ./testing/benchmarks/' + cluster_name + '_template.json ./' + cluster_name + '/config/template.json')
    run('sudo ./bin/admin workers -cluster=' + cluster_name)
    run('sudo cp -r ./quickstart/handlers/hello ./' + cluster_name + '/registry/hello')
    run('sudo cp -r ./testing/handlers/numpy ./' + cluster_name + '/registry/numpy')


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
    sleep(1)
    setup_cluster(type)
    sleep(1)

    min = 100000
    max = -1
    total_time = 0
    for i in range(0, iterations):
        before = get_time_millis()
        run_lambda(which_lambda)
        after = get_time_millis()
        iter_time = after - before
        total_time += iter_time
        if iter_time < min:
            min = iter_time
        if iter_time > max:
            max = iter_time
    clean_for_test(type)
    avg = int(round(total_time / iterations))
    return {'min': min, 'max': max, 'avg': avg}

# do no change these unless you also change config file cluster_name and their file names
NO_INTERPRETERS_NO_CONTAINERS = 'ninc'
INTERPRETERS_NO_CONTAINERS = 'inc'
NO_INTERPRETERS_CONTAINERS = 'nic'
INTERPRETERS_CONTAINERS = 'nc'

ITERATIONS = 5

try:
    run('sudo rm -rf perf')
except Exception:
    pass

run('sudo mkdir perf')

# No container pool and no interpreter pool
res = benchmark(NO_INTERPRETERS_NO_CONTAINERS, 'numpy', ITERATIONS)

# No container pool and interpreter pool
res = benchmark(INTERPRETERS_NO_CONTAINERS, 'numpy', ITERATIONS)
