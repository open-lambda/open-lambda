import httplib
import os
import subprocess
import time

def join_paths(base, relative):
    return os.path.normpath(os.path.join(base, relative))

INTEGRATION_TESTS_DIR = os.path.dirname(os.path.realpath(__file__))
TEST_CLUSTER_DIR = join_paths(INTEGRATION_TESTS_DIR, '../test-cluster')
REGISTRY_DIR = join_paths(INTEGRATION_TESTS_DIR, '../registry')
ADMIN_BIN = join_paths(INTEGRATION_TESTS_DIR, '../../bin/admin')
WORKER_TIMEOUT = 30

def kill_worker():
    template = """{bin} kill -cluster={cluster_dir}; rm -rf {cluster_dir}/workers/*"""
    data = { 'bin': ADMIN_BIN, 'cluster_dir': TEST_CLUSTER_DIR }
    cmd = template.format(**data)
    subprocess.call(cmd, shell=True)

def is_worker_active():
    template = """{bin} status -cluster={cluster_dir}"""
    data = { 'bin': ADMIN_BIN, 'cluster_dir': TEST_CLUSTER_DIR }
    cmd = template.format(**data)
    devnull = open(os.devnull, 'w')
    exit_code = subprocess.call(cmd, shell=True, stdout=devnull)
    return exit_code == 0

def set_worker_conf(conf):
    template = """{bin} setconf -cluster={cluster_dir} '{conf}'"""
    data = { 'bin': ADMIN_BIN, 'cluster_dir': TEST_CLUSTER_DIR, 'conf': conf }
    cmd = template.format(**data)
    subprocess.call(cmd, shell=True)

def start_test_worker():
    template = """{bin} workers -cluster={cluster_dir}"""
    data = { 'bin': ADMIN_BIN, 'cluster_dir': TEST_CLUSTER_DIR }
    cmd = template.format(**data)
    subprocess.call(cmd, shell=True)

def init_worker(conf):
    set_worker_conf(conf)
    start_test_worker()

def assert_worker_is_ready():
    for i in range(WORKER_TIMEOUT):
        time.sleep(2)
        if is_worker_active():
            return 
    raise IOError('Worker failed to initialize after ' 
            + str(WORKER_TIMEOUT * 2) + 'seconds.')

def run_lambda(name):
    conn = httplib.HTTPConnection('localhost', 8080, timeout=15)
    url = '/runLambda/' + name
    conn.request('POST', url, '{}')
    response = conn.getresponse()
    if response.status != 200:
        template = """"Request to run lambda '{name}'failed with status code{status}."""
        data = { 'name': name, 'status': response.status }
        msg = template.format(**data)
        raise IOError(msg)

def create_test_cluster():
    template = """{bin} new -cluster={cluster_dir}"""
    data = { 'bin': ADMIN_BIN, 'cluster_dir': TEST_CLUSTER_DIR }
    cmd = template.format(**data)
    subprocess.call(cmd, shell=True)

def start_test_cluster(): 
    print("Starting test cluster...")
    create_test_cluster()

    STARTUP_PKGS_CONF ='{"startup_pkgs": ["parso", "jedi", "urllib3", "idna", "chardet", "certifi", "requests", "simplejson"]}'
    REGISTRY_DIR_CONF ='{"registry_dir": "' + REGISTRY_DIR + '"}'
    set_worker_conf(STARTUP_PKGS_CONF)
    set_worker_conf(REGISTRY_DIR_CONF)

def run_cluster_test_with_conf(conf):
    print('Killing worker if running...')
    kill_worker()
    print('Starting worker...')
    init_worker(conf)
    print('Waiting for worker to initialize...')
    assert_worker_is_ready()
    print('Worker ready. Requesting lambdas...')
    run_lambda('echo')
    run_lambda('install')
    run_lambda('install2')
    run_lambda('install3')


