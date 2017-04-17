#!/usr/bin/python
import sys, os
from subprocess import check_output

PKG_PATH = '/handler/packages.txt'
HOST_PATH = '/host'
STDOUT_PATH = '%s/stdout2' % HOST_PATH
STDERR_PATH = '%s/stderr2' % HOST_PATH

def main():
    sys.stdout = open(STDOUT_PATH, 'w')
    sys.stderr = open(STDERR_PATH, 'w')

    if not os.path.exists(PKG_PATH):
        print('no packages to install')
        sys.stdout.flush()
        return

    with open(PKG_PATH) as fd:
        for line in fd:
            try:
                pkg = line.split(':')[1]
                if pkg != '':
                    check_output(['pip', 'install', '--index-url', 'http://128.104.222.169:9199/simple', '--trusted-host', '128.104.222.169', pkg])
            except Exception as e:
                print('failed to install %s with %s' % (pkg, e))
                sys.stdout.flush()


if __name__ == '__main__':
    main()
