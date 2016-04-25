#!/usr/bin/env python
import os, sys, random, string
from common import *

def main():
    apps = [
        ('hello', 'nodb.json'),
        ('echo', 'nodb.json'),
        ('thread_counter', 'thread_counter.json')
    ]
    root_dir = os.path.join(SCRIPT_DIR, '..')
    builder_dir = os.path.join(root_dir, 'lambda-generator')

    # create some applications
    for app_name, config in apps:
        # cleanup
        os.system('docker rm -f ' + app_name)
        os.system('docker rmi -f ' + app_name)

        # build image
        print '='*40
        print 'Building image'
        builder = os.path.join(builder_dir, 'builder.py')
        run(builder + ' -l %s -n %s -c %s' %
            (os.path.join(SCRIPT_DIR, 'lambdas', app_name+'.py'),
             app_name,
             os.path.join(SCRIPT_DIR, 'lambdas', config)))

    # create an application that is only in the registry
    run('docker tag -f hello nonlocal')
    run('docker tag -f nonlocal localhost:5000/nonlocal')
    run('docker push localhost:5000/nonlocal')
    run('docker rmi -f nonlocal')
    run('docker rmi -f localhost:5000/nonlocal')

    w = 80
    print '='*w
    print '= ' + 'Docker state initialized.  OK to run \"go test\" in worker now.'.ljust(w-4) + ' ='
    print '='*w

if __name__ == '__main__':
    main()
