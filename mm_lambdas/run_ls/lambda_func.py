from subprocess import call

def handler(conn, event):
    try:
        f = open("x", 'w')
        call(["ls"], stdout=f)
        f.close()
        f = open("x")
        return f.read()
    except Exception as e:
        return {'error': str(e)} 
