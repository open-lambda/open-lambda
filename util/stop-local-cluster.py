#!/usr/bin/env python
import os, sys, subprocess, json
from common import *

def main():
    cluster_dir = os.path.join(SCRIPT_DIR, 'cluster')
    if not os.path.exists(cluster_dir):
        print 'Cluster not running!'
        sys.exit(1)

    for filename in os.listdir(cluster_dir):
        path = os.path.join(cluster_dir, filename)
        if not filename.endswith('.json'):
            continue
        info = rdjs(path)
        cid = info['cid']
        run('docker kill '+cid)
        os.remove(path)
        print 'killed ' + info['ip'] + ' (' + filename + ')'
    os.rmdir(cluster_dir)

if __name__ == '__main__':
    main()
