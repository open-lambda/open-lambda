#!/usr/bin/python
import sys, os
from subprocess import check_output

PKG_PATH = '/handler/packages.txt'
HOST_PATH = '/host'
STDOUT_PATH = '%s/stdout' % HOST_PATH
STDERR_PATH = '%s/stderr' % HOST_PATH

def main():
    sys.stdout = open(STDOUT_PATH, 'w')
    sys.stderr = open(STDERR_PATH, 'w')

    if not os.path.exists(PKG_PATH):
        print('no packages to install')
        return

    with open(PKG_PATH) as fd:
        for line in fd:
            try:
                pkg = line.split(':')[1]
                if pkg != '':
                    check_output(['pip', 'install', '--index-url', 'http://192.168.103.144:9199/simple', '--trusted-host', '192.168.103.144', pkg])
                    install(pkg)
            except Exception as e:
                print('failed to install %s with %s' % (pkg, e))

if __name__ == '__main__':
    main()
