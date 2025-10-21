# pylint: disable=line-too-long,global-statement,invalid-name,broad-except

import os
import sys
import json
import argparse
import importlib
import traceback
import socket
import struct

sys.path.append("/usr/local/lib/python3.10/dist-packages")

import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.netutil
import tornado.wsgi

# ======== Global Variables ========

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

# ======== Utility Functions ========

def flush():
    sys.stdout.flush()
    sys.stderr.flush()


def redirect():
    sys.stdout.close()
    sys.stderr.close()
    sys.stdout = open(STDOUT_PATH, 'w')
    sys.stderr = open(STDERR_PATH, 'w')


# ======== Tornado Logic (for both Sock and Docker) ========

def create_tornado_web_server(handler_module):    
    class SockFileHandler(tornado.web.RequestHandler):
        # TODO: we should consider how are the different requests used in the context of different applications and functions
        # and consider what does the validations should look like for example, should we allow POST requests with no payload etc.
        def handle_request(self):
            try:
                data = self.request.body
                try:
                    event = json.loads(data) if data else None
                except Exception:
                    self.set_status(400) # bad request
                    self.write(f'Bad request data: "{data}"')
                    return

                result = handler_module.f(event) if event is not None else handler_module.f({})
                self.write(json.dumps(result)) # return result as JSON
            except Exception:
                self.set_status(500) # internal server errors
                self.write(traceback.format_exc()) # include traceback in response

        # define HTTP methods
        def get(self): self.handle_request()
        def post(self): self.handle_request()
        def put(self): self.handle_request()
        def delete(self): self.handle_request()
        def patch(self): self.handle_request()
        def options(self): self.handle_request()

    if hasattr(handler_module, "app"):
        def path_wrapper(environ, start_response):
            path = environ.get("PATH_INFO", "")

            # split path to get individual components
            parts = path.split("/")

            # set new environment path
            # `/run/<func-name>/a/b/c` -> `/a/b/c`
            environ["PATH_INFO"] = '/' + '/'.join(parts[3:])

            # set the root of the application
            environ["SCRIPT_NAME"] = '/run/' + parts[2]

            return handler_module.app(environ, start_response)
        
        # use WSGI entry calling wrapper to strip /run/<func-name> from path
        app = tornado.wsgi.WSGIContainer(path_wrapper)
    else:
        # use function entry
        app = tornado.web.Application([(".*", SockFileHandler)])
    return app


# ======== Docker Environment ========

def web_server_docker():
    global initialized, f
    if initialized:
        return
    
    # assume submitted .py file is /handler/f.py
    import f

    initialized = True

    app = create_tornado_web_server(f)

    server = tornado.httpserver.HTTPServer(app)
    socket = tornado.netutil.bind_unix_socket(SOCK_PATH)
    server.add_socket(socket)
    # notify worker server that we are ready through stdout
    # flush is necessary, and don't put it after tornado start; won't work
    with open(SERVER_PIPE_PATH, 'w', encoding='utf-8') as pipe:
        pipe.write('ready')
    tornado.ioloop.IOLoop.instance().start()
    server.start(PROCESSES_DEFAULT)


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
                    print(f'failed to import {mod}: {err}')

            print(f'signal: {signal}')
            flush()

        print('')
        flush()

        count += 1

    print('SERVING HANDLERS')
    flush()
    web_server_docker()


# ======== Sock Environment ========

# called by bootstrap file
def web_server():
    print(f"server.py: start web server on fd: {file_sock.fileno()}")

    # TODO: as a safeguard, we should add a mechanism so that the
    # import doesn't happen until the cgroup move completes, so that a
    # malicious child cannot eat up Zygote resources
    import f

    app = create_tornado_web_server(f)
    server = tornado.httpserver.HTTPServer(app)
    server.add_socket(file_sock)
    tornado.ioloop.IOLoop.instance().start()
    server.start()

# called by bootstrap file
def fork_server():
    global file_sock

    file_sock.setblocking(True)
    print(f"server.py: start fork server on fd: {file_sock.fileno()}")

    while True:
        client, _info = file_sock.accept()
        _, fds, _, _ = socket.recv_fds(client, 8, 2)
        root_fd, mem_cgroup_fd = fds

        pid = os.fork()

        if pid:
            # parent
            os.close(root_fd)
            os.close(mem_cgroup_fd)

            # the child opens the new ol.sock, forks the grandchild
            # (which will actually do the serving), then exits.  Thus,
            # by waiting for the child, we can be sure ol.sock exists
            # before we respond to the client that sent us the fork
            # request with the root FD.  This means the client doesn't
            # need to poll for ol.sock existence, because it is
            # guaranteed to exist.
            os.waitpid(pid, 0)
            client.sendall(struct.pack("I", pid))
            client.close()

        else:
            # child
            file_sock.close()
            file_sock = None

            # chroot
            os.fchdir(root_fd)
            os.chroot(".")
            os.close(root_fd)

            # mem cgroup
            os.write(mem_cgroup_fd, str(os.getpid()).encode('utf-8'))
            os.close(mem_cgroup_fd)

            # child
            start_sock_container()
            os._exit(1) # only reachable if program unnexpectedly returns


def start_sock_container():
    '''
    1. this assumes chroot has taken us to the location where the
        container should start.
    2. it launches the container code by running whatever is in the
        bootstrap file (from argv)
    '''

    global file_sock

    # TODO: if we can get rid of this, we can get rid of the ns module
    return_val = ol.unshare()
    assert return_val == 0

    # we open a new .sock file in the child, before starting the grand
    # child, which will actually use it.  This is so that the parent
    # can know that once the child exits, it is safe to start sending
    # messages to the sock file.
    file_sock = tornado.netutil.bind_unix_socket(SOCK_PATH)

    pid = os.fork()
    assert pid >= 0

    if pid > 0:
        # orphan the new process by exiting parent.  The parent
        # process is in a weird state because unshare only partially
        # works for the process that calls it.
        os._exit(0)

    with open(bootstrap_path, encoding='utf-8') as f:
        # this code can be whatever OL decides, but it will probably do the following:
        # 1. some imports
        # 2. call either web_server or fork_server
        code = f.read()
        try:
            exec(code)
        except Exception as _:
            print("Exception: " + traceback.format_exc())
            print("Problematic Python Code:\n" + code)


# ======== Main Entry Point ========

def main():
    '''
    Expected invocation for OL SOCK environment:
        python3 server.py --env sock --bootstrap <path-to-bootstrap.py> [--cgroup-count N] [--enable-seccomp]

    Expected invocation for OL DOCKER environment:
        python3 server.py --env docker [--cache] 
    '''
    # parse arguments
    parser = argparse.ArgumentParser(description="OpenLambda server")
    parser.add_argument("--env", choices=["sock", "docker"], default="sock", help="Runtime environment")
    parser.add_argument("--bootstrap", help="Path to bootstrap.py (for sock mode)")
    parser.add_argument('--cache', action='store_true', default=False, help='Begin as a cache entry.')
    parser.add_argument("--cgroup-count", type=int, default=0, help="Number of FDs (starting at 3) that refer to /sys/fs/cgroup/..../cgroup.procs files")
    parser.add_argument("--enable-seccomp", action="store_true", default=True)
    args = parser.parse_args()

    # run according to environment
    if args.env == "sock":
        print("Running in SOCK environment")

        global ol
        import ol

        # enable seccomp unless disabled
        if args.enable_seccomp: # default: True
            return_code = ol.enable_seccomp()
            assert return_code >= 0
            print("seccomp enabled")

        # set up bootstrap path
        global bootstrap_path
        bootstrap_path = args.bootstrap
        if not bootstrap_path:
            print("Error: --bootstrap is required in sock mode.")
            sys.exit(1)

        # set up cgroups
        cgroup_fds = args.cgroup_count # default: 0
        pid = str(os.getpid())
        for i in range(cgroup_fds):
            # golang guarantees extras start at 3: https://golang.org/pkg/os/exec/#Cmd
            fd_id = 3 + i
            with os.fdopen(fd_id, "w") as file:
                file.write(pid)
                print(f'server.py: joined cgroup, close FD {fd_id}')
        
        start_sock_container()

    elif args.env == "docker":
        redirect() # redirect stdout/stderr to host paths

        print("Running in Docker environment...")

        if args.cache:
            cache_loop()
        else:
            web_server_docker()

    else:
        print(f"Unknown environment: {args.env}")
        sys.exit(1)

if __name__ == "__main__":
    main()
