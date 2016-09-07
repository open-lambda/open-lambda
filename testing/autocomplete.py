#! /usr/bin/env python
import os, sys, random, string, requests
from common import *

def main():
    setup = os.path.join(SCRIPT_DIR, '..', 'util', 'setup.py')
    print run(setup + ' -c test_cluster -d autocomplete -f autocomplete.py')

    path = os.path.join(SCRIPT_DIR, '..', 'applications', 'autocomplete', 'static', 'config.json')
    config = rdjs(path)

    url = config['url']
    print 'POST ' + url
    args = json.dumps({"op":"keystroke", "pref":"ab"})
    r = requests.post(url, data=args, timeout = 30)
    print 'RESP ' + r.text
    r = r.json()
    if r['result'] == ["about", "above", "able", "ability", "absence"]:
        print 'PASS'
    else:
        print 'FAIL'
        sys.exit(1)

if __name__ == '__main__':
    main()

