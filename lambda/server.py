#!/usr/bin/python
import ns
import traceback, json, socket, struct, os, sys, socket, threading
import rethinkdb
import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.netutil
import time

PROCESSES_DEFAULT = 10 # TODO: IS TORNADO MULTI-THREADED (processed) BY DEFAULT?
PORT = 8080
initialized = False
config = None
db_conn = None

tornado_app = tornado.web.Application([
    (r".*", SockFileHandler),
])

# run once per process
def init():
    global initialized, config, db_conn
    if initialized:
        return

    if config.get('db', None) == 'rethinkdb':
        host = config.get('rethinkdb.host', 'localhost')
        port = config.get('rethinkdb.port', 28015)
        print 'Connect to %s:%d' % (host, port)
        db_conn = rethinkdb.connect(host, port)
    initialized = True

    # assume user handler code is /handler/lambda_func.py
    sys.path.append('/handler')
    import lambda_func 

class SockFileHandler(tornado.web.RequestHandler):
    # POST header for actual requests
    def post(self):
        try:
            init()
            data = self.request.body
            try:
                event = json.loads(data)
            except:
                self.set_status(400)
                self.write('bad POST data: "%s"' % str(data))
                return

            self.write(json.dumps(lambda_func.handler(db_conn, event)))

        except Exception:
            self.set_status(500) # internal error
            self.write(traceback.format_exc())

    # GET header for forkenter
    def get(self):
        try:
            data = self.request.body
            try:
                forkconf = json.loads(data)

            except:
                self.set_status(400)
                self.write('malformed forkenter request: "%s"' % str(data))
                return

            ns.forkenter(forkconf['pid'])

        except Exception:
            self.set_status(500)
            self.write('failed to forkenter with request: "%s"' % str(data))
            self.write(traceback.format_exc())


def listen():
    server = tornado.httpserver.HTTPServer(tornado_app)
    socket = tornado.netutil.bind_unix_socket('/host/' + config['sock_file'])
    server.add_socket(socket)
    tornado.ioloop.IOLoop.instance().start()

def main():
    try:
        listen()
    except Exception as e:
        print(e)

if __name__ == '__main__':
    main()
