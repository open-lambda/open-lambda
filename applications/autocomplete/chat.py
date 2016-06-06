import time, traceback
import rethinkdb as r

CHAT = 'chat'     # DB
MSGS = 'messages' # TABLE
TS   = 'ts'       # COLUMN
MSG  = 'msg'      # COLUMN

def init(conn, event):
    # try to drop table (may or may not exist)
    rv = ''
    try:
        r.db_drop(CHAT).run(conn)
        rv = 'dropped, then created'
    except:
        rv = 'created'
    r.db_create(CHAT).run(conn);
    r.db(CHAT).table_create(MSGS).run(conn);
    r.db(CHAT).table(MSGS).index_create(TS).run(conn)

    return rv

def msg(conn, event):
    ts = time.time()
    r.db(CHAT).table(MSGS).insert({MSG: event['msg'],
                                   TS:  ts}).run(conn)
    return 'insert %f complete' % ts

def updates(conn, event):
    ts = event.get('ts', 0)
    for row in (r.db(CHAT).table(MSGS).filter(r.row[TS] > ts).
                changes(include_initial=True).run(conn)):
        return row['new_val']

def handler(conn, event):
    fn = {'init':    init,
          'msg':     msg,
          'updates': updates}.get(event['op'], None)
    if fn != None:
        try:
            result = fn(conn, event)
            return {'result': result}
        except Exception:
            return {'error': traceback.format_exc()}
    else:
        return {'error': 'bad op'}
