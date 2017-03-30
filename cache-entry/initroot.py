"""
Usage: python initservers.py

Initializes the root forkserver.

Expects that a volume has been mapped into the container
from the worker as /host. The forkserver will have a
corresponding directory /host/fs.

Forkserver's directory directory contains:
stdout - file for logging stdout
stderr - file for logging stderr
fs.sock - unix domain socket file for sending forkenter requests

The internal container PIDs of the spawned forkserver will 
also be logged (in order) to '/host/fspid'.
"""

import sys, os, signal
from subprocess import Popen

def main():
    server_dir = '/host'
    if not os.path.exists(server_dir):
        try:
            os.mkdir(server_dir)
        except:
            print('Failed to make forkserver directory. /host mounted?')
            sys.exit(1)

    stdout_path = os.path.join(server_dir, 'stdout')
    stderr_path = os.path.join(server_dir, 'stderr')
    sock_path = os.path.join(server_dir, 'fs.sock')

    args = ['python', '/server.py', sock_path]
    stdout_fd = open(stdout_path, 'w')
    stderr_fd = open(stderr_path, 'w')
    p = Popen(args, stdout=stdout_fd, stderr=stderr_fd)

    # write forkserver pids to file
    with open('/host/pid', 'w') as fd:
        fd.write('%s\n' % p.pid)

    # sleep forever (so container doesn't exit)
    signal.pause()

if __name__ == '__main__':
    main()
