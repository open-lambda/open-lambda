import os, sys, subprocess, json, argparse, time, shutil
import netifaces
import rethinkdb as r
from common import *

def container_ip(cid):
    inspect = run_js('docker inspect '+cid)
    return only(inspect)['NetworkSettings']['IPAddress']

def lookup_host_port(cid, guest_port):
    inspect = run_js('docker inspect '+cid)
    return only(only(inspect)['NetworkSettings']['Ports'][str(guest_port)+'/tcp'])['HostPort']

def my_ip(name='eth0'):
    try :
        ip = netifaces.ifaddresses('eth0')[netifaces.AF_INET][0]['addr']
    except:
        print 'Could not find an IP using netifaces, using 127.0.0.1'
        ip = '127.0.0.1'
        pass
    return ip

class Cluster:
    def __init__(self, numworkers, cluster, ports):
        self.numworkers = numworkers
        self.workers = []
        self.cluster = cluster
        self.internal_reg_port = ports['registry']
        self.internal_worker_port = ports['worker']
        self.internal_lb_port = ports['balancer']
        self.script_dir = os.path.dirname(os.path.realpath(__file__))

    def make_cluster_dir(self):
        self.cluster_dir = os.path.join(self.script_dir, self.cluster)

        if os.path.exists(self.cluster_dir):
            print 'Cluster already running!'
            print 'Use stop-cluster.py to clean up.'
            sys.exit(1)

        os.mkdir(self.cluster_dir)

class LocalCluster(Cluster):
    def __init__(self, numworkers, cluster, ports):
        Cluster.__init__(self, numworkers, cluster, ports)
        self.host_ip = my_ip() 

    # TODO: specify type of registry
    def write_reg(self, cid):
        self.registry_ip = container_ip(cid)
        self.registry_port = lookup_host_port(cid, self.internal_reg_port)

        config_path = os.path.join(self.cluster_dir, 'registry.json')
        config = {'cid': cid,
                  'ip': self.registry_ip,
                  'host_ip': self.host_ip,
                  'host_port': self.registry_port,
                  'type': 'registry'}
        wrjs(config_path, config)

        print 'Started registry "localhost:%s"' % self.registry_port

    def start_olreg(self):
        c = 'docker run -d -p 28015:28015 -p 29015:29015 rethinkdb rethinkdb --bind all'

        # for subsequent hosts in cluster:
        #c = 'docker run -d -p 28015:28015 -p 29015:29015 dockerfile/rethinkdb rethinkdb --bind all -j <first-host-ip>:29015'
        cid = run(c).strip()

        self.rethinkdb_ip = container_ip(cid)
        self.rethinkdb_port = lookup_host_port(cid, 28015)

        print 'Started rethinkdb container "%s:%s"' % (self.rethinkdb_ip, "28015")

        config_path = os.path.join(self.cluster_dir, 'registry.json')
        config = {'cid': cid,
                  'ip': self.rethinkdb_ip,
                  'host_ip': self.host_ip,
                  'host_port': self.rethinkdb_port,
                  'type': 'rethinkdb'}
        wrjs(config_path, config)

        c = 'docker run -d -p 0:%s olregistry /open-lambda/registry %s:%s' % (self.internal_reg_port, self.rethinkdb_ip, "28015")
        cid = run(c).strip()

        self.write_reg(cid)

    def start_docker_reg(self):
        c = 'docker run -d -p 0:%s registry:2' % self.internal_reg_port
        cid = run(c).strip()

        self.write_reg(cid)

    def start_workers(self):
        assert(int(self.numworkers) > 0)
        for i in range(int(self.numworkers)):
            config_path = os.path.join(self.cluster_dir, 'worker-%d.json' % i)
            config = {'registry_host': self.registry_ip,
                      'registry_port': self.internal_reg_port,
                      'type': 'worker',
                      'cluster_name': self.cluster}
            if i > 0:
                config['rethinkdb_join'] = self.workers[0]['ip']+':29015'

            wrjs(config_path, config)
            volumes = [('/sys/fs/cgroup', '/sys/fs/cgroup'),
                       (config_path, '/open-lambda/config.json')]
            c = 'docker run -d --privileged <VOLUMES> -p 0:%s lambda-node' % self.internal_worker_port
            c = c.replace('<VOLUMES>', ' '.join(['-v %s:%s'%(host,guest)
                                                 for host,guest in volumes]))
            cid = run(c).strip()
            config['cid'] = cid
            config['ip'] = container_ip(cid)
            config['host_ip'] = self.host_ip
            config['host_port'] = lookup_host_port(cid, self.internal_worker_port)
            wrjs(config_path, config, atomic=True)

            print 'Started worker "localhost:%s"' % config['host_port']
            self.workers.append(config)

    def start_lb(self):
        nginx_path = os.path.join(self.script_dir, 'nginx.config')

        self.write_nginx_config(nginx_path)
        c = 'docker run -p 0:%s -v %s:/etc/nginx/nginx.conf:ro -d nginx' % (self.internal_lb_port, nginx_path)
        cid = run(c).strip()

        self.balancer_ip = container_ip(cid)
        self.balancer_port = lookup_host_port(cid, self.internal_lb_port)
        config_path = os.path.join(self.cluster_dir, 'loadbalancer.json')
        config = {'cid': cid,
                  'ip': self.balancer_ip,
                  'host_ip': self.host_ip,
                  'host_port': self.balancer_port,
                  'type': 'loadbalancer'}    

        config_path = os.path.join(self.cluster_dir, 'loadbalancer-%d.json' % 1)
        wrjs(config_path, config, atomic=True)

        print 'Started loadbalancer "localhost:%s"' % self.balancer_port

    def write_nginx_config(self, path):
        config = 'http {\n\tupstream handlers {\n'

        for worker in self.workers:
            config += '\t\tserver %s:%s;\n' % (worker['host_ip'], worker['host_port'])

        config += '\t}\n\tserver {\n\t\tlisten %s;\n\t\tlocation / {\n\t\t\tproxy_pass http://handlers;\n\t\t}\n\t}\n}\nevents{}' % self.internal_lb_port

        with open(path, 'w') as fd:
            fd.write(config)	

    def rethinkdb_wait(self):
        for i in range(10):
            try:
                r.connect(self.workers[0]['ip'], 28015).repl()
                up = len(list(r.db('rethinkdb').table('server_status').run()))
                if up < len(self.workers):
                    print '%d of %d rethinkdb instances are ready' % (up, len(self.workers))
            except:
                print 'waiting for first rethinkdb instance to come up'

            time.sleep(1)

        print 'All rethinkdb instances are ready'

    # TODO: update directions for rethink registry
    def print_directions(self):
        print '='*40
        print 'Push images to OpenLambda registry as follows (or similar):'
        print 'IMG=hello && docker tag $IMG localhost:%s/$IMG; docker push localhost:%s/$IMG' % (self.registry_port, self.registry_port)
        print '='*40
        print 'Send requests directly to workers as follows (or similar):'
        print "IMG=hello && curl -w \"\\n\" -X POST localhost:%s/runLambda/$IMG -d '{}'" % self.workers[-1]['host_port']
        print '='*40
        print 'Send requests to the loadbalancer as follows (or similar):'
        print "IMG=hello && curl -w \"\\n\" -X POST localhost:%s/runLambda/$IMG -d '{}'" % self.balancer_port

class RemoteCluster(Cluster):
    def __init__(self, numworkers, cluster, ports):
        Cluster.__init__(self, numworkers, cluster, ports)

    def start_docker_reg(self):
        c = 'docker run -d -p 0:%s registry:2' % self.internal_reg_port
        cid = run(c).strip()

        self.registry_ip = container_ip(cid)
        self.registry_port = lookup_host_port(cid, self.internal_reg_port)

        config_path = os.path.join(self.cluster_dir, 'registry.json')
        config = {'cid': cid,
                  'ip': self.registry_ip,
                  'host_ip': self.host_ip,
                  'host_port': self.registry_port,
                  'type': 'registry'}
        wrjs(config_path, config)

        print 'Started registry %s:%s (or localhost:%s)' % (self.registry_ip, self.internal_reg_port, self.registry_port)

    def start_workers(self):
        assert(int(self.numworkers) > 0)
        for i in range(int(self.numworkers)):
            config_path = os.path.join(self.cluster_dir, 'worker-%d.json' % i)
            config = {'registry_host': self.registry_ip,
                      'registry_port': self.internal_reg_port,
                      'type': 'worker',
                      'cluster_name': self.cluster}
            if i > 0:
                config['rethinkdb_join'] = self.workers[0]['ip']+':29015'

            wrjs(config_path, config)
            volumes = [('/sys/fs/cgroup', '/sys/fs/cgroup'),
                       (config_path, '/open-lambda/config.json')]
            c = 'docker run -d --privileged <VOLUMES> -p 0:%s lambda-node' % self.internal_worker_port
            c = c.replace('<VOLUMES>', ' '.join(['-v %s:%s'%(host,guest)
                                                 for host,guest in volumes]))
            cid = run(c).strip()
            config['cid'] = cid
            config['ip'] = container_ip(cid)
            config['host_ip'] = self.host_ip
            config['host_port'] = lookup_host_port(cid, self.internal_worker_port)
            wrjs(config_path, config, atomic=True)

            print 'started worker %s:%s' % (config['ip'], self.internal_worker_port)
            self.workers.append(config)

    def start_lb(self):
        nginx_path = os.path.join(self.script_dir, 'nginx.config')

        self.write_nginx_config(nginx_path)
        c = 'docker run -p 0:%s -v %s:/etc/nginx/nginx.conf:ro -d nginx' % (self.internal_lb_port, nginx_path)
        cid = run(c).strip()

        self.balancer_ip = container_ip(cid)
        self.balancer_port = lookup_host_port(cid, self.internal_lb_port)
        config_path = os.path.join(self.cluster_dir, 'loadbalancer.json')
        config = {'cid': cid,
                  'ip': self.balancer_ip,
                  'host_ip': self.host_ip,
                  'host_port': self.balancer_port,
                  'type': 'loadbalancer'}    

        config_path = os.path.join(self.cluster_dir, 'loadbalancer-%d.json' % 1)
        wrjs(config_path, config, atomic=True)

        print 'started loadbalancer ' + self.balancer_ip + ':%s' % self.internal_lb_port

    def write_nginx_config(self, path):
        config = 'http {\n\tupstream handlers {\n'

        for worker in self.workers:
            config += '\t\tserver %s:%s;\n' % (worker['host_ip'], worker['host_port'])

        config += '\t}\n\tserver {\n\t\tlisten %s;\n\t\tlocation / {\n\t\t\tproxy_pass http://handlers;\n\t\t}\n\t}\n}\nevents{}' % self.internal_lb_port

        with open(path, 'w') as fd:
            fd.write(config)	

    def rethinkdb_wait(self):
        for i in range(10):
            try:
                r.connect(self.workers[0]['ip'], 28015).repl()
                up = len(list(r.db('rethinkdb').table('server_status').run()))
                if up < len(self.workers):
                    print '%d of %d rethinkdb instances are ready' % (up, len(self.workers))
            except:
                print 'waiting for first rethinkdb instance to come up'

            time.sleep(1)

        print 'all rethinkdb instances are ready'

    def print_directions(self):
        print '='*40
        print 'Push images to OpenLambda registry as follows (or similar):'
        print 'IMG=hello && docker tag $IMG localhost:%s/$IMG; docker push localhost:%s/$IMG' % (self.registry_port, self.registry_port)
        print 'OR'
        print ('IMG=hello && docker tag $IMG %s:%s/$IMG; docker push %s:%s/$IMG' %
               (self.host_ip, self.registry_port, self.host_ip, self.registry_port))
        print '='*40
        print 'Send requests directly to workers as follows (or similar):'
        print "IMG=hello && curl -w \"\\n\" -X POST %s:%s/runLambda/$IMG -d '{}'" % (self.workers[-1]['ip'], self.internal_worker_port)
        print 'OR'
        print "IMG=hello && curl -w \"\\n\" -X POST %s:%s/runLambda/$IMG -d '{}'" % (self.host_ip, self.workers[-1]['host_port'])
        print '='*40
        print 'Send requests to the loadbalancer as follows (or similar):'
        print "IMG=hello && curl -w \"\\n\" -X POST %s:%s/runLambda/$IMG -d '{}'" % (self.balancer_ip, self.internal_lb_port)
        print 'OR'
        print "IMG=hello && curl -w \"\\n\" -X POST %s:%s/runLambda/$IMG -d '{}'" % (self.host_ip, self.balancer_port)
