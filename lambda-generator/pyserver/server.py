#!/usr/bin/python
import SimpleHTTPServer
import SocketServer
import logging
import cgi
import traceback, json, time, os, socket, struct
import lambda_func # assume submitted .py file is called lambda_func
import rethinkdb

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

class ServerHandler(SimpleHTTPServer.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kvargs):
        # gateway will refer to the Docker host on which this container runs
        self.db_conn = rethinkdb.connect(get_default_gateway_linux(), 28015)

        # do_POST is called by SimpleHTTPRequestHandler.__init__, so
        # do this after any other init
        SimpleHTTPServer.SimpleHTTPRequestHandler.__init__(self, *args, **kvargs)

    def send_result(self, result):
        self.send_response(200) # OK
        self.send_header('Content-type', 'text/json')
        self.end_headers()
        self.wfile.write(json.dumps(result))

    def do_GET(self):
        pass

    def do_POST(self):
        length = int(self.headers.getheader('content-length'))
        # parse req
        data = self.rfile.read(length)
        try :
            event = json.loads(data)
        except:
            self.send_result('could not parse ' + str(data))
            return
        # handle req
        try :
            result = lambda_func.handler(self.db_conn, event)
        except Exception:
            self.send_result(traceback.format_exc())
            return
        # respond
        self.send_result(result)

def main():
    httpd = SocketServer.TCPServer(("", PORT), ServerHandler)
    print "serving at port", PORT
    httpd.serve_forever()

if __name__ == '__main__':
    main()
