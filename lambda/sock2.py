import os, sys, json, argparse, importlib, traceback, time
import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.netutil
import ns

def web_server(sock_path):
    print("serve from Tornado")

    class SockFileHandler(tornado.web.RequestHandler):
        def post(self):
            # we don't import this until we get a request; this is a
            # safeguard in case lambda_func is malicious (we don't
            # want it to interfere with ongoing setup, such as the
            # move to the new cgroups)
            import lambda_func

            try:
                data = self.request.body
                try :
                    event = json.loads(data)
                except:
                    self.set_status(400)
                    self.write('bad POST data: "%s"'%str(data))
                    return
                self.write(json.dumps(lambda_func.handler(event)))
            except Exception:
                self.set_status(500) # internal error
                self.write(traceback.format_exc())

    tornado_app = tornado.web.Application([
        (".*", SockFileHandler),
    ])
    server = tornado.httpserver.HTTPServer(tornado_app)
    socket = tornado.netutil.bind_unix_socket(sock_path)
    server.add_socket(socket)
    tornado.ioloop.IOLoop.instance().start()
    server.start()


def fork_server(sock_path):
    sock = ns.open_sock_file(sock_path)
    print("Got sock file fd: %d" % sock)
    assert sock >= 0

    while True:
        # the parent will have already waited for the child to exit
        # before this even returns
        pid = ns.fork_to_next_fd(sock)

        if pid == 0:
            # child
            start_container()

            # start_container should never return, unless there's a bug in the user's code
            os._exit(1)


# 1. this assumes chroot has taken us to the location where the container should start.
# 2. it launches the container code by running whatever is in the bootstrap file (from argv)
def start_container():
    rv = ns.unshare()
    assert rv == 0

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
        exec(f.read())


# caller is expected to do chroot
def main():
    global bootstrap_path

    if len(sys.argv) < 2:
        print("Expected execution: chroot <path_to_root_fs> python3 sock2.py <path_to_bootstrap.py> [cgroup-count]")
        print("    cgroup-count: number of FDs (starting at 3) that refer to /sys/fs/cgroup/..../cgroup.procs files")
        sys.exit(1)

    print('started with args: ' + " ".join(sys.argv))

    bootstrap_path = sys.argv[1]
    cgroup_fds = 0
    if len(sys.argv) > 2:
        cgroup_fds = int(sys.argv[2])

    # join cgroups passed to us
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
