def hello(conn, event):
    return "Hello from inside a Lambda!"

def handler(conn, event):
    fn = {'hello': hello}.get(event['op'], None)

    if fn != None:
        try:
            result = fn(conn, event)
            return {'result': result}
        except Exception:
            return {'error': traceback.format_exc()}
    else:
        return {'error': 'bad op'}
