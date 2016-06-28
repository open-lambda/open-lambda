#!/usr/bin/python
import os, requests, time, json
from pssh import ParallelSSHClient

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

    hosts = [ip]
    client = ParallelSSHClient(hosts)

    client.copy_file('test.sh', '/tmp/test.sh')

    output = client.run_command('bash /tmp/test.sh', sudo=True)
    output = output.values()[0]
    for l in output["stdout"]:
        print l
    for l in output["stderr"]:
        print l

    # make sure we cleanup everything!
    kill()

if __name__ == '__main__':
    main()
