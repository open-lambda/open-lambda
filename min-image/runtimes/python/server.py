# pylint: disable=line-too-long,global-statement,invalid-name,broad-except

''' Python runtime for sock '''

import os, sys, json, argparse, importlib, traceback, time, fcntl, array, socket, struct
import asyncio
from enum import Enum
from http.server import BaseHTTPRequestHandler
from io import BytesIO
from urllib.parse import urlparse

sys.path.append("/usr/local/lib/python3.10/dist-packages")

from dotenv import load_dotenv
import ol

file_sock_path = "/host/ol.sock"
file_sock = None
bootstrap_path = None

class EntryType(Enum):
    FUNC = "func"    # f(event) -> result
    WSGI = "wsgi"    # app(environ, start_response) -> iterable
    ASGI = "asgi"    # await app(scope, receive, send)


class RequestParser(BaseHTTPRequestHandler):
    def __init__(self, request_data):
        self.rfile = BytesIO(request_data)
        self.raw_requestline = self.rfile.readline()
        self.parse_request()


def handle_func(conn, request, entry_point):
    """Handle direct function calls: f(event) -> result"""
    try:
        content_length = int(request.headers.get('Content-Length', 0))
        body = request.rfile.read(content_length) if content_length else b""
        event = json.loads(body) if body else {}
        result = entry_point(event)
        response_body = json.dumps(result).encode()
        status, status_text = 200, "OK"
        content_type = "application/json"
    except Exception:
        response_body = traceback.format_exc().encode()
        status, status_text = 500, "Internal Server Error"
        content_type = "text/plain"

    conn.sendall(f"HTTP/1.1 {status} {status_text}\r\n".encode())
    conn.sendall(f"Content-Type: {content_type}\r\n".encode())
    conn.sendall(f"Content-Length: {len(response_body)}\r\n".encode())
    conn.sendall(b"\r\n")
    conn.sendall(response_body)


def handle_wsgi(conn, request, entry_point, app_name, path_info, query_string):
    """Handle WSGI apps: app(environ, start_response) -> iterable"""
    content_length = request.headers.get('Content-Length', '')
    body = request.rfile.read(int(content_length)) if content_length else b""

    # Host header is required in HTTP/1.1 (RFC 2616 section 14.23)
    # Note: we listen on a Unix socket, so port may not be meaningful
    host = request.headers['Host']
    if ':' in host:
        server_name, server_port = host.split(':', 1)
    else:
        server_name, server_port = host, ""

    # WSGI 1.0 (PEP 3333): https://peps.python.org/pep-3333/#environ-variables
    environ = {
        # CGI variables (required)
        "REQUEST_METHOD": request.command,
        "SCRIPT_NAME": "/run/" + app_name,
        "PATH_INFO": path_info,
        "QUERY_STRING": query_string,
        "SERVER_NAME": server_name,
        "SERVER_PORT": server_port,
        "SERVER_PROTOCOL": request.request_version,
        # wsgi.* variables (required)
        "wsgi.version": (1, 0),       # PEP 3333 specifies tuple (1, 0)
        "wsgi.url_scheme": "http",
        "wsgi.input": BytesIO(body),
        "wsgi.errors": sys.stderr,
        "wsgi.multithread": False,
        "wsgi.multiprocess": False,
        "wsgi.run_once": False,
    }
    # HTTP headers -> environ per CGI spec (RFC 3875 section 4.1.18):
    # - Convert to uppercase, replace "-" with "_"
    # - Prefix with "HTTP_" except Content-Type and Content-Length
    for key, value in request.headers.items():
        key = key.upper().replace("-", "_")
        if key in ("CONTENT_TYPE", "CONTENT_LENGTH"):
            environ[key] = value
        else:
            environ["HTTP_" + key] = value

    def start_response(status, response_headers, exc_info=None):
        conn.sendall(f"HTTP/1.1 {status}\r\n".encode())
        for name, value in response_headers:
            conn.sendall(f"{name}: {value}\r\n".encode())
        conn.sendall(b"\r\n")

    result = entry_point(environ, start_response)
    for chunk in result:
        conn.sendall(chunk)
    # PEP 3333: if iterable has close(), server must call it for cleanup
    if hasattr(result, 'close'):
        result.close()


def handle_asgi(conn, request, entry_point, app_name, path_info, query_string):
    """Handle ASGI apps: await app(scope, receive, send)"""
    content_length = int(request.headers.get('Content-Length', 0))
    body = request.rfile.read(content_length) if content_length else b""

    # ASGI 3.0: https://asgi.readthedocs.io/en/latest/specs/www.html#http-connection-scope
    scope = {
        "type": "http",
        "asgi": {"version": "3.0"},
        "http_version": request.request_version.split("/")[1],  # "HTTP/1.1" -> "1.1"
        "method": request.command,
        "scheme": "http",
        "path": path_info,
        "query_string": query_string.encode(),
        "root_path": "/run/" + app_name,
        "headers": [(k.lower().encode(), v.encode()) for k, v in request.headers.items()],
    }

    response_started = False

    async def receive():
        return {"type": "http.request", "body": body, "more_body": False}

    async def send(message):
        nonlocal response_started
        if message["type"] == "http.response.start":
            response_started = True
            status = message["status"]
            conn.sendall(f"HTTP/1.1 {status} OK\r\n".encode())
            for name, value in message.get("headers", []):
                conn.sendall(name + b": " + value + b"\r\n")
            conn.sendall(b"\r\n")
        elif message["type"] == "http.response.body":
            conn.sendall(message.get("body", b""))

    try:
        asyncio.run(entry_point(scope, receive, send))
    except Exception:
        if not response_started:
            error = traceback.format_exc().encode()
            conn.sendall(b"HTTP/1.1 500 Internal Server Error\r\n")
            conn.sendall(b"Content-Type: text/plain\r\n")
            conn.sendall(f"Content-Length: {len(error)}\r\n".encode())
            conn.sendall(b"\r\n")
            conn.sendall(error)


def web_server():
    print(f"server.py: start web server on fd: {file_sock.fileno()}")
    sys.path.append('/handler')
    os.chdir('/handler')  # so relative paths in app code work

    # Load environment variables from .env file if it exists
    env_path = '/handler/.env'
    if os.path.exists(env_path):
        load_dotenv(env_path)
        print(f"server.py: loaded environment variables from {env_path}")

    # Import handler module
    entry_file = os.environ.get('OL_ENTRY_FILE', 'f.py')
    if not entry_file.endswith('.py'):
        raise ValueError(f"OL_ENTRY_FILE must end with .py, got: {entry_file}")
    module_name = entry_file[:-3]
    handler_module = importlib.import_module(module_name)

    # Determine entry point and type
    wsgi_entry = os.environ.get('OL_WSGI_ENTRY')
    asgi_entry = os.environ.get('OL_ASGI_ENTRY')
    if wsgi_entry:
        entry_point = getattr(handler_module, wsgi_entry)
        entry_type = EntryType.WSGI
    elif asgi_entry:
        entry_point = getattr(handler_module, asgi_entry)
        entry_type = EntryType.ASGI
    elif hasattr(handler_module, 'f'):
        entry_point = handler_module.f
        entry_type = EntryType.FUNC
    elif hasattr(handler_module, 'app'):
        entry_point = handler_module.app
        # Detect ASGI vs WSGI: ASGI apps have async __call__
        if asyncio.iscoroutinefunction(getattr(entry_point, '__call__', None)):
            entry_type = EntryType.ASGI
        else:
            entry_type = EntryType.WSGI
    else:
        raise ValueError("No entry point found. Define 'f' or 'app' in your module.")

    print(f"server.py: entry_type={entry_type.value}")

    while True:
        conn, addr = file_sock.accept()
        data = conn.recv(4096)
        request = RequestParser(data)

        # Parse path: `/run/<app-name>/a/b/c` -> app_name, `/a/b/c`, query
        parsed = urlparse(request.path)
        parts = parsed.path.split("/")  # ["", "run", <app-name>, ...]
        app_name = parts[2]
        path_info = '/' + '/'.join(parts[3:])
        query_string = parsed.query

        if entry_type == EntryType.FUNC:
            handle_func(conn, request, entry_point)
        elif entry_type == EntryType.WSGI:
            handle_wsgi(conn, request, entry_point, app_name, path_info, query_string)
        elif entry_type == EntryType.ASGI:
            handle_asgi(conn, request, entry_point, app_name, path_info, query_string)

        conn.close()


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
            start_container()
            os._exit(1) # only reachable if program unnexpectedly returns


def start_container():
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
    if os.path.exists(file_sock_path):
        os.remove(file_sock_path)
    file_sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    file_sock.bind(file_sock_path)
    file_sock.listen(1)  # backlog=1: we handle one request at a time, no concurrency

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


def main():
    '''
    caller is expected to do chroot, because we want to use the
    python.exe inside the container
    '''

    global bootstrap_path

    if len(sys.argv) < 2:
        print("Expected execution: chroot <path_to_root_fs> python3 server.py <path_to_bootstrap.py> [cgroup-count] [enable-seccomp]")
        print("    cgroup-count: number of FDs (starting at 3) that refer to /sys/fs/cgroup/..../cgroup.procs files")
        print("    enable-seccomp: true/false to enable or disables seccomp filtering")
        sys.exit(1)

    print('server.py: started new process with args: ' + " ".join(sys.argv))

    #enable_seccomp if enable-seccomp is not passed
    if len(sys.argv) < 3 or sys.argv[3] == 'true':
        return_code = ol.enable_seccomp()
        assert return_code >= 0
        print('seccomp enabled')

    bootstrap_path = sys.argv[1]
    cgroup_fds = 0
    if len(sys.argv) > 2:
        cgroup_fds = int(sys.argv[2])

    # join cgroups passed to us.  The fact that chroot is called
    # before we start means we also need to pass FDs to the cgroups we
    # want to join, because chroot happens before we run, so we can no
    # longer reach them by paths.
    pid = str(os.getpid())
    for i in range(cgroup_fds):
        # golang guarantees extras start at 3: https://golang.org/pkg/os/exec/#Cmd
        fd_id = 3 + i
        with os.fdopen(fd_id, "w") as file:
            file.write(pid)
            print(f'server.py: joined cgroup, close FD {fd_id}')

    start_container()


if __name__ == '__main__':
    main()
