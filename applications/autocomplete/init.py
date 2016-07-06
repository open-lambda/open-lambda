import rethinkdb as r
import os, os.path, argparse
from common import *
AC = 'ac' # DB
WORDS = 'words' # TABLE
WORD = 'word' # COLUMN
FREQ = 'freq' # COLUMN
NGINX_EXAMPLE = 'docker run -d -p 80:80 -v %s:/usr/share/nginx/html:ro nginx'

def makeDB(host):
    conn = r.connect(host, 28015)
    dbs = r.db_list().run(conn)
    if AC in dbs:
        return 'already there'
        #r.db_drop(AC).run(conn)
    r.db_create(AC).run(conn)
    r.db(AC).table_create(WORDS, primary_key = WORD).run(conn)
    ra = {WORD: None, FREQ: None}
    f = open(os.path.join(SCRIPT_DIR, "wordsCSV.txt"), 'r')
    for line in f:
        line = line.strip()
        linesplit = line.split(',')
        w = linesplit[0]
        ra[WORD] = unicode(w)
        ra[FREQ] = int(linesplit[1])
        if len(linesplit[0]) == 1:
            print linesplit[0]
        r.db(AC).table(WORDS).insert(ra).run(conn)
    f.close()
    return 'initialized'

parser = argparse.ArgumentParser()
parser.add_argument('--cluster', '-c')
args = parser.parse_args()
cluster_dir = os.path.join(SCRIPT_DIR, "..", "..","util", args.cluster)
worker0 = rdjs(os.path.join(cluster_dir, 'worker-0.json'))
msg = makeDB(worker0['ip'])
print msg
