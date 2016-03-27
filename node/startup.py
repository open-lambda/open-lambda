#!/usr/bin/env python
import os, sys, time, json

def get_config():
    path = '/open-lambda-config.js'
    if not os.path.exists(path):
        return {}
    with open(path) as f:
        return json.loads(f.read())

def cmd(c, check=True):
    print c
    rv = os.system(c)
    if check:
        assert(rv == 0)

def main():
    config = get_config()

    # start Docker in container
    PID_FILE = '/tmp/docker.pid'
    STORAGE_DRIVER = 'aufs'
    GRAPH = '/docker_vol'
    c = ('docker -d --pidfile=<PID_FILE> ' +
         '--storage-driver=<STORAGE_DRIVER> ' +
         '--insecure-registry=<REGISTRY_HOST>:<REGISTRY_PORT> '+
         '--graph=<GRAPH> &> /tmp/docker.log &')
    cmd(c
        .replace('<PID_FILE>', PID_FILE)
        .replace('<STORAGE_DRIVER>', STORAGE_DRIVER)
        .replace('<GRAPH>', GRAPH)
        .replace('<REGISTRY_HOST>', config.get('registry_host', 'localhost'))
        .replace('<REGISTRY_PORT>', config.get('registry_port', '5000'))
    )
    # wait up to 5 seconds for startup
    for i in range(5):
        if os.path.exists(PID_FILE):
            break
        time.sleep(1)
    assert(os.path.exists(PID_FILE))

    # start lambda worker
    c = ('/open-lambda/bin/worker <REGISTRY_HOST> <REGISTRY_PORT>')
    cmd(c
        .replace('<REGISTRY_HOST>', config.get('registry_host', 'localhost'))
        .replace('<REGISTRY_PORT>', config.get('registry_port', '5000')))

if __name__ == '__main__':
    main()
