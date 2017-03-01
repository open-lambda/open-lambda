from time import time, sleep
import collections, os, sys, math, json, subprocess

TRACE_RUN = False

def docker_clean_container(name):
    status = docker_status(name)
    if status == 'image':
        pass
    elif status == 'paused':
        run('docker unpause '+name)
        run('docker kill '+name)
        run('docker rm '+name)
    elif status == 'running':
        run('docker kill '+name)
        run('docker rm '+name)
    elif status == 'stopped':
        run('docker rm '+name)
    elif status == 'none':
        pass
    else:
        panic()

def docker_status(name):
    try:
        js = run_js('docker inspect '+name, quiet=True)
        state = js[0].get('State')
        if state == None:
            return 'image'
        if state['Paused']:
            return 'paused'
        if state['Running']:
            return 'running'
        return 'stopped'
    except:
        return 'none'

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
    run('sudo ./bin/admin new -cluster ' + cluster_name)
    run('sudo cp ./testing/benchmarks/' + cluster_name + '_template.json ./' + cluster_name + '/config/template.json')
    run('sudo ./bin/admin workers -cluster=' + cluster_name)
    run('sudo cp -r ./quickstart/handlers/hello ./' + cluster_name + '/registry/hello')

def run_lambda(which):
    # If this throws an error it is most likely a race condition where the worker has not fully started yet
    if which == 'hello':
        run('curl -X POST localhost:8080/runLambda/hello -d \'{"name": "Alice"}\'', quiet=True)

def interpreters_no_containers():
    pass

def benchmark(stage, type, which_lambda, iterations):
    if stage == 'WARM':
        clean_for_test(type)
        sleep(1)
        setup_cluster(type)
        sleep(1)
        run_lambda(which_lambda)

    min = 100000
    max = -1
    total_time = 0
    for i in range(0, iterations):
        if stage == 'COLD':
            clean_for_test(type)
            sleep(1)
            setup_cluster(type)
            sleep(1)
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

# No container pool and no interpreter pool
# warm, worker will have code pulled already
res = benchmark('WARM', NO_INTERPRETERS_NO_CONTAINERS, 'hello', ITERATIONS)
print 'No container pool, no interpreter pool, code already pulled (min/max/avg ms): ' + \
      str(res['min']) + ' ' + str(res['max']) + ' ' + str(res['avg'])

# cold, worker will have to pull handler
res = benchmark('COLD', NO_INTERPRETERS_NO_CONTAINERS, 'hello', ITERATIONS)
print 'No container pool, no interpreter pool, code not already pulled (min/max/avg ms): ' + \
      str(res['min']) + ' ' + str(res['max']) + ' ' + str(res['avg'])

# No container pool and interpreter pool
# warm, worker will have code pulled already
res = benchmark('WARM', INTERPRETERS_NO_CONTAINERS, 'hello', ITERATIONS)
print 'No container pool, interpreter pool, code already pulled (min/max/avg ms): ' +  \
      str(res['min']) + ' ' + str(res['max']) + ' ' + str(res['avg'])

# cold, worker will have to pull handler
res = benchmark('COLD', INTERPRETERS_NO_CONTAINERS, 'hello', 5)
print 'No container pool, interpreter pool, code not already pulled (min/max/avg ms): ' + \
      str(res['min']) + ' ' + str(res['max']) + ' ' + str(res['avg'])
