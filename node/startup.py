#!/usr/bin/env python
import os, sys, time

def cmd(c, check=True):
    print c
    rv = os.system(c)
    if check:
        assert(rv == 0)

def main():
    PID_FILE = '/tmp/docker.pid'
    STORAGE_DRIVER = 'aufs'
    GRAPH = '/docker_vol'
    c = ('docker -d --pidfile=<PID_FILE> --storage-driver=<STORAGE_DRIVER> ' +
         '--graph=<GRAPH> &> /tmp/docker.log &')
    cmd(c
        .replace('<PID_FILE>', PID_FILE)
        .replace('<STORAGE_DRIVER>', STORAGE_DRIVER)
        .replace('<GRAPH>', GRAPH)
    )
    # wait up to 5 seconds for startup
    for i in range(5):
        if os.path.exists(PID_FILE):
            break
        time.sleep(1)
    assert(os.path.exists(PID_FILE))

    cmd('/open-lambda/bin/worker localhost 5000')

if __name__ == '__main__':
    main()
