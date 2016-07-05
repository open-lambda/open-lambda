#!/usr/bin/env python
import os, sys, random, string, requests
from common import *

HEADERS = {
    "Content-Type": "application/json"
}

def main():
    setup = os.path.join(SCRIPT_DIR, '..', 'util', 'setup.py')
    print run(setup+' -c test_cluster -d pychat -f chat.py')

    path = os.path.join(SCRIPT_DIR, '..', 'applications', 'pychat', 'static', 'config.json')
    config = rdjs(path)

    url = config['url']
    print 'POST ' + url
    args = {"op": "init"}
    r = requests.post(url, data=json.dumps(args), headers=HEADERS)
    r = r.json()
    print 'RESP ' + str(r)
    if r.get('result', None) == 'created':
        print 'PASS'
    else:
        print 'FAIL'

if __name__ == '__main__':
    main()
