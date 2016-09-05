#!/usr/bin/env python
import os, subprocess, json, random, string, time
import netifaces
import rethinkdb as r
from common import *

CLIENT_PORT = '28015'
CLUSTER_PORT = '29015'
HTTP_PORT = '8080'
REGISTRY_PORT = '5000'
WORKER_PORT = '8090'

def lookup_host_port(cid, guest_port):
    inspect = run_js('docker inspect '+cid)
    return only(only(inspect)['NetworkSettings']['Ports'][str(guest_port)+'/tcp'])['HostPort']

def main():
    SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
    cluster = {}

    # Get host IP
    try :
        my_ip = netifaces.ifaddresses('eth0')[netifaces.AF_INET][0]['addr']
    except:
        print 'Could not find an IP using netifaces, using 127.0.0.1'
        my_ip = '127.0.0.1'

    # Pull image for the lambdas
    c = 'docker pull eoakes/lambda:latest'
    run(c, True)

    # Start rethinkdb for the registry 
    rethink_dir = os.path.join(SCRIPT_DIR, "rethinkdb_data")
    c = 'rethinkdb --bind all -d %s' % rethink_dir
    cluster['rethinkdb'] = runbg(c, True)

    print 'Started rethinkdb on localhost (client port: %s, cluster port: %s, http port: %s)' % (CLIENT_PORT, CLUSTER_PORT, HTTP_PORT)
    print '='*40

    # Start the container for the registry
    c = 'docker run -d -p 0:%s olregistry /open-lambda/registry %s %s:%s' % (REGISTRY_PORT, REGISTRY_PORT, my_ip, CLIENT_PORT)
    reg_cid = run(c).strip()
    cluster['registry'] = reg_cid
    reg_port = lookup_host_port(reg_cid, REGISTRY_PORT)

    # Start worker
    registry = "olregistry"

    config_path = os.path.join(SCRIPT_DIR, 'quickstart.json')
    worker = os.path.join(SCRIPT_DIR, '..', '..', 'node', 'bin', 'worker')

    config = {}
    config['registry'] = 'olregistry'
    config['reg_cluster'] = ['localhost:%s' % CLIENT_PORT]
    wrjs(config_path, config, atomic=False)

    c = '%s %s' % (worker, config_path)
    cluster['worker'] = runbg(c, True)

    # Write cluster information to json file
    cluster_path = os.path.join(SCRIPT_DIR, 'cluster.json')
    wrjs(cluster_path, cluster, atomic=False)

    print 'Started worker process localhost:%s' % WORKER_PORT
    print '='*40

    # Wait for rethinkdb to come up
    # TODO: better way to do this
    for i in range(10):
        try:
            r.connect('127.0.0.1', 28015).repl()
        except:
            print 'Waiting for rethinkdb to come up'

        time.sleep(1)

    # Push code to registry
    app_name = ''.join(random.choice(string.ascii_lowercase) for _ in range(12))
    regpush = os.path.join(SCRIPT_DIR, '..', 'regpush')
    lambdaFn = os.path.join(SCRIPT_DIR, '..', '..', 'applications', 'hello', 'handler.tar.gz')
    c = '%s localhost:%s %s %s' % (regpush, reg_port, app_name, lambdaFn)
    run(c, True)
    print '='*40

    # Show them the curl command
    print 'Send requests directly to the "hello" worker as follows:'
    print "IMG=%s && curl -w \"\\n\" -X POST localhost:%s/runLambda/$IMG -d '{\"op\":\"hello\"}'" % (app_name, WORKER_PORT)
    print '='*40

if __name__ == "__main__":
    main()
