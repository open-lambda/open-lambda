#!/usr/bin/env python
import os, sys, random, string
from common import *

NGINX_EXAMPLE = 'docker run -d -p 80:80 -v %s:/usr/share/nginx/html:ro nginx'

def main():
    app_name = ''.join(random.choice(string.ascii_lowercase) for _ in range(12))
    static_dir = os.path.join(SCRIPT_DIR, 'static')
    root_dir = os.path.join(SCRIPT_DIR, '..', '..')
    cluster_dir = os.path.join(root_dir, 'util', 'cluster')
    builder_dir = os.path.join(root_dir, 'lambda-generator')
    if not os.path.exists(cluster_dir):
        return 'cluster not running'

    # build image
    print '='*40
    print 'Building image'
    builder = os.path.join(builder_dir, 'builder.py')
    run(builder + ' -n %s -l %s -c %s -e %s' %
        (app_name,
         os.path.join(SCRIPT_DIR, 'ocr.py'),
         os.path.join(SCRIPT_DIR, 'lambda-config.json'),
         os.path.join(SCRIPT_DIR, 'environment.json')))

    # push image
    print '='*40
    print 'Pushing image'
    registry = rdjs(os.path.join(cluster_dir, 'registry.json'))
    img = 'localhost:%s/%s' % (registry['host_port'], app_name)
    run('docker tag -f %s %s' % (app_name, img))
    run('docker push ' + img)

    # setup config
    balancer = rdjs(os.path.join(cluster_dir, 'loadbalancer-1.json'))
    config_file = os.path.join(static_dir, 'config.json')
    url = ("http://%s:%s/runLambda/%s" %
           (balancer['host_ip'], balancer['host_port'], app_name))
    wrjs(config_file, {'url': url})

    # directions
    print '='*40
    print 'Consider serving the app with nginx as follows:'
    print NGINX_EXAMPLE % static_dir
    return None

if __name__ == '__main__':
    rv = main()
    if rv != None:
        print 'ERROR: ' + rv
        sys.exit(1)
    sys.exit(0)
