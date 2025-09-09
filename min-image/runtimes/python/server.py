# pylint: disable=line-too-long,global-statement,invalid-name,broad-except

''' Python runtime for sock '''

# --- FIX 1: ADD 're' IMPORT ---
import os, sys, json, argparse, importlib, traceback, time, fcntl, array, socket, struct, re

sys.path.append("/usr/local/lib/python3.10/dist-packages")

import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.wsgi
import tornado.netutil

import ol

# --- FIX 2: INSERT THE MIDDLEWARE CLASS ---
class PrefixWSGI:
    """
    Mount a WSGI app at /run/<app>. Adjusts SCRIPT_NAME/PATH_INFO so Flask/Django
    routes work when accessed behind the /run/<app> prefix.
    If OL_APP_NAME is set, only that prefix is accepted; other prefixes 404.
    """
    _any_re = re.compile(r"^/run/[^/]+")

    def __init__(self, app, app_name=None):
        self.app = app
        self.app_name = app_name
        self._fixed_prefix = f"/run/{app_name}" if app_name else None

    def __call__(self, environ, start_response):
        path = environ.get("PATH_INFO") or "/"
        script = environ.get("SCRIPT_NAME", "")

        if self._fixed_prefix:
            prefix = self._fixed_prefix
            if path == prefix:
                rest = "/"
            elif path.startswith(prefix + "/"):
                rest = path[len(prefix):]
            else:
                start_response("404 Not Found", [("Content-Type", "text/plain")])
                return [b"Not Found"]
        else:
            m = self._any_re.match(path)
            if not m:
                start_response("404 Not Found", [("Content-Type", "text/plain")])
                return [b"Not Found"]
            prefix = m.group(0)
            rest = path[len(prefix):] or "/"

        environ["SCRIPT_NAME"] = script + prefix
        environ["PATH_INFO"] = rest
        environ.setdefault("HTTP_X_FORWARDWARDED_PREFIX", prefix)

        return self.app(environ, start_response)


file_sock_path = "/host/ol.sock"
file_sock = None
bootstrap_path = None

# --- FIX 3: UPGRADE THE web_server FUNCTION ---
def web_server():
    print(f"server.py: start web server on fd: {file_sock.fileno()}")
    sys.path.append('/handler')

    # FIX 3A: Add app/ to the path to find dependencies like werkzeug
    sys.path.append('/handler/app')

    import f

    class SockFileHandler(tornado.web.RequestHandler):
        # This original handler is kept for non-WSGI functions
        def handle_request(self):
            try:
                data = self.request.body
                try:
                    event = json.loads(data) if data else None
                except:
                    self.set_status(400)
                    self.write(f'bad request data: "{data}"')
                    return

                result = f.f(event) if event is not None else f.f({})
                self.write(json.dumps(result))
            except Exception:
                self.set_status(500)
                self.write(traceback.format_exc())
        
        def get(self): self.handle_request()
        def post(self): self.handle_request()
        def put(self): self.handle_request()
        def delete(self): self.handle_request()
        def patch(self): self.handle_request()
        def options(self): self.handle_request()

    # FIX 3B: Use the middleware to wrap the WSGI app
    if hasattr(f, "app"):
        # use WSGI entry (Flask/Django/etc), mounted under /run/<app>
        app_name = os.environ.get("OL_APP_NAME")  # optional: restrict to one app name
        mounted = PrefixWSGI(f.app, app_name=app_name)
        app = tornado.wsgi.WSGIContainer(mounted)
    else:
        # use original function entry
        app = tornado.web.Application([
            (".*", SockFileHandler),
        ])
    
    server = tornado.httpserver.HTTPServer(app)
    server.add_socket(file_sock)
    
    # FIX 3C: Send the "ready" signal to prevent deadlock
    print()
    sys.stdout.flush()
    
    tornado.ioloop.IOLoop.instance().start()
    server.start()


# --- NO CHANGES BELOW THIS LINE ---

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
            os.waitpid(pid, 0)
            client.sendall(struct.pack("I", pid))
            client.close()
        else:
            # child
            file_sock.close()
            file_sock = None
            os.fchdir(root_fd)
            os.chroot(".")
            os.close(root_fd)
            os.write(mem_cgroup_fd, str(os.getpid()).encode('utf-8'))
            os.close(mem_cgroup_fd)
            start_container()
            os._exit(1)


def start_container():
    global file_sock

    return_val = ol.unshare()
    assert return_val == 0

    file_sock = tornado.netutil.bind_unix_socket(file_sock_path)

    pid = os.fork()
    assert pid >= 0

    if pid > 0:
        os._exit(0)

    with open(bootstrap_path, encoding='utf-8') as f:
        code = f.read()
        try:
            exec(code)
        except Exception as _:
            print("Exception: " + traceback.format_exc())
            print("Problematic Python Code:\n" + code)

def main():
    global bootstrap_path

    if len(sys.argv) < 2:
        print("Expected execution: chroot <path_to_root_fs> python3 server.py <path_to_bootstrap.py> [cgroup-count] [enable-seccomp]")
        sys.exit(1)

    print('server.py: started new process with args: ' + " ".join(sys.argv))

    if len(sys.argv) < 3 or sys.argv[3] == 'true':
        return_code = ol.enable_seccomp()
        assert return_code >= 0
        print('seccomp enabled')

    bootstrap_path = sys.argv[1]
    cgroup_fds = 0
    if len(sys.argv) > 2:
        cgroup_fds = int(sys.argv[2])
    
    pid = str(os.getpid())
    for i in range(cgroup_fds):
        fd_id = 3 + i
        with os.fdopen(fd_id, "w") as file:
            file.write(pid)
            print(f'server.py: joined cgroup, close FD {fd_id}')

    start_container()


if __name__ == '__main__':
    main()
