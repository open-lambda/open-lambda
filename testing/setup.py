#!/usr/bin/env python
import os, sys, random, string
from common import *

def main():
    app_names = ['hello', 'echo']
    root_dir = os.path.join(SCRIPT_DIR, '..')
    builder_dir = os.path.join(root_dir, 'lambda-generator')

    for app_name in app_names:
        # cleanup
        os.system('docker rm -f ' + app_name)
        os.system('docker rmi -f ' + app_name)

        # build image
        print '='*40
        print 'Building image'
        builder = os.path.join(builder_dir, 'builder.py')
        run(builder + ' -l %s -n %s -c %s' %
            (os.path.join(SCRIPT_DIR, 'lambdas/%s.py'%app_name),
             app_name,
             os.path.join(SCRIPT_DIR, 'lambdas/nodb.json')))

if __name__ == '__main__':
    main()
