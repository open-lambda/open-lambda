#!/usr/bin/env python
import os, sys, json, time
from collections import defaultdict as ddict
import random

def join_cg(cg):
    with open(cg+'/cgroup.procs', 'w') as f:
        f.write(str(os.getpid()))

def usage(cg):
    try:
        with open(cg+'/memory.usage_in_bytes') as f:
            return int(f.read()) / 1024 / 1024
    except:
        return 'unknown'

def main():
    cg1 = '/sys/fs/cgroup/memory/%d' % random.randint(0, 1000)
    cg2 = '/sys/fs/cgroup/memory/%d' % random.randint(0, 1000)
    print("CG1", cg1)
    print("CG2", cg2)
    os.mkdir(cg1)
    os.mkdir(cg2)

    #for cg in (cg1, cg2):
    #    with open(cg+"/memory.move_charge_at_immigrate", "w") as f:
    #        f.write("1")

    join_cg(cg1)
    time.sleep(1)
    print('after join cg1: ', usage(cg1), usage(cg2))

    A = 'a' * 100000000 # 100MB
    print('after allocating A=100MB: ', usage(cg1), usage(cg2))

    B = 'B' * 200000000 # 200MB
    print('after allocating B=200MB: ', usage(cg1), usage(cg2))

    pid = os.fork()
    assert(pid >= 0)

    if pid != 0:
        print('after fork: ', usage(cg1), usage(cg2))

        time.sleep(1)
        A = '' # free
        print('after freeing A (parent): ', usage(cg1), usage(cg2))

        C = 'c' * 50000000 # 50MB
        print('after allocating C=50MB (parent): ', usage(cg1), usage(cg2))

        C = '' # free
        print('after freeing C (parent): ', usage(cg1), usage(cg2))

        time.sleep(2)
        os._exit(0)

    else:
        join_cg(cg2)
        time.sleep(2)

        A = '' # free
        print('after freeing A (child): ', usage(cg1), usage(cg2))

        D = 'd' * 50000000 # 50MB
        print('after allocating D=50MB (child): ', usage(cg1), usage(cg2))

        D = '' # free
        print('after freeing D (child): ', usage(cg1), usage(cg2))

        time.sleep(2)
        print('after parent exited: ', usage(cg1), usage(cg2))

        with open(cg1+"/cgroup.procs") as f:
            print('parent procs: ', f.read())

        if 0:
            with open(cg1+"/memory.limit_in_bytes", "w") as f:
                f.write("1M")
            with open(cg1+"/memory.limit_in_bytes") as f:
                print("after trying to set parent CG to 1MB, the limit is: ", f.read())

        while True:
            print('curr mem: ', usage(cg1), usage(cg2))
            time.sleep(1)

if __name__ == '__main__':
    main()
