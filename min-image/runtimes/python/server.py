# pylint: disable=line-too-long,global-statement,invalid-name,broad-except

''' Python runtime for sock '''
import http
import os, sys, json, argparse, importlib, traceback, time, fcntl, array, socket, struct, resource
import subprocess

sys.path.append("/usr/local/lib/python3.10/dist-packages")

import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.wsgi
import tornado.netutil
import requests

import ol

file_sock_path = "/host/ol.sock"
file_sock = None
bootstrap_path = None
proc_path = "/proc"
# when current container is a zygote, this is the generation of the zygote, otherwise -1
split_gen = -1


def get_page_count(pid):
    with open(f"{proc_path}/{pid}/statm") as f:
        return int(f.readline().split()[0])


def get_pss(pid):
    with open(f"{proc_path}/{pid}/smaps") as f:
        pss = 0
        for line in f:
            if line.startswith("Pss:"):
                pss += int(line.split()[1])
        return pss


def recv_fds(sock, msglen, maxfds):
    '''
    copied from https://docs.python.org/3/library/socket.html#socket.socket.recvmsg
    '''

    fds = array.array("i")  # Array of ints
    msg, ancdata, _flags, _addr = sock.recvmsg(msglen, socket.CMSG_LEN(maxfds * fds.itemsize))
    for cmsg_level, cmsg_type, cmsg_data in ancdata:
        if (cmsg_level == socket.SOL_SOCKET and cmsg_type == socket.SCM_RIGHTS):
            # Append data, ignoring any truncated integers at the end.
            fds.frombytes(cmsg_data[:len(cmsg_data) - (len(cmsg_data) % fds.itemsize)])
    return msg, list(fds)


def web_server():
    print(f"server.py: start web server on fd: {file_sock.fileno()}")
    sys.path.append('/handler')

    # TODO: as a safeguard, we should add a mechanism so that the
    # import doesn't happen until the cgroup move completes, so that a
    # malicious child cannot eat up Zygote resources
    import f

    class SockFileHandler(tornado.web.RequestHandler):
        def post(self):
            try:
                data = self.request.body
                try:
                    event = json.loads(data)
                except:
                    self.set_status(400)
                    self.write(f'bad POST data: "{data}"')
                    return
                self.write(json.dumps(f.f(event)))
            except Exception:
                self.set_status(500)  # internal error
                self.write(traceback.format_exc())

    if hasattr(f, "app"):
        # use WSGI entry
        app = tornado.wsgi.WSGIContainer(f.app)
    else:
        # use function entry
        app = tornado.web.Application([
            (".*", SockFileHandler),
        ])
    server = tornado.httpserver.HTTPServer(app)
    server.add_socket(file_sock)
    tornado.ioloop.IOLoop.instance().start()
    server.start()


def fork_server(split_generation=-1):
    global split_gen
    split_gen = split_generation
    fork_count = 0

    global file_sock

    file_sock.setblocking(True)
    print(f"server.py: start fork server on fd: {file_sock.fileno()}")

    while True:
        client, _info = file_sock.accept()
        _, fds = recv_fds(client, 8, 2)
        root_fd, mem_cgroup_fd = fds

        t0 = time.time()
        pid = os.fork()
        t1 = time.time()
        fork_count += 1
        if pid:
            try:
                resp = requests.post("http://127.0.0.1:4998/fork",
                                     json={"splitGeneration": split_gen,
                                           "forkTime": (t1 - t0) * 1000,
                                           "forkCount": fork_count,
                                           "pages": get_page_count(os.getpid()),  # num
                                           "maxRss": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,  # KB
                                           "pss": get_pss(os.getpid()),  # KB
                                           }
                                     )
                if resp.status_code != 200:
                    print(f"server.py: response status code is invalid: {resp.status_code}")
            except Exception as e:
                print(f"server.py: fork_server: failed to send fork info to RESTApi: {e}")

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
            client.sendall(
                struct.pack("I", pid)
            )  # who is the client: C.sendRootFD, but C.sendRootFD ignore the return pid
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
            os._exit(1)  # only reachable if program unexpectedly returns


# start_container will be called in 3 situations:
# 1. when the container is a zygote, it will be called by fork_server
# 2. when the container is a root zygote, it will be called by main()
# 3. when the container is a pip-install container, it will be called by main()
def start_container():
    '''
    1. this assumes chroot has taken us to the location where the
        container should start.
    2. it launches the container code by running whatever is in the
        bootstrap file (from argv)
    '''

    global file_sock
    # TODO: if we can get rid of this, we can get rid of the ns module
    try:
        return_val = ol.unshare()
    except RuntimeError as e:
        print("An error occurred in ol.unshare():", e)
        return_val = 1
    assert return_val == 0

    # we open a new .sock file in the child, before starting the grand
    # child, which will actually use it.  This is so that the parent
    # can know that once the child exits, it is safe to start sending
    # messages to the sock file.
    file_sock = tornado.netutil.bind_unix_socket(file_sock_path)

    t0 = time.time()
    pid = os.fork()
    t1 = time.time()

    assert pid >= 0

    if pid > 0:
        # print(f"start_container fork = {(t1 - t0)*1000} ms, #pages={get_page_count(os.getpid())}")
        # orphan the new process by exiting parent.  The parent
        # process is in a weird state because unshare only partially
        # works for the process that calls it.
        os._exit(0)

    # # try to mount /proc, so that we could get number of pages (host dir seems not writable)
    # if os.path.exists(proc_path) is False:
    #     os.mkdir(proc_path)
    print(os.path.exists(proc_path))
    result = subprocess.run(["mount", "-t", "proc", "proc", f"{proc_path}"],
                            stderr=subprocess.PIPE)

    if result.returncode != 0:
        print("mount proc err:", result.stderr.decode('utf-8'))
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
        print(
            "Expected execution: chroot <path_to_root_fs> python3 server.py <path_to_bootstrap.py> [cgroup-count] [enable-seccomp]")
        print("    cgroup-count: number of FDs (starting at 3) that refer to /sys/fs/cgroup/..../cgroup.procs files")
        print("    enable-seccomp: true/false to enable or disables seccomp filtering")
        sys.exit(1)

    print('server.py: started new process with args: ' + " ".join(sys.argv))

    # enable_seccomp if enable-seccomp is not passed
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
