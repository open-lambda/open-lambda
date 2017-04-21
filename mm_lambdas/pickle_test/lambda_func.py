import cPickle as pickle

def handler(conn, event):
    try:
        foo = { "turn" : "pow", "weem" : "wam" }
        foo_p = pickle.dumps(foo)
        un_p = pickle.loads(foo_p)
        return "Recovered dict: " + str(un_p)
    except Exception as e:
        return {'error': str(e)} 
