# pylint: disable=line-too-long,invalid-name,broad-except

'''
Common code shared between server.py and server_legacy.py
'''

import os
import sys
import json
import asyncio
import importlib
import traceback
from enum import Enum
from http.server import BaseHTTPRequestHandler
from urllib.parse import urlparse

from dotenv import load_dotenv


class EntryType(Enum):
    FUNC = "func"    # f(event) -> result
    WSGI = "wsgi"    # app(environ, start_response) -> iterable
    ASGI = "asgi"    # await app(scope, receive, send)


class RequestParser(BaseHTTPRequestHandler):
    def __init__(self, conn):
        self.rfile = conn.makefile('rb', buffering=65536)
        self.raw_requestline = self.rfile.readline()
        self.parse_request()
        self.remaining = int(self.headers.get('Content-Length', 0))

    def read(self, size=-1):
        if size < 0:
            size = self.remaining
        size = min(size, self.remaining)
        data = self.rfile.read(size)
        self.remaining -= len(data)
        return data


def handle_func(conn, request, entry_point):
    """Handle direct function calls: f(event) -> result"""
    try:
        body = request.read()
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
        "wsgi.input": request,  # request.read() handles Content-Length limiting
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
    # TODO: stream body using more_body flag instead of reading all upfront
    body = request.read()

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


def web_server_on_sock(file_sock, server_name="server"):
    """
    Main web server loop. Accepts connections and dispatches to appropriate handler.

    Args:
        file_sock: The socket to accept connections on
        server_name: Name for logging (e.g., "server.py" or "server_legacy.py")
    """
    print(f"{server_name}: start web server on fd: {file_sock.fileno()}")
    sys.path.append('/handler')

    # Load environment variables from .env file if it exists
    env_path = '/handler/.env'
    if os.path.exists(env_path):
        load_dotenv(env_path)
        print(f"{server_name}: loaded environment variables from {env_path}")

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

    print(f"{server_name}: entry_type={entry_type.value}")

    while True:
        conn, _ = file_sock.accept()
        request = RequestParser(conn)

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
