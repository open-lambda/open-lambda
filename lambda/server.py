#!/usr/bin/python
import traceback, json, sys, socket, os, time, hashlib
import rethinkdb
import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.netutil
from subprocess import check_output

HOST_PATH = '/host'
SOCK_PATH = '%s/ol.sock' % HOST_PATH
STDOUT_PATH = '%s/stdout' % HOST_PATH
STDERR_PATH = '%s/stderr' % HOST_PATH

PKGS_PATH = '/packages'
PKG_PATH = '/handler/packages.txt'

global INDEX_HOST
global INDEX_PORT
MIRROR = False

PROCESSES_DEFAULT = 10
initialized = False
config = None
db_conn = None

# run once per process
def init():
    global initialized, config, db_conn, lambda_func
    if initialized:
        return
    
    config = json.loads(os.environ['ol.config'])
    if config != None and config.get('db', None) == 'rethinkdb':
        host = config.get('rethinkdb.host', 'localhost')
        port = config.get('rethinkdb.port', 28015)
        print 'Connect to %s:%d' % (host, port)
        db_conn = rethinkdb.connect(host, port)

    sys.path.append('/handler')
    import lambda_func # assume submitted .py file is /handler/lambda_func.py

    initialized = True

# create symbolic links from install cache to dist-packages, return if success
def create_link(pkg):
    hsh = hashlib.sha256(pkg).hexdigest()
    # assume no version (e.g. "==1.2.1")
    pkgdir = '%s/%s/%s/%s/%s' % (PKGS_PATH, hsh[:2], hsh[2:4], hsh[4:], pkg)
    if os.path.exists(pkgdir):
        for name in os.listdir(pkgdir):
            source = pkgdir + '/' + name
            link_name = '/usr/lib/python2.7/dist-packages/' + name
            if os.path.exists(link_name):
                print('link failed, path already exists: %s' % link_name)
                sys.stdout.flush()
                continue
            os.symlink(source, link_name)
        return True
    return False

def install(pkg):
    if MIRROR:
        check_output(['pip', 'install', '--no-cache-dir', '--index-url', 'http://%s:%s/simple' % (INDEX_HOST, INDEX_PORT), '--trusted-host', INDEX_HOST, pkg])
    else:
        check_output(['pip', 'install', pkg])

def do_installs():
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
                        install(pkg)
                        sys.stdout.flush()
                    except Exception as e:
                        print('failed to install %s with %s' % (pkg, e))
                        sys.stdout.flush()

class SockFileHandler(tornado.web.RequestHandler):
    def post(self):
        try:
            init()
            data = self.request.body
            try :
                event = json.loads(data)
            except:
                self.set_status(400)
                self.write('bad POST data: "%s"'%str(data))
                return
            self.write(json.dumps(lambda_func.handler(db_conn, event)))
        except Exception:
            self.set_status(500) # internal error
            self.write(traceback.format_exc())

tornado_app = tornado.web.Application([
    (r".*", SockFileHandler),
])

# listen on sock file with Tornado
def lambda_server():
    do_installs()
    server = tornado.httpserver.HTTPServer(tornado_app)
    socket = tornado.netutil.bind_unix_socket(SOCK_PATH)
    server.add_socket(socket)
    tornado.ioloop.IOLoop.instance().start()
    server.start(PROCESSES_DEFAULT)

if __name__ == '__main__':
    global INDEX_HOST
    global INDEX_PORT
    try:
        sys.stdout = open(STDOUT_PATH, 'w')
        sys.stderr = open(STDERR_PATH, 'w')
    except Exception as e:
        with open('/ERROR', 'w') as fd:
            fd.write('failed to open stdout/stderr with: %s\n' % e)
            sys.exit(1)

    if len(sys.argv) != 1 and len(sys.argv) != 3:
        print('Usage: python %s or python %s <index_host> <index_sock>' % (sys.argv[0], sys.argv[0]))
        sys.exit(1)

    try:
        INDEX_HOST = sys.argv[1]
        INDEX_PORT = sys.argv[2]
        MIRROR = True
    except:
        pass

    curr = 0.0
    while not os.path.exists(PKGS_PATH):
        time.sleep(0.005)
        curr += 0.005
        if curr > 1.0:
            print('packages.txt missing (path=%s)' % PKGS_PATH)
            sys.stdout.flush()
            sys.exit(1)

    lambda_server()
