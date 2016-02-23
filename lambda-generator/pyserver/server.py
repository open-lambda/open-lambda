#!/usr/bin/python
import SimpleHTTPServer
import SocketServer
import logging
import cgi
import traceback, json, time, os
import lambda_func # assume submitted .py file is called lambda_func

PORT = 8080
class ServerHandler(SimpleHTTPServer.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kvargs):
        SimpleHTTPServer.SimpleHTTPRequestHandler.__init__(self, *args, **kvargs)

    def do_GET(self):
        pass

    def do_POST(self):
        length = int(self.headers.getheader('content-length'))
        event = json.loads(self.rfile.read(length))
        result = lambda_func.handler(event)

        self.send_response(200) # OK
        self.send_header('Content-type', 'text/html')
        self.end_headers()
        self.wfile.write(json.dumps(result))

def main():
    httpd = SocketServer.TCPServer(("", PORT), ServerHandler)
    print "serving at port", PORT
    httpd.serve_forever()

if __name__ == '__main__':
    main()
