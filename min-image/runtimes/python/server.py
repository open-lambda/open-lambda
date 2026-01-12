# pylint: disable=line-too-long,global-statement,invalid-name,broad-except

''' Python runtime for sock '''

import os
import sys
import socket
import struct
import traceback

sys.path.append("/usr/local/lib/python3.10/dist-packages")
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

import ol
from server_common import web_server_on_sock

file_sock_path = "/host/ol.sock"
file_sock = None
bootstrap_path = None


def web_server():
    """Wrapper that calls web_server_on_sock with the global file_sock."""
    web_server_on_sock(file_sock, server_name="server.py")


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
