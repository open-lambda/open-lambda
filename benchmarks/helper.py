from subprocess import check_output, Popen
from time import sleep

import copy
import subprocess
import requests
import os
import json

OLDIR="./bench-dir"
BENCH_FILTER=[]
REG_DIR=os.path.abspath("test-registry")

''' Issues a post request to the OL worker '''
def post(path, data=None):
    return requests.post('http://localhost:5000/'+path, json.dumps(data))

def bench_in_filter(name):
    if len(BENCH_FILTER) == 0:
        return True

    return name in BENCH_FILTER

def put_conf(conf):
    global curr_conf
    with open(os.path.join(OLDIR, "config.json"), "w") as f:
        json.dump(conf, f, indent=2)
    curr_conf = conf

''' Loads a config and overwrites certain fields with what is set in **keywords '''
class BenchConf:
    def __init__(self, **keywords):
        with open(os.path.join(OLDIR, "config.json")) as f:
            orig = json.load(f)
        new = copy.deepcopy(orig)
        for k in keywords:
            if not k in new:
                raise Exception("unknown config param: %s" % k)
            if type(keywords[k]) == dict:
                for k2 in keywords[k]:
                    new[k][k2] = keywords[k][k2]
            else:
                new[k] = keywords[k]

        # setup
        put_conf(new)
        self.orig = orig

    def __del__(self):
        # cleanup
        put_conf(self.orig)

def run(cmd):
    print("RUN", " ".join(cmd))
    try:
        out = check_output(cmd, stderr=subprocess.STDOUT)
        fail = False
    except subprocess.CalledProcessError as e:
        out = e.output
        fail = True

    out = str(out, 'utf-8')
    if len(out) > 500:
        out = out[:500] + "..."

    if fail:
        raise Exception("command (%s) failed: %s"  % (" ".join(cmd), out))

class ContainerWorker():
    def __init__(self):
        self._running = False
        self._config = BenchConf(registry=REG_DIR, sandbox="sock")

        try:
            print("Starting container worker")
            run(['./ol', 'worker', '-p='+OLDIR, '--detach'])
        except Exception as e:
            raise RuntimeError("failed to start worker: %s" % str(e))

        self._running = True

    def __del__(self):
        self.stop()

    def is_running(self):
        return self._running

    def name(self):
        return "container"

    def run(self, fn_name, args=None):
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
        except Exception as e:
            raise RuntimeError("failed to start worker: %s" % str(e))

class WasmWorker():
    def __init__(self):
        print("Starting WebAssembly worker")
        self._config = BenchConf(registry=REG_DIR)
        self._process = Popen(["./ol-wasm"])

        sleep(0.5)

    def __del__(self):
        self.stop()

    def is_running(self):
        return self._process != None

    def name(self):
        return "wasm"

    def run(self, fn_name, args=None):
        result = post("run/%s"%fn_name, data=args)

        if result.status_code != 200:
            raise RuntimeError("Benchmark was not successful. %s" % result.text)

    def stop(self):
        if not self.is_running():
            return

        print("Stopping WebAssembly worker")
        self._process.kill()
        self._process = None

'''
Sets up the working director for open lambda,
and stops currently running worker processes (if any)
'''
def prepare_open_lambda(reuse_config=False):
    if os.path.exists(OLDIR):
        try:
            run(['./ol', 'kill', '-p='+OLDIR])
            print("stopped existing worker")
        except Exception as e:
            print('could not kill existing worker: %s' % str(e))

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
            except:
                pass
        else:
            # There was never a config in the first place, create one
            run(['./ol', 'new', '-p='+OLDIR])

