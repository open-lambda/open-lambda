#pylint: disable=too-few-public-methods, c-extension-no-member, broad-except, global-statement

from subprocess import check_output, Popen
from time import sleep
from collections import OrderedDict

import copy
import subprocess
import requests
import os
import json
import lambdastore

OLDIR=None
REG_DIR=None
CURR_CONF=None

def setup_config(ol_dir, registry_dir):
    global OLDIR
    global REG_DIR

    OLDIR = ol_dir
    REG_DIR = os.path.abspath(registry_dir)

class Datastore:
    def __init__(self, enable_wasm=False, num_replicas=1):
        if num_replicas < 1:
            raise RuntimeError("Need at least one storage replica")

        args = []
        if enable_wasm:
            args.append("--enable_wasm=true")
            args.append("--registry_path=./test-registry.wasm")
        else:
            args.append("--enable_wasm=false")

        args.append("--replica_set_size=%i" % num_replicas)

        print("Starting lambda store")
        self._running = False
        self._coord = Popen(["lambda-store-coordinator"] + args)
        sleep(0.5)

        self._nodes = []

        for pos in range(num_replicas):
            identifier = pos+1

            node = Popen(["lambda-store-node", "--identifier=%i" % identifier,
                "-p=localhost:%i"%(50000+pos), "-l=localhost:%i"%(51000+pos)])

            self._nodes.append(node)

        self._running = True
        sleep(0.5)

        #Maybe remove this?
        print("Creating default collection")
        client = lambdastore.create_client('localhost')
        client.create_collection('default', str, {'value': str})

        self._known_programs = []

        print("Datastore set up")

    def __del__(self):
        self.stop()

    def is_running(self):
        return self._running

    @staticmethod
    def call(fn_name, args=None):
        client = lambdastore.create_client('localhost')
        client.call(fn_name, args)

    def stop(self):
        if self.is_running():
            self._running = False
        else:
            return # Already stopped

        print("Stopping lambda store")

        try:
            self._coord.terminate()
            self._coord.wait()
            self._coord = None
        except Exception as err:
            raise RuntimeError("Failed to stop lambda store coordinator: %s") from err

        try:
            for node in self._nodes:
                node.terminate()
                node.wait()

            self._nodes = []
        except Exception as err:
            raise RuntimeError("Failed to stop lambda store node") from err

def get_ol_stats():
    if os.path.exists(OLDIR+"/worker/stats.json"):
        with open(OLDIR+"/worker/stats.json") as statsfile:
            olstats = json.load(statsfile)
        return OrderedDict(sorted(list(olstats.items())))

    return None

def get_worker_output():
    with open(os.path.join(OLDIR, "worker.out")) as workerfile:
        return workerfile.read().splitlines()

def get_current_config():
    return CURR_CONF

def post(path, data=None):
    ''' Issues a post request to the OL worker '''
    return requests.post('http://localhost:5000/'+path, json.dumps(data))

def put_conf(conf):
    global CURR_CONF
    with open(os.path.join(OLDIR, "config.json"), "w") as cfile:
        json.dump(conf, cfile, indent=2)
    CURR_CONF = conf

class TestConf:
    ''' Loads a config and overwrites certain fields with what is set in **keywords '''
    def __init__(self, **keywords):
        with open(os.path.join(OLDIR, "config.json")) as cfile:
            orig = json.load(cfile)
        new = copy.deepcopy(orig)
        for key in keywords:
            if not key in new:
                raise Exception("unknown config param: %s" % key)
            if isinstance(keywords[key], dict):
                for key2 in keywords[key]:
                    new[key][key2] = keywords[key][key2]
            else:
                new[key] = keywords[key]

        # setup
        put_conf(new)
        self.orig = orig

    def __del__(self):
        # cleanup
        put_conf(self.orig)

class TestConfContext:
    def __init__(self, **keywords):
        self._conf = None
        self._keywords = keywords

    def __enter__(self):
        self._conf = TestConf(**self._keywords)

    def __exit__(self, _exc_type, _exc_value, _exc_traceback):
        self._conf = None

def run(cmd):
    print("RUN", " ".join(cmd))
    try:
        out = check_output(cmd, stderr=subprocess.STDOUT)
        fail = False
    except subprocess.CalledProcessError as err:
        out = err.output
        fail = True

    out = str(out, 'utf-8')
    if len(out) > 500:
        out = out[:500] + "..."

    if fail:
        raise Exception("command (%s) failed: %s"  % (" ".join(cmd), out))

class DatastoreWorker():
    def __init__(self):
        self._datastore = Datastore(enable_wasm=True)

    def __del__(self):
        self.stop()

    def is_running(self):
        self._datastore.is_running()

    def stop(self):
        self._datastore.stop()

    @staticmethod
    def run(fn_name, args=None):
        Datastore.call(fn_name, args)

    @staticmethod
    def name():
        return "lambda-store"

class ContainerWorker():
    def __init__(self):
        self._running = False
        self._config = TestConf(registry=REG_DIR, sandbox="sock")

        self._datastore = Datastore()

        try:
            print("Starting container worker")
            run(['./ol', 'worker', '-p='+OLDIR, '--detach'])
        except Exception as err:
            raise RuntimeError("failed to start worker: %s" % str(err)) from err

        self._running = True

    def __del__(self):
        self.stop()

    def is_running(self):
        return self._running

    @staticmethod
    def name():
        return "container"

    @staticmethod
    def run(fn_name, args=None):
        result = post("run/rust-%s"%fn_name, data=args)

        if result.status_code != 200:
            raise RuntimeError("Benchmark was not successful: %s" % result.text)

    def stop(self):
        if self.is_running():
            self._running = False
        else:
            return # Already stopped

        try:
            print("Stopping container worker")
            run(['./ol', 'kill', '-p='+OLDIR])
        except Exception as err:
            raise RuntimeError("Failed to start worker") from err

        self._datastore.stop()

class WasmWorker():
    def __init__(self):
        print("Starting WebAssembly worker")
        self._config = TestConf(registry=REG_DIR)
        self._datastore = Datastore()
        self._process = Popen(["./ol-wasm"])

        sleep(0.5)

    def __del__(self):
        self.stop()

    def is_running(self):
        return self._process is not None

    @staticmethod
    def name():
        return "wasm"

    @staticmethod
    def run(fn_name, args=None):
        result = post("run/%s"%fn_name, data=args)

        if result.status_code != 200:
            raise RuntimeError("Benchmark was not successful. %s" % result.text)

    def stop(self):
        if not self.is_running():
            return

        print("Stopping WebAssembly worker")
        self._process.terminate()
        self._process = None

        self._datastore.stop()

def prepare_open_lambda(reuse_config=False):
    '''
    Sets up the working director for open lambda,
    and stops currently running worker processes (if any)
    '''
    if os.path.exists(OLDIR):
        try:
            run(['./ol', 'kill', '-p='+OLDIR])
            print("stopped existing worker")
        except Exception as err:
            print('could not kill existing worker: %s' % str(err))

    # general setup
    if not reuse_config:
        if os.path.exists(OLDIR):
            run(['rm', '-rf', OLDIR])

        run(['./ol', 'new', '-p='+OLDIR])
    else:
        if os.path.exists(OLDIR):
            # Make sure the pid file is gone even if the previous worker crashed
            try:
                run(['rm', '-rf', '%s/worker' % OLDIR])
            except Exception as _:
                pass
        else:
            # There was never a config in the first place, create one
            run(['./ol', 'new', '-p='+OLDIR])
