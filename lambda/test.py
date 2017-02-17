import os, time

def main():
    pid = os.getpid()
    with open('/proc/self/task/%d/comm' % pid, 'w') as f:
        f.write('hello_world')
    time.sleep(100000)

if __name__ == '__main__':
    main()
