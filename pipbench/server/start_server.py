import subprocess
import signal
import sys
import os

SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))

def kill_all(signal, frame):
    p1.kill()
    p2.kill()
    sys.exit(0)

if not os.path.exists('packages'):
    os.makedirs('packages')

p1 = subprocess.Popen(['gunicorn', 'server:app', '-b', '127.0.0.1:9198', '--log-file', '%s/http_server.log' % SCRIPT_DIR])
p2 = subprocess.Popen(['pypi-server', '-p', '9199', '--log-file', '%s/pypi_server.log' % SCRIPT_DIR, './packages'])

print('Started PipBench server')

signal.signal(signal.SIGINT, kill_all)
signal.pause()
