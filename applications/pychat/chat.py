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
    for row in (r.db(CHAT).table(MSGS).filter(r.row[TS] > ts).
                changes(include_initial=True).run(conn)):
        return row['new_val']

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
