#!/usr/bin/env python
import os, sys, subprocess, json, argparse
from common import *

def container_ip(cid):
    inspect = run_js('docker inspect '+cid)
    return only(inspect)['NetworkSettings']['IPAddress']

def lookup_registry_port(cid):
    inspect = run_js('docker inspect '+cid)
    return only(only(inspect)['NetworkSettings']['Ports']['5000/tcp'])['HostPort']

def main():
    parser = argparse.ArgumentParser(description='number of workers')
    parser.add_argument('--workers', '-w', default='1')
    args = parser.parse_args()

    cluster_dir = os.path.join(SCRIPT_DIR, 'cluster')
    if os.path.exists(cluster_dir):
        print 'Cluster already running!'
        sys.exit(1)
    os.mkdir(cluster_dir)

    # start registry
    c = 'docker run -d -p 5000 registry:2'
    cid = run(c).strip()
    registry_ip = container_ip(cid)
    registry_port = lookup_registry_port(cid)
    config = {'cid': cid,
              'ip': registry_ip,
              'host_port': registry_port}
    config_path = os.path.join(cluster_dir, 'registry.json')
    wrjs(config_path, config)
    print 'started registry ' + registry_ip + ':5000 (or localhost:' + registry_port + ')'
    print '='*40

    # start workers
    workers = []
    assert(int(args.workers) > 0)
    for i in range(int(args.workers)):
        config = {'registry_host': registry_ip,
                  'registry_port': '5000'}
        config_path = os.path.join(cluster_dir, 'worker-%d.json' % i)
        wrjs(config_path, config)
        volumes = [('/sys/fs/cgroup', '/sys/fs/cgroup'),
                   (config_path, '/open-lambda-config.js')]
        c = 'docker run -d --privileged <VOLUMES> lambda-node'
        c = c.replace('<VOLUMES>', ' '.join(['-v %s:%s'%(host,guest)
                                             for host,guest in volumes]))
        cid = run(c).strip()
        config['cid'] = cid
        config['ip'] = container_ip(cid)
        wrjs(config_path, config, atomic=True)

        info_path = os.path.join(cluster_dir, 'worker-info-%d.json' % i)
        print 'started worker ' + config['ip']
        workers.append(config)
    print '='*40
    print 'Push images to OpenLambda registry as follows (or similar):'
    print 'docker tag hello localhost:%s/hello; docker push localhost:%s/hello' % (registry_port, registry_port)
    print '='*40
    print 'Send requests as follows (or similar):'
    print "curl -X POST %s:8080/runLambda/hello -d '{}'" % workers[-1]['ip']

if __name__ == '__main__':
    main()
