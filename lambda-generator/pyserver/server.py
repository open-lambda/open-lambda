#!/usr/bin/python
import SimpleHTTPServer
import SocketServer
import logging
import cgi
import traceback, json, time, os, socket, struct
import lambda_func # assume submitted .py file is called lambda_func
import rethinkdb
import flask
app = flask.Flask(__name__)

PORT = 8080
initialized = False
config = None
db_conn = None

# run once per process
def init():
    global initialized, config, db_conn
    if initialized:
        return
    with open('config.json') as f:
        config = json.loads(f.read())
    if config.get('db', None) == 'rethinkdb':
        db_conn = rethinkdb.connect(get_default_gateway_linux(), 28015)
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
        event = json.loads(data)
        return json.dumps(lambda_func.handler(db_conn, event))
    except Exception:
        return (traceback.format_exc(), 500) # internal error

def main():
    # TODO(tyler): shouldn't be fixed at 10 (make dynamic, or maybe
    # use config)
    init()
    app.run(processes=10, host='0.0.0.0', port=PORT)

if __name__ == '__main__':
    main()
