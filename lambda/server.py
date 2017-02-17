#!/usr/bin/python
import ns
import traceback, json, socket, struct, os, sys, socket, threading
import rethinkdb
import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.netutil
import time

flask_app = flask.Flask(__name__)
PROCESSES_DEFAULT = 10
PORT = 8080
initialized = False
config = None
db_conn = None

# run once per process
def init():
    global initialized, config, db_conn
    if initialized:
        return
    sys.stdout = sys.stderr # flask supresses stdout :(
    if config.get('db', None) == 'rethinkdb':
        host = config.get('rethinkdb.host', 'localhost')
        port = config.get('rethinkdb.port', 28015)
        print 'Connect to %s:%d' % (host, port)
        db_conn = rethinkdb.connect(host, port)
    initialized = True

# catch everything
@flask_app.route('/', defaults={'path': ''}, methods=['POST'])
@flask_app.route('/<path:path>', methods=['POST'])
def flask_post(path):
    try:
        init()
        data = flask.request.get_data()
        try:
            event = json.loads(data)
        except Exception:
            return ('bad POST data: "%s"' % str(data), 400)
        return json.dumps(lambda_func.handler(db_conn, event))
    except Exception as e:
        print(e)
        return (traceback.format_exc(), 500) # internal error

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

def start_container(conf):
    sys.path.append('/handler')
    global lambda_func, config

    import lambda_func # assume submitted .py file is /handler/lambda_func
    config = conf

    if 'sock_file' in config:
        #f.write("listening socket\n")
        # listen on sock file with Tornado
        server = tornado.httpserver.HTTPServer(tornado_app)
        socket = tornado.netutil.bind_unix_socket('/host/' + config['sock_file'])
        server.add_socket(socket)
        tornado.ioloop.IOLoop.instance().start()
    else:
        #f.write("listening flask\n")
        # listen on port with Flask
        procs = config.get('processes', PROCESSES_DEFAULT)
        flask_app.run(processes=procs, host='0.0.0.0', port=PORT)

def listen(path):
    args = ""
    with open(path) as fifo:
        while True:
            data = fifo.read()
            if len(data) == 0:
                break
            args += data
    return args

def main():
    sys.stdout = sys.stderr
    if len(sys.argv) < 2:
        print("Usage: %s <fifo>" % sys.argv[0])
        sys.exit(1)

    fifo = os.path.abspath(sys.argv[1])

    #f = open('/tmp/log', 'w')
    while True:
        print("host listening")
        #f.write("host listening\n")
        pid, conf = listen(fifo).split(None, 1)
        conf = json.loads(conf)
        print("pid: %s\nconf: %s\n" % (pid, conf))
        #f.write("pid: %s\nconf: %s\n" % (pid, conf))
        
        r = ns.forkenter(pid)
        # child escape
        if r == 0:
            print("forkentered")
            #f.write("forkentered\n")
            break

    try:
        start_container(conf)
    except Exception as e:
        print(e)

if __name__ == '__main__':
    main()
