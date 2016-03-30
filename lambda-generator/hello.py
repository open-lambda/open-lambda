import rethinkdb as r

def handler(db_conn, event):
    servers = r.db('rethinkdb').table('server_status').run(db_conn)
    server_count = len(list(servers))
    return "I'm in a cluster with %d rethinkdb servers" % server_count
