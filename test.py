#!/usr/bin/env python3
import os, sys, json, time, requests
from subprocess import Popen

echo_py = """
def handler(event):
    return event
"""

OLDIR = 'test-cluster'


def put_code(name, code):
    with open(os.path.join(OLDIR, "registry", name+".py"), "w") as f:
        f.write(code.lstrip())


def run(cmd):
    print("RUN", " ".join(cmd))
    p = Popen(cmd, stdout=sys.stdout, stderr=sys.stderr)
    rc = p.wait()
    if rc:
        raise Exception("command failed: " + " ".join(cmd))


def test1():
    run(['./bin/ol', 'worker', '-p='+OLDIR, '--detach'])
    put_code("echo", echo_py)
    for i in range(100):
        r = requests.post("http://localhost:5000/run/echo", data='"hello world"')
        r.raise_for_status()
        print(r.text)
    run(['./bin/ol', 'kill', '-p='+OLDIR])


def main():
    if os.path.exists(OLDIR):
        try:
            run(['./bin/ol', 'kill', '-p='+OLDIR])
        except:
            print('could not kill cluster')
        run(['rm', '-rf', OLDIR])
    run(['./bin/ol', 'new', '-p='+OLDIR])
    test1()


if __name__ == '__main__':
    main()
