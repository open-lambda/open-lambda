import time, traceback
import rethinkdb as r

TIX   = 'tix'   # DB
SEATS = 'seats' # TABLE
TS    = 'ts'    # COLUMN
SNUM  = 'snum'  # COLUMN
STAT  = 'stat'  # COLUMN

def init(conn, event):
    # try to drop table (may or may not exist)
    rv = ''
    try:
        r.db_drop(TIX).run(conn)
        rv = 'dropped, then created'
    except:
        rv = 'created'
    r.db_create(TIX).run(conn)
    r.db(TIX).table_create(SEATS, primary_key = SNUM).run(conn)
    r.db(TIX).table(SEATS).index_create(TS).run(conn)

    for x in range(1,11):
        rv += str(r.db(TIX).table(SEATS).insert({SNUM: x,
                                       STAT: 'free',
                                       TS:   time.time()}).run(conn))

    return rv

def hold(conn, event):
    snum = event.get('snum')
    if r.db(TIX).table(SEATS).get(snum).get_field(STAT).run(conn) != 'free':
        return 'seat %f not free' % snum

    r.db(TIX).table(SEATS).get(snum).update({STAT: 'held',
                                     TS:   time.time()}).run(conn)
    return 'held %f' % snum

def book(conn, event):
    snum = event.get('snum')
    if r.db(TIX).table(SEATS).get(snum).get_field(STAT).run(conn) != 'held':
        return 'seat %f not held' % snum

    r.db(TIX).table(SEATS).get(snum).update({STAT: 'booked',
                                     NAME: name,
                                     TS:   time.time()}).run(conn)
    return 'booked %f' % (snum)

def updates(conn, event):
    ts = event.get('ts', 0)
    for row in (r.db(TIX).table(SEATS).filter(r.row[TS] > ts).
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
