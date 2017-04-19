#!/usr/bin/python
import traceback, json, sys, socket, os, importlib, pip
import rethinkdb
import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.netutil
from subprocess import check_output

import ns

PKGS_PATH = '/packages'
HOST_PATH = '/host'
SOCK_PATH = '%s/ol.sock' % HOST_PATH
STDOUT_PATH = '%s/stdout' % HOST_PATH
STDERR_PATH = '%s/stderr' % HOST_PATH

INDEX_HOST = '128.104.222.179'
INDEX_PORT = '9199'

PROCESSES_DEFAULT = 10
initialized = False
config = None
db_conn = None
installed = {}

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
    global installed, mirror
    if pkg in installed:
        return

    #if mirror:
        #ret = pip.main(['install', '-i', mirror, pkg])
    check_output(['pip', 'install', '--index-url', 'http://%s:%s/simple' % (INDEX_HOST, INDEX_PORT), '--trusted-host', INDEX_HOST, pkg])
    #else:
        #ret = pip.main(['install', pkg])
     #   check_output(['pip', 'install', pkg])

    installed[pkg] = True

# listen on sock file with Tornado
def lambda_server():
    server = tornado.httpserver.HTTPServer(tornado_app)
    socket = tornado.netutil.bind_unix_socket(SOCK_PATH)
    server.add_socket(socket)
    tornado.ioloop.IOLoop.instance().start()
    server.start(PROCESSES_DEFAULT)

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

# listen for fds to forkenter
def fdlisten(path):
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
        data = ns.fdlisten(path).split()

        r = ns.forkenter()
        if r == 0:
            redirect()

            imps = []
            pkgs = []
            for info in data[:-1]:
                split = info.split(':')
                imps.append(split[0])
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
            
            # import modules
            for imp in imps:
                print('importing: %s' % imp)
                try:
                    globals()[imp] = importlib.import_module(imp)
                except Exception as e:
                    print('failed to import %s with: %s' % (imp, e))

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
    if len(sys.argv) < 2 or len(sys.argv) > 3:
        print('Usage: python %s <sock> or python %s <sock> <pip_mirror>' % (sys.argv[0], sys.argv[0]))
        sys.exit(1)

    try:
        mirror = sys.argv[2]
    except:
        mirror = None

    fdlisten(sys.argv[1])
