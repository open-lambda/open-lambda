#! /usr/bin/python3

from time import time, sleep
from subprocess import Popen, call
from tempfile import mkdtemp

import os
import copy
import json
import requests

def post(path, data=None):
    return requests.post('http://localhost:5000/'+path, json.dumps(data))

ARG_SIZE=10000
arg = [[x for x in range(ARG_SIZE)] for _ in range(1)]

WASM=True
DOCKER=True

MANY=1000

OLDIR=os.path.abspath("./test-dir")
OL_REGISTRY=os.path.abspath("./test-registry")

if WASM:
    print("# WASM")
    print("Single")
    p = Popen(["./ol-wasm"])
    sleep(0.1)

    start = time()
    post("run/numpy", arg)
    end = time()

    elapsed = end - start
    print("Elapsed: %f s" % (elapsed))
    p.kill()

    print("Many (cold)")
    p = Popen(["./ol-wasm"])
    sleep(0.1)

    start = time()
    for _ in range(MANY):
        post("run/numpy", arg)
    end = time()

    elapsed = end - start
    print("Elapsed: %fs (%fs per request)" % (elapsed, elapsed/MANY))
    p.kill()

    print("Many (warm)")
    p = Popen(["./ol-wasm"])
    sleep(0.1)

    post("run/numpy", arg)

    start = time()
    for _ in range(MANY):
        post("run/numpy", arg)
    end = time()

    elapsed = end - start
    print("Elapsed: %fs (%fs per request)" % (elapsed, elapsed/MANY))
    p.kill()


if DOCKER:
    def put_conf(conf):
        global curr_conf
        with open(os.path.join(OLDIR, "config.json"), "w") as f:
            json.dump(conf, f, indent=2)
        curr_conf = conf

    def update_config(**keywords):
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

        put_conf(new)

 
    print("# DOCKER")
    call(["./ol", "kill"])

    print("Single")
    d = "./test-dir/"# mkdtemp()+"/ol/"
    call(["./ol", "new", "-p="+d])
    update_config(registry=OL_REGISTRY)

    call(["./ol", "worker", "-p="+d, "--detach"])
    sleep(0.1)

    start = time()
    r = post("run/numpy16", arg)
    end = time()
    if r.status_code != 200:
        raise Exception("STATUS %d: %s" % (r.status_code, r.text))


    elapsed = end - start
    print("Elapsed: %f s" % (elapsed))
    call(["./ol", "kill"])

    print("Many (cold)")
    d = "./test-dir/"# mkdtemp()+"/ol"
    call(["./ol", "new", "-p="+d])
    update_config(registry=OL_REGISTRY)

    p = Popen(["./ol", "worker", "--detach", "-p="+d])
    sleep(0.1)

    start = time()
    for _ in range(MANY):
        post("run/numpy16", arg)
    end = time()

    elapsed = end - start
    print("Elapsed: %fs (%fs per req)" % (elapsed, elapsed/MANY))
    call(["./ol", "kill"])

    print("Many (warm)")
    d = "./test-dir/"# mkdtemp()+"/ol"
    call(["./ol", "new", "-p="+d])
    update_config(registry=OL_REGISTRY)

    p = Popen(["./ol", "worker", "--detach", "-p="+d])
    sleep(0.1)

    post("run/numpy16", arg)

    start = time()
    for _ in range(MANY):
        post("run/numpy16", arg)
    end = time()

    elapsed = end - start
    print("Elapsed: %fs (%fs per req)" % (elapsed, elapsed/MANY))
    call(["./ol", "kill"])


