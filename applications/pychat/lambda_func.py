import time, traceback
import rethinkdb as r

CHAT = 'chat'     # DB
MSGS = 'messages' # TABLE
TS   = 'ts'       # COLUMN
MSG  = 'msg'      # COLUMN

def msg(conn, event):
    ts = time.time()
    r.db(CHAT).table(MSGS).insert({MSG: event['msg'],
                                   TS:  ts}).run(conn)
    return 'insert %f complete' % ts

def updates(conn, event):
    ts = event.get('ts', 0)
    rows = list(r.db(CHAT).table(MSGS).filter(r.row[TS] > ts).run(conn))
    if len(rows) == 0:
        wait(conn, ts)
        rows = list(r.db(CHAT).table(MSGS).filter(r.row[TS] > ts).run(conn))
        assert(len(rows) > 0)
    
    rows.sort(key=lambda row: row[TS])
    return rows

# TODO: have timeout
def wait(conn, ts):
    for row in (r.db(CHAT).table(MSGS).filter(r.row[TS] > ts).
                changes(include_initial=True).run(conn)):
        break

def handler(conn, event):
    fn = {'msg':     msg,
          'updates': updates}.get(event['op'], None)
    if fn != None:
        try:
            result = fn(conn, event)
            return {'result': result}
        except Exception:
            return {'error': traceback.format_exc()}
    else:
        return {'error': 'bad op'}
