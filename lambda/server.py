#!/usr/bin/python
import ns
import traceback, json, socket, struct, os, sys, socket
import rethinkdb
import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.netutil
import time

PROCESSES_DEFAULT = 10 # TODO: IS TORNADO MULTI-THREADED (processed) BY DEFAULT?
db_conn = None

class ChildHandler(tornado.web.RequestHandler):
    # POST data for actual requests
    def post(self):
        try:
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

class ParentHandler(tornado.web.RequestHandler):
    # POST nspid and new sockfile to listen on
    def post(self):
        try:
            data = self.request.body
            try:
                forkconf = json.loads(data)

            except:
                self.set_status(400)
                self.write('malformed forkenter request: "%s"' % str(data))
                return

            ns.forkenter(forkconf['nspid'])

            child_init(forkconf['db'])
            child_listen(forkconf['sock_file'])

        except Exception:
            self.set_status(500)
            self.write('failed to forkenter with request: "%s"' % str(data))
            self.write(traceback.format_exc())

def child_init(db):
    global db_conn

    if db == 'rethinkdb':
        host = config.get('rethinkdb.host', 'localhost')
        port = config.get('rethinkdb.port', 28015)
        print 'Connect to %s:%d' % (host, port)
        db_conn = rethinkdb.connect(host, port)

    # assume user handler code is /handler/lambda_func.py
    sys.path.append('/handler')
    import lambda_func 

child_app = tornado.web.Application([
    (r".*", ChildHandler),
])

def child_listen(sock_file):
    tornado.ioloop.IOLoop.instance().stop()

    server = tornado.httpserver.HTTPServer(child_app)
    socket = tornado.netutil.bind_unix_socket('/host/%s' % sock_file)
    server.add_socket(socket)
    tornado.ioloop.IOLoop.instance().start()

    return

parent_app = tornado.web.Application([
    (r".*", ParentHandler),
])

def parent_listen(sock_file):
    server = tornado.httpserver.HTTPServer(parent_app)
    socket = tornado.netutil.bind_unix_socket('/host/%s' % sock_file) # should probably be in different dir?
    server.add_socket(socket)
    tornado.ioloop.IOLoop.instance().start()

    return

def main():
    if len(sys.argv) != 2:
        print('Usage: %s <sock_file>' % sys.argv[0])
        sys.exit(1)

    try:
        parent_listen(sys.argv[1])
    except Exception as e:
        print('Failed to listen on %s with:\n%s' % (sys.argv[1], e))

if __name__ == '__main__':
    main()
