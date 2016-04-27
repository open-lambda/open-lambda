#!/usr/bin/python
import traceback, json, socket, struct, os, sys
import lambda_func # assume submitted .py file is called lambda_func
import rethinkdb
import flask
app = flask.Flask(__name__)

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
    with open('config.json') as f:
        config = json.loads(f.read())
    if config.get('db', None) == 'rethinkdb':
        addr = os.environ.get('RETHINKDB_PORT_28015_TCP', None)
        if addr != None:
            host, port = addr.split('//')[-1].split(':')
        else:
            host, port = get_default_gateway_linux(), '28015'
        db_conn = rethinkdb.connect(host, int(port))
    initialized = True

# source: http://stackoverflow.com/a/6556951
def get_default_gateway_linux():
    """Read the default gateway directly from /proc."""
    with open("/proc/net/route") as fh:
        for line in fh:
            fields = line.strip().split()
            if fields[1] != '00000000' or not int(fields[3], 16) & 2:
                continue

            return socket.inet_ntoa(struct.pack("<L", int(fields[2], 16)))

# catch everything
@app.route('/', defaults={'path': ''}, methods=['POST'])
@app.route('/<path:path>', methods=['POST'])
def flask_post(path):
    try:
        init()
        flask.request.get_data()
        data = flask.request.data
        try :
            event = json.loads(data)
        except:
            return ('bad POST data: "%s"'%str(data), 400)
        return json.dumps(lambda_func.handler(db_conn, event))
    except Exception:
        return (traceback.format_exc(), 500) # internal error

def main():
    with open('config.json') as f:
        config = json.loads(f.read())
    procs = config.get('processes', PROCESSES_DEFAULT)
    print 'Starting %d flask processes' % procs
    app.run(processes=procs, host='0.0.0.0', port=PORT)

if __name__ == '__main__':
    main()
