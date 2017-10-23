#!/usr/bin/python
import traceback, json, sys, socket, os, importlib, pip, hashlib, signal
import rethinkdb
import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.netutil
from subprocess import check_output

import ns

PKGS_PATH = '/packages'
HOST_PATH = '/host'

FS_PATH = '%s/fs.sock' % HOST_PATH
SOCK_PATH = '%s/ol.sock' % HOST_PATH

STDOUT_PATH = '%s/stdout' % HOST_PATH
STDERR_PATH = '%s/stderr' % HOST_PATH

global INDEX_HOST
global INDEX_PORT
MIRROR = False

PROCESSES_DEFAULT = 10
initialized = False
config = None
db_conn = None

# run after forking into sandbox
def init():
    global initialized, config, db_conn, lambda_func

    # assume submitted .py file is /handler/lambda_func.py
    sys.path.append('/handler')
    import lambda_func

    # need alternate config mechanism
    if False:
        config = json.loads(os.environ['ol.config'])
        if config.get('db', None) == 'rethinkdb':
            host = config.get('rethinkdb.host', 'localhost')
            port = config.get('rethinkdb.port', 28015)
            print 'Connect to %s:%d' % (host, port)
            db_conn = rethinkdb.connect(host, port)

class SockFileHandler(tornado.web.RequestHandler):
    def post(self):
        try:
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

def install(pkg):
    if MIRROR:
        check_output(' '.join(['pip', 'install', '--no-cache-dir', '--index-url', 'http://%s:%s/simple' % (INDEX_HOST, INDEX_PORT), '--trusted-host', INDEX_HOST, pkg]), shell=True)
    else:
        check_output(' '.join(['pip', 'install', pkg]), shell=True)

# listen on sock file with Tornado
def lambda_server():
    server = tornado.httpserver.HTTPServer(tornado_app)
    socket = tornado.netutil.bind_unix_socket(SOCK_PATH)
    server.add_socket(socket)
    tornado.ioloop.IOLoop.instance().start()
    server.start(PROCESSES_DEFAULT)

# create symbolic links from install cache to dist-packages, return if success
def create_link(pkg):
    hsh = hashlib.sha256(pkg).hexdigest()
    # assume no version (e.g. "==1.2.1")
    pkgdir = '%s/%s/%s/%s/%s' % (PKGS_PATH, hsh[:2], hsh[2:4], hsh[4:], pkg)
    if os.path.exists(pkgdir):
        for name in os.listdir(pkgdir):
            source = pkgdir + '/' + name
            link_name = '/host/pip/%s' % name
            if os.path.exists(link_name):
                continue # should we report this?
            os.symlink(source, link_name)
        return True
    return False

# listen for fds to forkenter
def fdlisten():
    signal = "cache"
    r = -1
    count = 0
    # only child meant to serve ever escapes the loop
    while r != 0 or signal == "cache":
        if r == 0:
            print('RESET')
            sys.stdout.flush()
            ns.reset()

        print('LISTENING')
        sys.stdout.flush()
        data = ns.fdlisten(FS_PATH).split()

        r = ns.forkenter()
        if r == 0:
            redirect()

            mods = []
            pkgs = []
            for info in data[:-1]:
                split = info.split(':')
                mods.append(split[0])
                if split[1] != '':
                    pkgs.append(split[1])

            # use install cache
            remains = []
            for pkg in pkgs:
                if create_link(pkg):
                    print('using install cache: %s' % pkg)
                else:
                    remains.append(pkg)
            pkgs = remains

            # install from pip mirror
            for pkg in pkgs:
                print('installing: %s' % pkg)
                try:
                    install(pkg)
                except Exception as e:
                    print('install %s failed with: %s' % (split[1], e))
            
	    sys.path.append('/host/pip')
            # import modules
            for mod in mods:
                print('importing: %s' % mod)
                try:
                    globals()[mod] = importlib.import_module(mod)
                except Exception as e:
                    print('failed to import %s with: %s' % (mod, e))

            signal = data[-1]
            print('signal: %s' % signal)

            sys.stdout.flush()
            sys.stderr.flush()

        print('')
        sys.stdout.flush()

        count += 1

    print('SERVING HANDLERS')
    sys.stdout.flush()
    init()
    lambda_server()

def redirect():
    sys.stdout.close()
    sys.stderr.close()
    sys.stdout = open(STDOUT_PATH, 'w')
    sys.stderr = open(STDERR_PATH, 'w')

if __name__ == '__main__':
    global INDEX_HOST
    global INDEX_PORT
    sys.stdout = open(STDOUT_PATH, 'w')
    sys.stderr = open(STDERR_PATH, 'w')

    if len(sys.argv) != 1 and len(sys.argv) != 3:
        print('Usage: python %s or python %s <index_host> <index_sock>' % (sys.argv[0], sys.argv[0]))
        sys.exit(1)

    try:
        INDEX_HOST = sys.argv[1]
        INDEX_PORT = sys.argv[2]
        MIRROR = True
    except:
        pass

    fdlisten()
