#!/usr/bin/python
import traceback, json, socket, struct, os, sys
import rethinkdb
import flask

sys.path.append('/handler')
import lambda_func # assume submitted .py file is /handler/lambda_func

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
    config = json.loads(os.environ['ol.config'])
    if config.get('db', None) == 'rethinkdb':
        host = config.get('rethinkdb.host', 'localhost')
        port = config.get('rethinkdb.port', 28015)
        print 'Connect to %s:%d' % (host, port)
        db_conn = rethinkdb.connect(host, port)
    initialized = True

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
    config = json.loads(os.environ['ol.config'])
    print 'CONFIG: %s' % str(config)
    procs = config.get('processes', PROCESSES_DEFAULT)
    print 'Starting %d flask processes' % procs
    app.run(processes=procs, host='0.0.0.0', port=PORT)

if __name__ == '__main__':
    main()
