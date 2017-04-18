#!/usr/bin/python
import sys, os
from subprocess import check_output

PKG_PATH = '/handler/packages.txt'
PKGS_PATH = '/packages'
HOST_PATH = '/host'
STDOUT_PATH = '%s/stdout2' % HOST_PATH
STDERR_PATH = '%s/stderr2' % HOST_PATH

INDEX_HOST = '128.104.222.169'
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
        sys.stdout.flush()
        return

    with open(PKG_PATH) as fd:
        for line in fd:
            pkg = line.strip().split(':')[1]
            if pkg != '':
                if create_link(pkg):
                    print('using install cache: %s' % pkg)
                    sys.stdout.flush()
                else:
                    print('installing: %s' % pkg)
                    sys.stdout.flush()
                    try:
                        print(check_output(['pip', 'install', '--index-url', 'http://%s:%s/simple' % (INDEX_HOST, INDEX_PORT), '--trusted-host', INDEX_HOST, pkg]))
                        sys.stdout.flush()
                    except Exception as e:
                        print('failed to install %s with %s' % (pkg, e))
                        sys.stdout.flush()

if __name__ == '__main__':
    main()
