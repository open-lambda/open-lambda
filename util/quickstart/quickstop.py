#!/usr/bin/env python
import os, signal
from common import rdjs, run

def main():
    SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
    cluster = rdjs(os.path.join(SCRIPT_DIR, 'cluster.json'))

    try:
        run('docker kill %s' % cluster['registry'])
        print "Registry killed"
    except:
        print "Registry container not running or bad CID: %s" % cluster['registry']

    try:
        run('pkill -9 rethinkdb', False)
        print "RethinkDB killed"
    except:
        print "RethinkDB not running"

    try:
        os.kill(cluster['worker'], signal.SIGINT)
        print "Worker killed"
    except:
        print "Worker not running or bad PID: %s" % cluster['worker']

    rethinkdb_data = os.path.join(SCRIPT_DIR, 'rethinkdb_data')
    run('rm -rf %s' % rethinkdb_data)
    print "RethinkDB data directory removed"

    run('rm -rf /tmp/handlers')
    print "Handler directories removed"

if __name__ == "__main__":
    main()
