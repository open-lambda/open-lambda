#!/usr/bin/env python3
import os, sys, json, time, requests, copy, traceback
from collections import OrderedDict
from subprocess import Popen
from contextlib import contextmanager        
                

OLDIR = 'test-cluster'

results = OrderedDict({"runs": []})
curr_conf = None


def test(fn):
    def wrapper(*args, **kwargs):
        rc = None
        result = OrderedDict()
        result["test"] = fn.__name__
        result["pass"] = None
        result["seconds"] = None
        result["conf"] = curr_conf
        result["exception"] = None
        result["worker_tail"] = None

        t0 = time.time()
        try:
            rc = fn(*args, **kwargs)
            result["pass"] = True
        except Exception:
            result["pass"] = False
            result["exception"] = traceback.format_exc().split("\n")
        t1 = time.time()
        result["seconds"] = t1-t0
            
        with open(os.path.join(OLDIR, "worker.out")) as f:
            result["worker_tail"] = f.read().split("\n")[-20:]

        results["runs"].append(result)
        print(json.dumps(result, indent=2))
        return rc

    return wrapper


def put_conf(conf):
    global curr_conf
    with open(os.path.join(OLDIR, "config.json"), "w") as f:
        json.dump(conf, f, indent=2)
    curr_conf = conf

        
@contextmanager
def TestConf(launch_worker=True, **keywords):
    with open(os.path.join(OLDIR, "config.json")) as f:
        orig = json.load(f)
    new = copy.deepcopy(orig)
    for k in keywords:
        if not k in new:
            raise Exception("unknown config param: %s" % k)
        new[k] = keywords[k]

    # setup
    print("PUSH conf:", keywords)
    put_conf(new)
    if launch_worker:
        run(['./bin/ol', 'worker', '-p='+OLDIR, '--detach'])
    yield new

    # cleanup
    print("POP conf:", keywords)
    if launch_worker:
        run(['./bin/ol', 'kill', '-p='+OLDIR])
    put_conf(orig)


def run(cmd):
    print("RUN", " ".join(cmd))
    p = Popen(cmd, stdout=sys.stdout, stderr=sys.stderr)
    rc = p.wait()
    if rc:
        raise Exception("command failed: " + " ".join(cmd))


@test
def test_smoke_echo():
    msg = '"hello world"'
    r = requests.post("http://localhost:5000/run/echo", data=msg)
    if r.status_code != 200:
        raise Exception("STATUS %d: %s" % (r.status_code, r.text))
    assert r.text == msg


@test
def test_smoke_install(num=None):
    name = "install"
    if num != None:
        name += str(num)
    r = requests.post("http://localhost:5000/run/"+name, data="{}")
    if r.status_code != 200:
        raise Exception("STATUS %d: %s" % (r.status_code, r.text))
    assert r.json() == "imported"
    

def smoke_tests():
    test_smoke_echo()
    test_smoke_install()
    test_smoke_install(2)
    test_smoke_install(3)


def main():
    # general setup
    if os.path.exists(OLDIR):
        try:
            run(['./bin/ol', 'kill', '-p='+OLDIR])
        except:
            print('could not kill cluster')
        run(['rm', '-rf', OLDIR])
    run(['./bin/ol', 'new', '-p='+OLDIR])

    # run tests with various configs
    startup_pkgs = ["parso", "jedi", "urllib3", "idna", "chardet", "certifi", "requests", "simplejson"]

    with TestConf(launch_worker=False, registry=os.path.abspath("test-registry"),
                  startup_pkgs=startup_pkgs):
        with TestConf(sandbox="sock", handler_cache_size=0, import_cache_size=0, cg_pool_size=10):
            smoke_tests()
        with TestConf(sandbox="sock", handler_cache_size=10000000, import_cache_size=0, cg_pool_size=10):
            smoke_tests()
        with TestConf(sandbox="sock", handler_cache_size=0, import_cache_size=10000000, cg_pool_size=10):
            smoke_tests()
        with TestConf(sandbox="sock", handler_cache_size=10000000, import_cache_size=10000000, cg_pool_size=10):
            smoke_tests()
        with TestConf(sandbox="docker", handler_cache_size=0, import_cache_size=0, cg_pool_size=0):
            smoke_tests()
        with TestConf(sandbox="docker", handler_cache_size=10000000, import_cache_size=0, cg_pool_size=0):
            smoke_tests()

    # save test results
    passed = len([t for t in results["runs"] if t["pass"]])
    failed = len([t for t in results["runs"] if not t["pass"]])
    results["passed"] = passed
    results["failed"] = failed
    print("PASSED: %d, FAILED: %d" % (passed, failed))

    with open("test.json", "w") as f:
        json.dump(results, f, indent=2)

    sys.exit(failed)


if __name__ == '__main__':
    main()
