#!/usr/bin/env python
import os, sys, subprocess, json, argparse
from common import *

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--cluster', '-c', default='cluster')
    parser.add_argument('--force', '-f', default=False, action='store_true')
    parser.add_argument('--if-running', default=False, action='store_true')
    args = parser.parse_args()

    # we'll greate a dir with a file describing each node in the cluster
    cluster_dir = os.path.join(SCRIPT_DIR, args.cluster)
    if not os.path.exists(cluster_dir):
        if args.if_running:
            sys.exit(0)
        else:
            print 'Cluster not running!'
            sys.exit(1)

    for filename in os.listdir(cluster_dir):
        path = os.path.join(cluster_dir, filename)
        if not filename.endswith('.json'):
            continue
        try:
            info = rdjs(path)
            if info['type'] == 'rethinkdb':
                for node in info['cluster']:
                    try:
                        run('docker kill '+node['cid'])
                    except Exception as e:
                        if args.force:
                            print 'continue because force was used (cleanup may not be complete)'
                        else:
                            raise e
                os.remove(path)
                print 'killed ' + info['ip'] + ' (' + filename + ')'
                continue
            cid = info['cid']
            if info['type'] == 'worker':
                # need this script, otherwise it hangs if Docker inside
                # the container has paused sub containers.
                run('docker exec '+cid+' /open-lambda/kill.py')
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
