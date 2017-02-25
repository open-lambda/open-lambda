#!/usr/bin/python
import traceback, json, struct, os, sys, socket
import rethinkdb
import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.netutil
from subprocess import check_output


SOCKET_PATH = "/host/ol.sock"
PROCESSES_DEFAULT = 10
initialized = False
config = None
db_conn = None

# run once per process
def init():
    global initialized, config, db_conn, lambda_func
    if initialized:
        return

    sys.stdout = sys.stderr # flask supresses stdout :(
    #config = json.loads(os.environ['ol.config'])
    #if config.get('db', None) == 'rethinkdb':
    #    host = config.get('rethinkdb.host', 'localhost')
    #    port = config.get('rethinkdb.port', 28015)
    #    print 'Connect to %s:%d' % (host, port)
    #    db_conn = rethinkdb.connect(host, port)

    sys.path.append('/handler')
    import lambda_func # assume submitted .py file is /handler/lambda_func.py

    initialized = True

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
def listen_socket():
    server = tornado.httpserver.HTTPServer(tornado_app)
    socket = tornado.netutil.bind_unix_socket(SOCKET_PATH)
    server.add_socket(socket)
    tornado.ioloop.IOLoop.instance().start()

def listen_fifo(fifo):
    args = ""
    while True: #TODO
        data = fifo.read()
        if len(data) == 0:
            break
        args += data

    return args

# wait for NS to enter, listen on sock file
def fork(path):
    with open(path) as fifo:
        while True:
            pid = listen_fifo(fifo)

            r = ns.forkenter(pid)
            if r == 0:
                break # child escapes

    listen_socket()

if __name__ == '__main__':
    if len(sys.argv) == 1:
        listen_socket()
    elif len(sys.argv) == 2:
        fork(os.path.abspath(sys.argv[1]))
    else:
        print('Usage (nofork): python %s' % sys.argv[0])
        print('Usage (fork): python %s --fork <fifo>' % sys.argv[0])
        sys.exit(1)
