'''
Python Runtime for Docker

Note: SOCK doesn't use this anymore (it uses server.py instead), but
this is still here because we haven't updated docker.go yet.
'''

#pylint: disable=invalid-name,line-too-long,global-statement

import os
import sys
import argparse
import importlib
import socket

sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from dotenv import load_dotenv
from server_common import web_server_on_sock

HOST_DIR = '/host'
PKGS_DIR = '/packages'
HANDLER_DIR = '/handler'

# Load environment variables from .env file if it exists
env_path = f'{HANDLER_DIR}/.env'
if os.path.exists(env_path):
    load_dotenv(env_path)
    print(f"server_legacy.py: loaded environment variables from {env_path}")

sys.path.append(PKGS_DIR)
sys.path.append(HANDLER_DIR)

SOCK_PATH = os.path.join(HOST_DIR, 'ol.sock')
FS_PATH = os.path.join(HOST_DIR, 'fs.sock')
STDOUT_PATH = os.path.join(HOST_DIR, 'stdout')
STDERR_PATH = os.path.join(HOST_DIR, 'stderr')
SERVER_PIPE_PATH = os.path.join(HOST_DIR, 'server_pipe')

PROCESSES_DEFAULT = 10

parser = argparse.ArgumentParser(description='Listen and serve cache requests or lambda invocations.')
parser.add_argument('--cache', action='store_true', default=False, help='Begin as a cache entry.')


def lambda_server():
    """Start the lambda server on a Unix socket."""
    # Create and bind the socket
    if os.path.exists(SOCK_PATH):
        os.remove(SOCK_PATH)
    file_sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    file_sock.bind(SOCK_PATH)
    file_sock.listen(1)

    # Notify worker server that we are ready
    with open(SERVER_PIPE_PATH, 'w', encoding='utf-8') as pipe:
        pipe.write('ready')

    # Run the web server
    web_server_on_sock(file_sock, server_name="server_legacy.py")


def cache_loop():
    """Listen for fds to forkenter (Docker cache mode)."""
    import ns

    signal = "cache"
    r = -1
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
