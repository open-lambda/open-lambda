#!/usr/bin/env python
import os, sys, subprocess, json
from common import *

def run(cmd):
    print 'EXEC ' + cmd
    return subprocess.check_output(cmd, shell=True)

def run_js(cmd):
    return json.loads(run(cmd))

def container_ip(cid):
    inspect = run_js('docker inspect '+cid)
    return only(inspect)['NetworkSettings']['IPAddress']

def registry_port(cid):
    inspect = run_js('docker inspect '+cid)
    return only(inspect)['NetworkSettings']['Ports']['5000/tcp']['HostPort']

def main():
    cluster_dir = os.path.join(SCRIPT_DIR, 'cluster')
    if not os.path.exists(cluster_dir):
        os.mkdir(cluster_dir)

    # start registry
    c = 'docker run -d -p 5000 registry:2'
    cid = run(c).strip()
    registry_ip = container_ip(cid)
    config = {'cid': cid,
              'ip': registry_ip,
              'host_port': registry_port(cid)}
    config_path = os.path.join(cluster_dir, 'registry.json' % i)

    # start workers
    for i in range(1):
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

        # TODO: run('docker kill '+cid)

        info_path = os.path.join(cluster_dir, 'worker-info-%d.json' % i)

if __name__ == '__main__':
    main()
