import os, time, ns

def listen():
    with open('/tmp/fifo', 'rw') as f:
        return f.read()

def main():
    while True:
        pid = listen().strip()
        args = listen()
        r = ns.forkenter(pid)
        if r == 0:
            break
    os.system(args)
        

if __name__ == '__main__':
    main()
