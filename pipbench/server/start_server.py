import subprocess
import signal
import sys
import os


def kill_all(signal, frame):
    p1.kill()
    p2.kill()
    sys.exit(0)

if not os.path.exists('packages'):
    os.makedirs('packages')

p1 = subprocess.Popen(['gunicorn', 'server:app', '-b', '127.0.0.1:9198'])
p2 = subprocess.Popen(['pypi-server', '-p', '9199', './packages'])

signal.signal(signal.SIGINT, kill_all)
signal.pause()
