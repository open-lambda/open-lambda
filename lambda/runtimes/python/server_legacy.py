'''
Python Runtime for Docker

Note: SOCK doesn't use this anymore (it uses server.py instead), but
this is still here because we haven't updated docker.go yet.
'''

#pylint: disable=invalid-name,line-too-long,global-statement

import os
import sys
import json
import argparse
import importlib
import traceback

import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.netutil

HOST_DIR = '/host'
PKGS_DIR = '/packages'
HANDLER_DIR = '/handler'

sys.path.append(PKGS_DIR)
sys.path.append(HANDLER_DIR)

FS_PATH = os.path.join(HOST_DIR, 'fs.sock')
SOCK_PATH = os.path.join(HOST_DIR, 'ol.sock')
STDOUT_PATH = os.path.join(HOST_DIR, 'stdout')
STDERR_PATH = os.path.join(HOST_DIR, 'stderr')
SERVER_PIPE_PATH = os.path.join(HOST_DIR, 'server_pipe')

PROCESSES_DEFAULT = 10
initialized = False

parser = argparse.ArgumentParser(description='Listen and serve cache requests or lambda invocations.')
parser.add_argument('--cache', action='store_true', default=False, help='Begin as a cache entry.')

# run after forking into sandbox
def init():
    global initialized, f
    if initialized:
        return

    # assume submitted .py file is /handler/f.py
    import f

    initialized = True

class SockFileHandler(tornado.web.RequestHandler):
    def post(self):
        try:
            data = self.request.body
            try :
                event = json.loads(data)
            except:
                self.set_status(400)
                self.write(f'bad POST data: "{data}"')
                return
            self.write(json.dumps(f.f(event)))
        except Exception:
            self.set_status(500) # internal error
            self.write(traceback.format_exc())

tornado_app = tornado.web.Application([
    (r".*", SockFileHandler),
])

# listen on sock file with Tornado
def lambda_server():
    init()
    server = tornado.httpserver.HTTPServer(tornado_app)
    socket = tornado.netutil.bind_unix_socket(SOCK_PATH)
    server.add_socket(socket)
    # notify worker server that we are ready through stdout
    # flush is necessary, and don't put it after tornado start; won't work
    with open(SERVER_PIPE_PATH, 'w', encoding='utf-8') as pipe:
        pipe.write('ready')
    tornado.ioloop.IOLoop.instance().start()
    server.start(PROCESSES_DEFAULT)

# listen for fds to forkenter
def cache_loop():
    import ns

    signal = "cache"
    r = -1
    count = 0
    # only child meant to serve ever escapes the loop
    while r != 0 or signal == "cache":
        if r == 0:
            print('RESET')
            flush()
            ns.reset()

        print('LISTENING')
        flush()
        data = ns.fdlisten(FS_PATH).split()
        flush()

        mods = data[:-1]
        signal = data[-1]

        ret_val = ns.forkenter()
        sys.stdout.flush()
        if ret_val == 0:
            redirect()
            # import modules
            for mod in mods:
                print(f'importing: {mod}')
                try:
                    globals()[mod] = importlib.import_module(mod)
                except Exception as err:
                    print(f'failed to import {mod} with: {err}')

            print(f'signal: {signal}')
            flush()

        print('')
        flush()

        count += 1

    print('SERVING HANDLERS')
    flush()
    lambda_server()

def flush():
    sys.stdout.flush()
    sys.stderr.flush()

def redirect():
    sys.stdout.close()
    sys.stderr.close()
    sys.stdout = open(STDOUT_PATH, 'w')
    sys.stderr = open(STDERR_PATH, 'w')

if __name__ == '__main__':
    args = parser.parse_args()
    redirect()

    if args.cache:
        cache_loop()
    else:
        lambda_server()
