#!/usr/bin/python
import traceback, json, sys, socket, os, importlib
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

# run after forking into sandbox
def init():
    global initialized, config, db_conn, lambda_func

    sys.stdout = open(STDOUT_PATH, 'w')
    sys.stderr = open(STDERR_PATH, 'w')

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

# listen on sock file with Tornado
def lambda_server():
    server = tornado.httpserver.HTTPServer(tornado_app)
    socket = tornado.netutil.bind_unix_socket(SOCK_PATH)
    server.add_socket(socket)
    tornado.ioloop.IOLoop.instance().start()
    server.start(PROCESSES_DEFAULT)

# listen for fds to forkenter
def fdlisten(path):
    r = -1
    # only child ever escapes the loop
    while r != 0:
        pkgs = ns.fdlisten(path)

        # import packages into global scope
        for pkg in pkgs.split():
            globals()[pkg] = importlib.import_module(pkg)

        r = ns.forkenter()

    init()
    lambda_server()

if __name__ == '__main__':
    if len(sys.argv) != 2:
        print('Usage: python %s <sock>' % sys.argv[0])
        sys.exit(1)

    fdlisten(sys.argv[1])
