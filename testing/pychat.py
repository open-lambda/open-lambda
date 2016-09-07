#!/usr/bin/env python
import os, sys, random, string, requests, time
from common import *

def main():
    setup = os.path.join(SCRIPT_DIR, '..', 'util', 'setup.py')
    print run(setup+' -c test_cluster -d pychat -f lambda_func.py')

    path = os.path.join(SCRIPT_DIR, '..', 'applications', 'pychat', 'static', 'config.json')
    config = rdjs(path)

    url = config['url']
    print 'POST ' + url
    args = json.dumps({"op": "msg", "msg": "hello"})
    cmd = "curl -X POST %s -d '%s'" % (config['url'], args)
    print run(cmd, False)
    r = requests.post(url, data=args)
    print 'RESP ' + r.text
    r = r.json()
    if r['result'].startswith('insert'):
        print 'PASS'
    else:
        print 'FAIL'
        sys.exit(1)
    sys.exit(1)

if __name__ == '__main__':
    main()
