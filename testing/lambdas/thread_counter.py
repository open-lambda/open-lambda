from threading import *
import time, sys

t = None

def worker():
    counter = 0
    while True:
        print 'counter=%d' % counter
        sys.stdout.flush()
        counter += 1
        time.sleep(0.001)

def handler(db_conn, event):
    global t
    if t == None:
        print 'Init worker thread'
        t = Thread(target=worker)
        t.start()
    time.sleep(0.1)
    return 'Background thread started'

def main():
    print handler(None, None)

if __name__ == '__main__':
    main()
