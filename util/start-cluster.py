#!/usr/bin/env python
import argparse
from cluster_manager import *

DB_WAIT = '--skip-db-wait'
REGISTRY_PORT = '5000'
WORKER_PORT =   '8080'
BALANCER_PORT = '85'

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--workers', '-w', default='1')
    parser.add_argument('--cluster', '-c', default='cluster')
    parser.add_argument(DB_WAIT, default=False, action='store_true')
    parser.add_argument('--olregistry', default=False, action='store_true')
    parser.add_argument('--nrethink', default='1')

    # TODO
    parser.add_argument('--remote', default=False, action='store_true')

    global args
    args = parser.parse_args()

    ports = {
        'registry': REGISTRY_PORT,
        'balancer': BALANCER_PORT,
        'worker':   WORKER_PORT
    }

    if args.remote:
        cluster = RemoteCluster(args.workers, args.cluster, ports)
    else:
        cluster = LocalCluster(args.workers, args.cluster, ports)

    cluster.make_cluster_dir()
    print '='*40

    if args.olregistry:
        cluster.start_olreg(int(args.nrethink))
    else:
        cluster.start_docker_reg()
    print '='*40

    cluster.start_workers(args.olregistry) 
    print '='*40

    cluster.start_lb()
    print '='*40

    if args.skip_db_wait:
        print 'Not waiting for rethinkdb'
    else:
        print 'To continue without waiting for the DB, use %s' % DB_WAIT 
        cluster.rethinkdb_wait()

    cluster.print_directions()

if __name__ == '__main__':
    main()
