#!/usr/bin/env python
import os, sys, subprocess, json, argparse
from common import *

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--force', '-f', default=False, action='store_true')
    args = parser.parse_args()

    cluster_dir = os.path.join(SCRIPT_DIR, 'cluster')
    if not os.path.exists(cluster_dir):
        print 'Cluster not running!'
        sys.exit(1)

    for filename in os.listdir(cluster_dir):
        path = os.path.join(cluster_dir, filename)
        if not filename.endswith('.json'):
            continue
        try:
            info = rdjs(path)
            cid = info['cid']
            run('docker kill '+cid)
        except Exception as e:
            print e
            if args.force:
                print 'continue because force was used (cleanup may not be complete)'
            else:
                print 'giving up (consider using --force or manually cleaning up)'
                sys.exit(1)
        os.remove(path)
        print 'killed ' + info['ip'] + ' (' + filename + ')'
    os.rmdir(cluster_dir)

if __name__ == '__main__':
    main()
