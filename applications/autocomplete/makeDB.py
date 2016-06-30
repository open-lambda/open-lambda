import rethinkdb as r
import os, os.path
from common import *
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
        #r.db_drop(AC).run(conn)
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

msg = makeDB("localhost")
print msg
