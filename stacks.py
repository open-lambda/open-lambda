#!/usr/bin/env python3

# debug tool, dumps stacks of every goroutine for the specified worker

import os, sys, json, re
from collections import defaultdict as ddict
from subprocess import check_output

def main():
    dirname = 'default-ol'
    if len(sys.argv) > 1:
        dirname = sys.argv[1]

    with open(dirname+'/worker/worker.pid') as f:
        pid = f.read()
    info = check_output(['gdb', 'ol', pid, '-batch', '-ex', 'info goroutines'])
    info = str(info, 'utf-8')
    info = info.replace("*", " ")
    ids = []
    for line in info.split("\n"):
        m = re.match(r'  (\d+) ', line)
        if m:
            ids.append(int(m.group(1)))

    for gid in ids:
        bt = check_output(['gdb', 'ol', pid, '-batch', '-ex', 'goroutine %d bt' % gid])
        print(str(bt, 'utf-8'))


if __name__ == '__main__':
    main()
