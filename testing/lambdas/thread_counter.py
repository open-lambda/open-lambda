from threading import *
import time

t = None
counter = 0
m = Lock()

def worker():
    global counter
    while True:
        m.acquire()
        counter += 1
        m.release()
        time.sleep(0.01)

def handler(db_conn, event):
    global t, counter
    if t == None:
        print 'Init worker thread'
        t = Thread(target=worker)
        t.start()
    m.acquire()
    _counter = counter
    m.release()
    return _counter

def main():
    print handler(None, None)
    time.sleep(1)
    print handler(None, None)

if __name__ == '__main__':
    main()
