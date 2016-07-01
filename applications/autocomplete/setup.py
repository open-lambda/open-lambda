#!/usr/bin/env python
import os, sys, random, string
from common import *
import rethinkdb as r
AC = 'ac' # DB
WORDS = 'words' # TABLE
LINE = 'line' # COLUMN
WORD = 'word' # COLUMN
FREQ = 'freq' # COLUMN
PREFS = 'prefs' # TABLE
PREF = 'pref' # COLUMN
LOWER = 'lower' # COLUMN
UPPER = 'upper' # COLUMN
NGINX_EXAMPLE = 'docker run -d -p 80:80 -v %s:/usr/share/nginx/html:ro nginx'

def makeDB(host):
    conn = r.connect(host, 28015)
    dbs = r.db_list().run(conn)
    if AC in dbs:
        return 'already there'
    r.db_create(AC).run(conn)
    r.db(AC).table_create(WORDS, primary_key = LINE).run(conn)
    r.db(AC).table_create(PREFS, primary_key = PREF).run(conn)
    ra = {LINE: None, WORD: None, FREQ: None}
    f = open(os.path.join(SCRIPT_DIR, "wordsCSV.txt"), 'r')
    for line in f:
        line = line.strip()
        linesplit = line.split(',')
        ra[LINE] = int(linesplit[0])
        ra[WORD] = linesplit[1]
        ra[FREQ] = int(linesplit[2])
        if int(linesplit[0]) % 5000 == 0:
            print linesplit[0]
        r.db(AC).table(WORDS).insert(ra).run(conn)
    f.close()
    print "========================"
    g = open(os.path.join(SCRIPT_DIR, "rangesCSV.txt"), 'r')
    ra = {PREF: None, LOWER: None, UPPER: None}
    for line in g:
        line = line.strip()
        linesplit = line.split(',')
        ra[PREF] = linesplit[0]
        ra[LOWER] = int(linesplit[1])
        ra[UPPER] = int(linesplit[2])
        if len(linesplit[0]) == 1:
            print linesplit[0]

        r.db(AC).table(PREFS).insert(ra).run(conn)
    g.close()
    return 'initialized'


def main():
    app_name = ''.join(random.choice(string.ascii_lowercase) for _ in range(12))
    static_dir = os.path.join(SCRIPT_DIR, 'static')
    root_dir = os.path.join(SCRIPT_DIR, '..', '..')
    cluster_dir = os.path.join(root_dir, 'util', 'cluster')
    builder_dir = os.path.join(root_dir, 'lambda-generator')
    if not os.path.exists(cluster_dir):
        return 'cluster not running'

    # build image
    print '='*40
    print 'Building image'
    builder = os.path.join(builder_dir, 'builder.py')
    run(builder + ' -n %s -l %s -c %s -e %s' %
        (app_name,
         os.path.join(SCRIPT_DIR, 'autocomplete.py'),
         os.path.join(SCRIPT_DIR, 'lambda-config.json'),
         os.path.join(SCRIPT_DIR, 'environment.json')))

    # push image
    print '='*40
    print 'Pushing image'
    registry = rdjs(os.path.join(cluster_dir, 'registry.json'))
    img = 'localhost:%s/%s' % (registry['host_port'], app_name)
    run('docker tag -f %s %s' % (app_name, img))
    run('docker push ' + img)

    # setup config
    worker0 = rdjs(os.path.join(cluster_dir, 'worker-0.json'))
    balancer = rdjs(os.path.join(cluster_dir, 'loadbalancer-1.json'))
    config_file = os.path.join(static_dir, 'config.json')
    url = ("http://%s:%s/runLambda/%s" %
           (balancer['host_ip'], balancer['host_port'], app_name))
    wrjs(config_file, {'url': url})

    #init DB
    print '='*40
    print 'Init DB'
    makeDB(worker0['ip'])

    # directions
    print '='*40
    print 'Consider serving the app with nginx as follows:'
    print NGINX_EXAMPLE % static_dir
    return None

if __name__ == '__main__':
    rv = main()
    if rv != None:
        print 'ERROR: ' + rv
        sys.exit(1)
    sys.exit(0)
