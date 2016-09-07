#!/usr/bin/env python
import os, sys, random, string, argparse
from common import *

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--cluster', '-c', default='cluster')
    args = parser.parse_args()

    cluster_dir = os.path.join(SCRIPT_DIR, '..', 'util', args.cluster)

    apps = [
        ('hello', 'nodb.json'),
        ('echo', 'nodb.json'), ('thread_counter', 'thread_counter.json') ]
    root_dir = os.path.join(SCRIPT_DIR, '..')
    builder_dir = os.path.join(root_dir, 'lambda-generator')

    # create some applications
    for app_name, config in apps:
        # cleanup
        docker_clean_container(app_name)

        # build image
        print '='*40
        print 'Building image'
        builder = os.path.join(builder_dir, 'builder.py')
        run(builder + ' -l %s -n %s -c %s' %
            (os.path.join(SCRIPT_DIR, 'lambdas', app_name+'.py'),
             app_name,
             os.path.join(SCRIPT_DIR, 'lambdas', config)))

    # cleanup aliased containers too
    docker_clean_container('hello2')
    docker_clean_container('nonlocal')

    print '='*40

    # create an application that is only in the registry
    registry_config = rdjs(os.path.join(cluster_dir, 'registry.json'))
    TEST_REGISTRY = 'localhost:' + registry_config['host_port']
    print 'Push test images to ' + TEST_REGISTRY

    run('docker tag -f hello nonlocal')
    run('docker tag -f hello hello2')
    run('docker tag -f nonlocal %s/nonlocal' % TEST_REGISTRY)
    run('docker push %s/nonlocal' % TEST_REGISTRY)
    run('docker rmi -f nonlocal')
    run('docker rmi -f %s/nonlocal' % TEST_REGISTRY)

    # generate config
    print '='*40
    path = os.path.join(SCRIPT_DIR, 'worker-config.json')
    print 'writing config to ' + path
    config = {
        "cluster_name": args.cluster, 
        "registry_host": "localhost", 
        "registry_port": registry_config['host_port'], 
        "registry": "docker"
    }
    wrjs(path, config)

    w = 80
    print '='*w
    print '= ' + 'State initialized.  OK to run \"go test\" in worker now.'.ljust(w-4) + ' ='
    print '='*w

if __name__ == '__main__':
    main()
