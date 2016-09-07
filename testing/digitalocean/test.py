#!/usr/bin/python
import os, requests, time, json, argparse

API = "https://api.digitalocean.com/v2/droplets"
DROPLET_NAME = "ol-tester"
HEADERS = {
    "Authorization": "Bearer "+os.environ['TOKEN'],
    "Content-Type": "application/json"
}

def post(args):
    r = requests.post(API, data=args, headers=HEADERS)
    return r.json()

def get(args):
    r = requests.get(API, data=args, headers=HEADERS)
    return r.json()

def start():
    r = requests.get("https://api.digitalocean.com/v2/account/keys", headers=HEADERS)
    keys = map(lambda row: row['id'], r.json()['ssh_keys'])

    args = {
        "name":DROPLET_NAME,
        "region":"nyc2",
        "size":"512mb",
        "image":"ubuntu-14-04-x64",
        "ssh_keys":keys
    }
    r = requests.post(API, data=json.dumps(args), headers=HEADERS)
    return r.json()

def kill():
    args = {}
    droplets = get(args)['droplets']
    for d in droplets:
        if d['name'] == DROPLET_NAME:
            print 'Deleting %s (%d)' % (d['name'], d['id'])
            print requests.delete(API+'/'+str(d['id']), headers=HEADERS)

def lookup(droplet_id):
    r = requests.get(API+'/'+str(droplet_id), headers=HEADERS)
    return r.json()['droplet']
    
def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--quickstart', default=False, action='store_true')
    args = parser.parse_args()

    global TEST_SCRIPT
    if args.quickstart:
        TEST_SCRIPT = "qs_test.sh"
    else:
        TEST_SCRIPT = "test.sh"

    # cleanup just in case
    kill()

    # create new droplet and wait for it
    droplet = start()['droplet']
    print droplet

    while True:
        droplet = lookup(droplet['id'])

        # status
        s = droplet['status']
        assert(s in ['active', 'new'])

        # addr
        ip = None
        for addr in droplet["networks"]["v4"]:
            if addr["type"] == "public":
                ip = addr["ip_address"]
        
        print 'STATUS: %s, IP: %s' % (str(s), str(ip))
        if s == 'active' and ip != None:
            break

        time.sleep(3)

    time.sleep(30) # give SSH some time

    scp = 'scp -o "StrictHostKeyChecking no" %s root@%s:/tmp' % (TEST_SCRIPT, ip)
    print 'RUN ' + scp
    rv = os.system(scp)
    assert(rv == 0)

    cmds = 'bash /tmp/%s' % TEST_SCRIPT
    ssh = 'echo "<CMDS>" | ssh -o "StrictHostKeyChecking no" root@<IP>'
    ssh = ssh.replace('<CMDS>', cmds).replace('<IP>', ip)
    print 'RUN ' + ssh
    rv = os.system(ssh)
    assert(rv == 0)

    # make sure we cleanup everything!
    kill()

if __name__ == '__main__':
    main()
