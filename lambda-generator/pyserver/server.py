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

# source: http://stackoverflow.com/a/6556951
def get_default_gateway_linux():
    """Read the default gateway directly from /proc."""
    with open("/proc/net/route") as fh:
        for line in fh:
            fields = line.strip().split()
            if fields[1] != '00000000' or not int(fields[3], 16) & 2:
                continue

            return socket.inet_ntoa(struct.pack("<L", int(fields[2], 16)))

db_conn_cache = None

def db_conn():
    global db_conn_cache
    if db_conn_cache == None:
        db_conn_cache = rethinkdb.connect(get_default_gateway_linux(), 28015)
    return db_conn_cache

# catch everything
@app.route('/', defaults={'path': ''}, methods=['POST'])
@app.route('/<path:path>', methods=['POST'])
def flask_post(path):
    flask.request.get_data()
    data = flask.request.data
    try:
        event = json.loads(data)
    except:
        return 'could not parse ' + str(data)
    # handle req
    try:
        return json.dumps(lambda_func.handler(db_conn(), event))
    except Exception:
        return traceback.format_exc()

def main():
    # TODO(tyler): shouldn't be fixed at 10 (make dynamic, or maybe
    # use config)
    app.run(processes=10, host='0.0.0.0', port=PORT)

if __name__ == '__main__':
    main()
