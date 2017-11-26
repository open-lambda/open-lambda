#!/usr/bin/python
import traceback, json, sys, socket, os, importlib, pip, hashlib, signal
import rethinkdb
import tornado.ioloop
import tornado.web
import tornado.httpserver
import tornado.netutil
from subprocess import check_output


HOST_PATH = '/host'
SOCK_PATH = '%s/ol.sock' % HOST_PATH
FS_PATH = '%s/fs.sock' % HOST_PATH
STDOUT_PATH = '%s/stdout' % HOST_PATH
STDERR_PATH = '%s/stderr' % HOST_PATH

# path in container where install cache is mounted
INSTALL_CACHE_PATH = '/packages'
# path of file that container handler
HANDLER_FUNC_FILE = '/handler/lambda_func.py'
# path of file where packages are listed to be imported
PKGS_LIST_FILE = '/handler/packages.txt'

PROCESSES_DEFAULT = 10

# global for handler db access
db_conn = None

def import_lambda_func():
    global lambda_func
    sys.path.append('/handler')
    import lambda_func # assume submitted .py file is /handler/lambda_func.py

def load_config():
    return json.loads(os.environ['ol.config'])

def setup_db_conn(config):
    global db_conn
    if config.get('db', None) == 'rethinkdb':
        host = config.get('rethinkdb.host', 'localhost')
        port = config.get('rethinkdb.port', 28015)
        print 'Connect to %s:%d' % (host, port)
        db_conn = rethinkdb.connect(host, port)

# create symbolic links from install cache to dist-packages, return if success
def create_link(pkg):
    # assume no version (e.g. "==1.2.1")
    # the path where the package was already installed to, relative to the install cache
    pkg_src_dir = '%s/%s' % (INSTALL_CACHE_PATH, pkg)
    if os.path.exists(pkg_src_dir):
        link_name = '/host/pip/%s' % name
        if os.path.exists(link_name):
            print('link failed, path already exists, assuming okay: %s' % link_name)
            sys.stdout.flush()
        else:
            os.symlink(pkg_src_dir, link_name)
        return True
    return False

def create_official_pkg_installer():
    def installer(pkg):
        check_output(' '.join(['pip', 'install', '-t', '/host/pip', pkg]), shell=True)
    return installer

def create_mirror_pkg_installer(mirror_host, mirror_port):
    def installer(pkg):
        check_output(' '.join(['pip', 'install', '-t', '/host/pip', '--no-cache-dir', '--index-url', 'http://%s:%s/simple' % (mirror_host, mirror_port), '--trusted-host', mirror_host, pkg]), shell=True)
    return installer

def do_installs(installer):
    sys.path.append('/host/pip')
    with open(PKGS_LIST_FILE) as fd:
        for line in fd:
            try:
                line_split = line.strip().split(':')
                if len(line_split) != 2 or line_split[0] == '' or line_split[1] == '':
                    raise Exception('bad line %s' % pkg)
                else:
                    pkg = line_split[1]
                    if create_link(pkg):
                        print('using install cache: %s' % pkg)
                        sys.stdout.flush()
                    else:
                        print('installing: %s' % pkg)
                        sys.stdout.flush()
                        try:
                            installer(pkg)
                            sys.stdout.flush()
                        except Exception as e:
                            print('failed to install %s with %s' % (pkg, e))
                            sys.stdout.flush()
                            sys.exit(1)

            except Exception as e:
                print('malformed packages.txt file: %s' % e)
                sys.stdout.flush()
                sys.exit(1)

class SockFileHandler(tornado.web.RequestHandler):
    def post(self):
        try:
            data = self.request.body
            try :
                event = json.loads(data)
            except:
                self.set_status(400)
                self.write('bad POST data: "%s"'%str(data))
                return
            self.write(json.dumps(lambda_func.handler(db_conn, event)))
        except Exception:
            self.set_status(500) # internal error
            self.write(traceback.format_exc())

tornado_app = tornado.web.Application([
    (r".*", SockFileHandler),
])

# listen on sock file with Tornado
def init_server():
    server = tornado.httpserver.HTTPServer(tornado_app)
    socket = tornado.netutil.bind_unix_socket(SOCK_PATH)
    server.add_socket(socket)
    # notify worker server that we are ready through stdout
    # flush is necessary, and don't put it after tornado start; won't work
    with open('/host/server_pipe', 'a') as pipe:
        pipe.write('ready')
    tornado.ioloop.IOLoop.instance().start()
    server.start(PROCESSES_DEFAULT)

def setup_installer(installer_args):
    try:
        mirror_host = installer_args[1]
        mirror_port = installer_args[2]
        return create_mirror_pkg_installer(mirror_host, mirror_port)
    except:
        return create_official_pkg_installer()

def wait_for_mount():
    while not os.path.exists(HANDLER_FUNC_FILE):
        time.sleep(0.005)
        curr += 0.005
        if curr > 1.0:
            print('lambda_func.py missing (path=%s)' % HANDLER_FUNC_FILE)
            sys.stdout.flush()
            sys.exit(1)

def forward_stdio():
    try:
        sys.stdout = open(STDOUT_PATH, 'w')
        sys.stderr = open(STDERR_PATH, 'w')
    except Exception as e:
        with open('/ERROR', 'w') as fd:
            fd.write('failed to open stdout/stderr with: %s\n' % e)
            sys.exit(1)

# listen for fds to forkenter
def fdlisten(installer):
    import ns
 
    signal = "cache"
    r = -1
    count = 0
    # only child meant to serve ever escapes the loop
    while r != 0 or signal == "cache":
        if r == 0:
            print('RESET')
            sys.stdout.flush()
            ns.reset()

        print('LISTENING')
        sys.stdout.flush()
        data = ns.fdlisten(FS_PATH).split()
        sys.stdout.flush()

        r = ns.forkenter()
        sys.stdout.flush()
        if r == 0:
            redirect()

            mods = []
            pkgs = []
            for info in data[:-1]:
                split = info.split(':')
                mods.append(split[0])
                if split[1] != '':
                    pkgs.append(split[1])

            # use install cache
            remains = []
            for pkg in pkgs:
                if create_link(pkg):
                    print('using install cache: %s' % pkg)
                else:
                    remains.append(pkg)
            pkgs = remains

            # install from pip mirror
            for pkg in pkgs:
                print('installing: %s' % pkg)
                try:
                    installer(pkg)
                except Exception as e:
                    print('install %s failed with: %s' % (split[1], e))
            
	    sys.path.insert(1, '/host/pip')
            # import modules
            for mod in mods:
                print('importing: %s' % mod)
                try:
                    globals()[mod] = importlib.import_module(mod)
                except Exception as e:
                    print('failed to import %s with: %s' % (mod, e))

            signal = data[-1]
            print('signal: %s' % signal)

            sys.stdout.flush()
            sys.stderr.flush()

        print('')
        sys.stdout.flush()

        count += 1

    print('SERVING HANDLERS')
    sys.stdout.flush()
    lambda_init()

def redirect():
    sys.stdout.close()
    sys.stderr.close()
    sys.stdout = open(STDOUT_PATH, 'w')
    sys.stderr = open(STDERR_PATH, 'w')

def install_from_file(installer):
    wait_for_mount()
    if os.path.exists(PKGS_LIST_FILE):
        do_installs(installer)
    else:
        print('no packages list file found, assuming lambda doesn\'t import any non-runtime included packages')
        sys.stdout.flush()

def lambda_init():
    import_lambda_func()
    config = load_config()
    if config != None:
        setup_db_conn(config)
    init_server()

def cache_entry_init(installer):
    fdlisten(installer)

if __name__ == '__main__':
    forward_stdio()

    if len(sys.argv) != 2 and len(sys.argv) != 4:
        print('Usage: python <lambda | cache-entry> %s or python %s <lambda | cache-entry> <index_host> <index_sock>' % (sys.argv[0], sys.argv[0]))
        sys.exit(1)

    installer_args = []
    if len(sys.argv) == 4:
        installer_args = sys.argv[2:4]
    installer = setup_installer(installer_args)
    
    if sys.argv[1] == 'handler':
        install_from_file(installer)
        lambda_init()
    elif sys.argv[1] == 'cache':
        cache_entry_init(installer)
    else:
        print('Usage: python <lambda | cache-entry> %s or python %s <lambda | cache-entry> <index_host> <index_sock>' % (sys.argv[0], sys.argv[0]))
        sys.exit(1)
