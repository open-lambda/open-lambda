#! /usr/bin/python3

from time import time, sleep
from subprocess import Popen, call
from tempfile import mkdtemp

import json
import requests

def post(path, data=None):
    return requests.post('http://localhost:5000/'+path, json.dumps(data))

arg = [[x for x in range(100)] for _ in range(100)]

WASM=False

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

    print("Many")
    p = Popen(["./ol-wasm"])
    sleep(0.1)

    start = time()
    for _ in range(1000):
        post("run/numpy", arg)
    end = time()

    elapsed = end - start
    print("Elapsed: %f s" % (elapsed))
    p.kill()

print("# DOCKER")
print("Single")
d = mkdtemp()+"/ol"
call(["./ol", "new", "--path="+d])

p = Popen(["./ol", "worker", "--path="+d])
sleep(0.1)

start = time()
post("run/numpy", arg)
end = time()

elapsed = end - start
print("Elapsed: %f s" % (elapsed))
p.kill()

print("Many")
d = mkdtemp()+"/ol"
call(["./ol", "new", "--path="+d])

p = Popen(["./ol", "worker", "--path="+d])
sleep(0.1)

start = time()
for _ in range(1000):
    post("run/numpy", arg)
end = time()

elapsed = end - start
print("Elapsed: %f s" % (elapsed))
p.kill()


