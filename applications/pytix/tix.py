import time, traceback, json
import rethinkdb as r

TIX  = 'tix'  # DB
VENU = 'venu' # TABLE
ID   = 'id'   # COLUMN
TS   = 'ts'   # COLUMN
SMAP = 'smap' # COLUMN
UMAP = 'umap' # COLUMN
MAX  = 'max'  # COLUMN
CNT  = 20     # number of seats

def init(conn, event):
    # try to drop table (may or may not exist)
    rv = ''
    try:
        r.db_drop(TIX).run(conn)
        rv = 'dropped, then created'
    except:
        rv = 'created'
    r.db_create(TIX).run(conn)
    r.db(TIX).table_create(VENU).run(conn)
    r.db(TIX).table(VENU).index_create(TS).run(conn)

    smap = {}
    umap = {}
    for x in range(1, CNT + 1):
        smap[str(x)] = 'free' 
        umap[str(x)] = ''

    rv += str(r.db(TIX).table(VENU).insert({
        ID: 0,
        SMAP: smap,
        UMAP: umap,
        MAX: CNT,
        TS: time.time()
    }).run(conn))

    return rv

def hold(conn, event):
    snum = str(event.get('snum'))
    unum = str(event.get('unum'))

    smap = {}
    umap = {}
    smap = r.db(TIX).table(VENU).get(0).get_field(SMAP).run(conn)
    umap = r.db(TIX).table(VENU).get(0).get_field(UMAP).run(conn)
    smap[snum] = 'held'
    umap[snum] = unum
    result = r.db(TIX).table(VENU).get(0).update(lambda VENU:
        r.branch(
            VENU[SMAP][snum] == 'free',
            {SMAP: smap, UMAP: umap, TS: time.time()},
            {}
        )
    ).run(conn)
    if result:
        return result

def book(conn, event):
    unum = str(event.get('unum'))

    smap = {}
    umap = {}
    smap = r.db(TIX).table(VENU).get(0).get_field(SMAP).run(conn)
    umap = r.db(TIX).table(VENU).get(0).get_field(UMAP).run(conn)
    for x in range (1, CNT + 1):
        if smap[str(x)] == 'held' and umap[str(x)] == unum:
            smap[str(x)] = 'booked'

    result = r.db(TIX).table(VENU).get(0).update({
        SMAP: smap,
        TS: time.time()
    }).run(conn)

    if result:
        return result

def updates(conn, event):
    ts = event.get('ts', 0)
    for row in (r.db(TIX).table(VENU).filter(r.row[TS] > ts).
                changes(include_initial=True).run(conn)):
        return row['new_val']

def handler(conn, event):
    fn = {'init':    init,
          'hold':    hold,
          'book':    book,
          'updates': updates}.get(event['op'], None)
    if fn != None:
        try:
            result = fn(conn, event)
            return {'result': result}
        except Exception:
            return {'error': traceback.format_exc()}
    else:
        return {'error': 'bad op'}
