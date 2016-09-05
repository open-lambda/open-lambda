#!/usr/bin/env python
import os, sys, random, string, argparse
from common import *

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--cluster', '-c', default='cluster')
    args = parser.parse_args()

    cluster_dir = os.path.join(SCRIPT_DIR, '..', 'util', args.cluster)

    apps = ['hello', 'echo', 'thread_counter']
    root_dir = os.path.join(SCRIPT_DIR, '..')
    builder_dir = os.path.join(root_dir, 'lambda-generator')
    app_dir = os.path.join(SCRIPT_DIR, 'lambdas')

    registry = rdjs(os.path.join(cluster_dir, 'registry.json'))
    rethinkdb = rdjs(os.path.join(cluster_dir, 'rethinkdb.json'))

    # push applications to the registry
    for app_name in apps:
        cmd = '%s/util/regpush %s:%s %s %s/%s.tar.gz' % (root_dir, registry['host_ip'], registry['host_port'], app_name, app_dir, app_name)
        run(cmd, True)

    # push hello2
    cmd = '%s/util/regpush %s:%s %s %s/%s.tar.gz' % (root_dir, registry['host_ip'], registry['host_port'], 'hello2', app_dir, 'hello')
    run(cmd, True)

    print '='*40

    # generate config
    print '='*40
    path = os.path.join(SCRIPT_DIR, 'worker-config.json')
    print 'writing config to ' + path
    cluster = ['%s:%s' % (rethinkdb['host_ip'], rethinkdb['host_port'])]
    config = {
        "cluster_name": args.cluster, 
        "reg_cluster": cluster,
        "registry": "olregistry",
        "worker_port": "8080"
    }
    wrjs(path, config)

    w = 80
    print '='*w
    print '= ' + 'State initialized.  OK to run \"go test\" in worker now.'.ljust(w-4) + ' ='
    print '='*w

if __name__ == '__main__':
    main()
