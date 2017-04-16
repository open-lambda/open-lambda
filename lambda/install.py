#!/usr/bin/python
import sys, os
from subprocess import check_output

PKG_PATH = '/handler/packages.txt'
PKGS_PATH = '/packages'
HOST_PATH = '/host'
STDOUT_PATH = '%s/stdout' % HOST_PATH
STDERR_PATH = '%s/stderr' % HOST_PATH

INDEX_HOST = '172.17.0.1'
INDEX_PORT = '9199'

# create symbolic links from install cache to dist-packages, return if success
def create_link(pkg):
    # assume no version (e.g. "==1.2.1")
    pkgdir = '%s/%s' % (PKGS_PATH, pkg)
    if os.path.exists(pkgdir):
        for name in os.listdir(pkgdir):
            source = pkgdir + '/' + name
            link_name = '/usr/lib/python2.7/dist-packages/' + name
            if os.path.exists(link_name):
                continue # should we report this?
            os.symlink(source, link_name)
        return True
    return False

def main():
    sys.stdout = open(STDOUT_PATH, 'w')
    sys.stderr = open(STDERR_PATH, 'w')

    if not os.path.exists(PKG_PATH):
        print('no packages to install')
        return

    with open(PKG_PATH) as fd:
        for line in fd:
            pkg = line.strip().split(':')[1]
            if pkg != '':
                if create_link(pkg):
                    print('using install cache: %s' % pkg)
                else:
                    raise Exception('should already installed using cache!')
                    print('installing: %s' % pkg)
                    try:
                        check_output(['pip', 'install', '--index-url', 'http://%s:%s/simple' % (INDEX_HOST, INDEX_PORT), '--trusted-host', INDEX_HOST, pkg])
                    except Exception as e:
                        print('failed to install %s with %s' % (pkg, e))

if __name__ == '__main__':
    main()
