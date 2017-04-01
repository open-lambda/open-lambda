#!/usr/bin/python
import traceback, json, sys, socket, os, importlib, pip
import rethinkdb
import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.netutil

import ns

HOST_PATH = '/host'
SOCK_PATH = '%s/ol.sock' % HOST_PATH
STDOUT_PATH = '%s/stdout' % HOST_PATH
STDERR_PATH = '%s/stderr' % HOST_PATH


PROCESSES_DEFAULT = 10
initialized = False
config = None
db_conn = None
installed = {}

# run after forking into sandbox
def init():
    global initialized, config, db_conn, lambda_func

    redirect()

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
    if not pkg in installed:
        if mirror:
            pip.main(['install', '-i', mirror, pkg_path])
        else:
            pip.main(['install', pkg_path])

# listen on sock file with Tornado
def lambda_server():
    server = tornado.httpserver.HTTPServer(tornado_app)
    socket = tornado.netutil.bind_unix_socket(SOCK_PATH)
    server.add_socket(socket)
    tornado.ioloop.IOLoop.instance().start()
    server.start(PROCESSES_DEFAULT)

# listen for fds to forkenter
def fdlisten(path):
    signal = "cache"
    r = -1
    # only child meant to serve ever escapes the loop
    while r != 0 or signal == "cache":
        sys.stdout.flush()
        if r == 0:
            redirect()

        pkgs = ns.fdlisten(path).split()
        # import packages into global scope
        for k, pkg in enumerate(pkgs):
            if k < len(pkgs)-1:
                split = pkg.split(':')
                try:
                    install(pkg[1])
                    globals()[pkg] = importlib.import_module(pkg)
                    print("importing %s" % pkg)
                except Exception as e:
                    print('failed to install %s with: %s' % (pkg[1], e))
            else:
                signal = pkg
                print("signal: %s" % signal)

        sys.stdout.flush()
        r = ns.forkenter()

    print("%s escaped" % r)
    sys.stdout.flush()
    init()
    lambda_server()

def redirect():
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
