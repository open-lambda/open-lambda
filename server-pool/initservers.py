"""
Usage: python initservers.py <num>

Initializes <num> forkservers (<num> defaults to 1).

Expects that a volume has been mapped into the container
from the worker as /host. Each forkserver will have a
corresponding directory /host/<k>, with <k> in [0, <num>].

Each forkserver's directory directory contains:
stdout - file for logging stdout
stderr - file for logging stderr
fs.sock - unix domain socket file for sending forkenter requests

The internal container PIDs of the spawned forkservers will 
also be logged (in order) to '/host/fspids'.
"""

import sys, os, signal
from subprocess import Popen

def main(num):
    pids = []

    # track the 
    for k in range(num):
        server_dir = os.path.join('/host', 'fs%s' % k)
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

        pids.append(p.pid)

    # write forkserver pids to file
    with open('/host/fspids', 'w') as fd:
        for pid in pids:
            fd.write('%s\n' % pid)

    # wait for forkservers to start before returning
    for k in range(num):
        sock_path = os.path.join('/host', 'fs%s' % k, 'fs.sock')
        while not os.path.exists(sock_path):
            pass

    # sleep forever (so container doesn't exit)
    signal.pause()

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print('Number of forkservers not specified, defaulting to 1')
        num = 1
    else:
        try:
            num = int(sys.argv[1])
        except:
            print('Usage: python %s <num>' % sys.argv[0])
            sys.exit(1)

    main(num)
