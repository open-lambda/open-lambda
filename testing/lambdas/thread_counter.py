from threading import *
import time

t = None
ms = 0
m = Lock()

def worker():
    global ms
    while True:
        m.acquire()
        ms += 10
        m.release()
        time.sleep(0.01)

def handler(db_conn, event):
    global t, ms
    if t == None:
        print 'Init worker thread'
        t = Thread(target=worker)
        t.start()
    m.acquire()
    _ms = ms
    m.release()
    return _ms

def main():
    print handler(None, None)
    time.sleep(1)
    print handler(None, None)

if __name__ == '__main__':
    main()
