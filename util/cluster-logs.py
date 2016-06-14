#!/usr/bin/env python
import os, sys, subprocess, json, argparse
from common import *

def main():
    cluster_dir = os.path.join(SCRIPT_DIR, 'cluster')
    if not os.path.exists(cluster_dir):
        print 'Cluster not running!'
        sys.exit(1)

    logs = []
    for filename in os.listdir(cluster_dir):
        path = os.path.join(cluster_dir, filename)
        if not filename.endswith('.json'):
            continue
        try:
            info = rdjs(path)
            cid = info['cid']
            if info['type'] == 'worker':
		logs.append("Worker CID: %s\n" % cid)
		logs.append(subprocess.check_output(['docker', 'logs', cid]))

        except Exception as e:
            print e

    return ''.join(logs)

if __name__ == '__main__':
    print main()
