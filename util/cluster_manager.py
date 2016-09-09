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
    def write_reg(self, cid, regtype):
        self.registry_ip = container_ip(cid)
        self.registry_port = lookup_host_port(cid, self.internal_reg_port)

        config_path = os.path.join(self.cluster_dir, 'registry.json')
        config = {'cid': cid,
                  'ip': self.registry_ip,
                  'host_ip': self.host_ip,
                  'host_port': self.registry_port,
                  'type': regtype}
        wrjs(config_path, config)

        print 'Started registry "localhost:%s"' % self.registry_port

    def start_olreg(self, num):
        nodes = []
        c = 'docker run -d -p 28015:28015 -p 29015:29015 rethinkdb rethinkdb --bind all'
        cid = run(c).strip()
        cluster_ip = container_ip(cid)
        cluster_port = lookup_host_port(cid, 28015)
        nodes.append({'cid': cid,
                    'ip': cluster_ip,
                    'host_ip': self.host_ip,
                    'host_port': cluster_port})

        print 'Started rethinkdb instance "localhost:28015"'
        print 40*'='

        for k in range(1, num):
            client_port = 28015 + k
            comm_port = 29015 + k
            c = 'docker run -d -p %s:%s -p %s:%s rethinkdb rethinkdb --port-offset %s --bind all -j %s:29015' % (client_port, client_port, comm_port, comm_port, k, cluster_ip)
            cid = run(c).strip()
            host_port = lookup_host_port(cid, 28015+k)
            nodes.append({'cid': cid,
                        'ip': container_ip(cid),
                        'host_ip': self.host_ip,
                        'host_port': host_port})

            print 'Started rethinkdb container "localhost:%s"' % host_port
            print 40*'=' 

        config_path = os.path.join(self.cluster_dir, 'rethinkdb.json')
        config = {
            'cluster': nodes,
            'ip': nodes[0]['ip'],
            'host_ip': self.host_ip,
            'host_port': nodes[0]['host_port'],
            'type': 'rethinkdb'
        }
        wrjs(config_path, config)

        c = 'docker run -d -p 0:%s olregistry /open-lambda/bin/pushserver %s %s:%s' % (self.internal_reg_port, self.internal_reg_port, cluster_ip, cluster_port)
        cid = run(c).strip()

        self.write_reg(cid, 'olregistry')

    def start_docker_reg(self):
        c = 'docker run -d -p 0:%s registry:2' % self.internal_reg_port
        cid = run(c).strip()

        self.write_reg(cid, 'docker')

    def start_workers(self, olregistry):
        if olregistry:
            registry = "olregistry"
        else:
            registry = "docker"

        assert(int(self.numworkers) > 0)
        for i in range(int(self.numworkers)):
            config_path = os.path.join(self.cluster_dir, 'worker-%d.json' % i)
            config = {'worker_port': self.internal_worker_port,
                      'registry_host': self.registry_ip,
                      'registry_port': self.internal_reg_port,
                      'type': 'worker',
                      'cluster_name': self.cluster,
                      'registry': registry}

            #TODO: clean this up -> should we actually use whole cluster as arg to pullclient?
            if registry == "olregistry":
                rethinkdb = rdjs(os.path.join(self.cluster_dir, 'rethinkdb.json'))
                cluster = ['%s:%s' % (rethinkdb['host_ip'], rethinkdb['host_port'])]

                config['reg_cluster'] = cluster
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

            print 'Started worker container "localhost:%s"' % config['host_port']
            self.workers.append(config)

    def start_lb(self):
        nginx_path = os.path.join(self.script_dir, 'nginx.config')

        self.write_nginx_config(nginx_path)
        c = 'docker run -p 0:%s -v %s:/etc/nginx/nginx.conf:ro -d nginx:1.11.1' % (self.internal_lb_port, nginx_path)
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

        print 'Started loadbalancer container "localhost:%s"' % self.balancer_port

    def write_nginx_config(self, path):
        config = 'http {\n\tupstream handlers {\n'

        for worker in self.workers:
            config += '\t\tserver %s:%s;\n' % (worker['ip'], worker['worker_port'])

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
                print 'Waiting for first rethinkdb instance to come up'

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
        pass

    def start_workers(self):
        pass

    def start_lb(self):
        pass

    def rethinkdb_wait(self):
        pass

    def print_directions(self):
        pass
