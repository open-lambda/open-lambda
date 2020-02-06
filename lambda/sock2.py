import os, sys, json, argparse, importlib, traceback, time, fcntl, array, socket, struct
import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.netutil
import ol

file_sock_path = "/host/ol.sock"
file_sock = None

# copied from https://docs.python.org/3/library/socket.html#socket.socket.recvmsg
def recv_fds(sock, msglen, maxfds):
    fds = array.array("i")   # Array of ints
    msg, ancdata, flags, addr = sock.recvmsg(msglen, socket.CMSG_LEN(maxfds * fds.itemsize))
    for cmsg_level, cmsg_type, cmsg_data in ancdata:
        if (cmsg_level == socket.SOL_SOCKET and cmsg_type == socket.SCM_RIGHTS):
            # Append data, ignoring any truncated integers at the end.
            fds.fromstring(cmsg_data[:len(cmsg_data) - (len(cmsg_data) % fds.itemsize)])
    return msg, list(fds)


def web_server():
    print("sock2.py: start web server on fd: %d" % file_sock.fileno())
    sys.path.append('/handler')

    class SockFileHandler(tornado.web.RequestHandler):
        def post(self):
            # we don't import this until we get a request; this is a
            # safeguard in case f is malicious (we don't
            # want it to interfere with ongoing setup, such as the
            # move to the new cgroups)
            import f

            try:
                data = self.request.body
                try :
                    event = json.loads(data)
                except:
                    self.set_status(400)
                    self.write('bad POST data: "%s"'%str(data))
                    return
                self.write(json.dumps(f.f(event)))
            except Exception:
                self.set_status(500) # internal error
                self.write(traceback.format_exc())

    tornado_app = tornado.web.Application([
        (".*", SockFileHandler),
    ])
    server = tornado.httpserver.HTTPServer(tornado_app)
    server.add_socket(file_sock)
    tornado.ioloop.IOLoop.instance().start()
    server.start()


def fork_server():
    global file_sock

    file_sock.setblocking(True)
    print("sock2.py: start fork server on fd: %d" % file_sock.fileno())

    while True:
        client, info = file_sock.accept()
        _, fds = recv_fds(client, 8, 2)
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


# 1. this assumes chroot has taken us to the location where the
#    container should start.
# 2. it launches the container code by running whatever is in the
#    bootstrap file (from argv)
def start_container():
    global file_sock

    # TODO: if we can get rid of this, we can get rid of the ns module
    rv = ol.unshare()
    assert rv == 0

    # we open a new .sock file in the child, before starting the grand
    # child, which will actually use it.  This is so that the parent
    # can know that once the child exits, it is safe to start sending
    # messages to the sock file.
    file_sock = tornado.netutil.bind_unix_socket(file_sock_path)

    pid = os.fork()
    assert(pid >= 0)

    if pid > 0:
        # orhpan the new process by exiting parent.  The parent
        # process is in a weird state because unshare only partially
        # works for the process that calls it.
        os._exit(0)

    with open(bootstrap_path) as f:
        # this code can be whatever OL decides, but it will probably do the following:
        # 1. some imports
        # 2. call either web_server or fork_server
        code = f.read()
        try:
            exec(code)
        except Exception as e:
            print("Exception: " + traceback.format_exc())
            print("Problematic Python Code:\n" + code)

# caller is expected to do chroot, because we want to use the
# python.exe inside the container
def main():
    global bootstrap_path

    if len(sys.argv) < 2:
        print("Expected execution: chroot <path_to_root_fs> python3 sock2.py <path_to_bootstrap.py> [cgroup-count]")
        print("    cgroup-count: number of FDs (starting at 3) that refer to /sys/fs/cgroup/..../cgroup.procs files")
        sys.exit(1)

    print('sock2.py: started new process with args: ' + " ".join(sys.argv))

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
        fd = 3 + i
        f = os.fdopen(fd, "w")
        f.write(pid)
        print('sock2.py: joined cgroup, close FD %d' % fd)
        f.close()

    start_container()


if __name__ == '__main__':
    main()
