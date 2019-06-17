#!/usr/bin/env python3
import os, sys, json, time, requests, copy, traceback, tempfile
from collections import OrderedDict
from subprocess import Popen, check_output
from multiprocessing import Pool
from contextlib import contextmanager        
                

OLDIR = 'test-cluster'

results = OrderedDict({"runs": []})
curr_conf = None


def test(fn):
    def wrapper(*args, **kwargs):
        if len(args):
            raise Exception("positional args not supported for tests")

        result = OrderedDict()
        result["test"] = fn.__name__
        result["params"] = kwargs
        result["pass"] = None
        result["seconds"] = None
        result["total_seconds"] = None
        result["stats"] = None
        result["conf"] = curr_conf
        result["exception"] = None
        result["worker_tail"] = None

        total_t0 = time.time()
        try:           
            # setup worker
            mounts0 = mount_count()
            run(['./ol', 'worker', '-p='+OLDIR, '--detach'])

            # run test/benchmark
            test_t0 = time.time()
            rv = fn(**kwargs)
            test_t1 = time.time()
            result["seconds"] = test_t1 - test_t0

            # cleanup worker
            run(['./ol', 'kill', '-p='+OLDIR])
            mounts1 = mount_count()

            if mounts0 != mounts1:
                raise Exception("mounts are leaking (%d before, %d after)" % (mounts0, mounts1))

            result["pass"] = True
        except Exception:
            rv = None
            result["pass"] = False
            result["exception"] = traceback.format_exc().split("\n")
        total_t1 = time.time()
        result["total_seconds"] = total_t1-total_t0
        result["stats"] = rv

        with open(os.path.join(OLDIR, "worker.out")) as f:
            result["worker_tail"] = f.read().split("\n")
            if result["pass"]:
                # truncate because we probably won't use it for debugging
                result["worker_tail"] = result["worker_tail"][-10:]

        results["runs"].append(result)
        print(json.dumps(result, indent=2))
        return rv

    return wrapper


def put_conf(conf):
    global curr_conf
    with open(os.path.join(OLDIR, "config.json"), "w") as f:
        json.dump(conf, f, indent=2)
    curr_conf = conf


def mount_count():
    output = check_output(["mount"])
    output = str(output, "utf-8")
    output = output.split("\n")
    return len(output)

        
@contextmanager
def TestConf(**keywords):
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

    yield new

    # cleanup
    print("POP conf:", keywords)
    put_conf(orig)


def run(cmd):
    print("RUN", " ".join(cmd))
    p = Popen(cmd, stdout=sys.stdout, stderr=sys.stderr)
    rc = p.wait()
    if rc:
        raise Exception("command failed: " + " ".join(cmd) + ", ")


@test
def smoke_tests():
    msg = '"hello world"'
    r = requests.post("http://localhost:5000/run/echo", data=msg)
    if r.status_code != 200:
        raise Exception("STATUS %d: %s" % (r.status_code, r.text))
    assert r.text == msg

    for i in range(3):
        name = "install"
        if i != 0:
            name += str(i+1)
        r = requests.post("http://localhost:5000/run/"+name, data="{}")
        if r.status_code != 200:
            raise Exception("STATUS %d: %s" % (r.status_code, r.text))
        assert r.json() == "imported"    


def stress_one_lambda_task(args):
    t0, seconds = args
    i = 0
    while time.time() < t0 + seconds:
        r = requests.post("http://localhost:5000/run/echo", data=str(i))
        r.raise_for_status()
        assert r.text == str(i)
        i += 1
    return i


@test
def stress_one_lambda(procs, seconds):
    t0 = time.time()

    with Pool(procs) as p:
        reqs = sum(p.map(stress_one_lambda_task, [(t0, seconds)] * procs, chunksize=1))

    return {"reqs_per_sec": reqs/seconds}


@test
def call_each_once_exec(lambda_count, alloc_mb):
    # TODO: do in parallel
    t0 = time.time()
    for i in range(lambda_count):
        r = requests.post("http://localhost:5000/run/L%d"%i, data=json.dumps({"alloc_mb": alloc_mb}))
        r.raise_for_status()
        assert r.text == str(i)
    seconds = time.time() - t0

    return {"reqs_per_sec": lambda_count/seconds}


@test
def fork_bomb():
    limit = curr_conf["sock_cgroups"]["max_procs"]
    r = requests.post("http://localhost:5000/run/fbomb", data=json.dumps({"times": limit*2}))
    r.raise_for_status()
    # the function returns the number of children that we were able to fork
    actual = int(r.text)
    assert(1 <= actual <= limit)


def call_each_once(lambda_count, alloc_mb=0):
    with tempfile.TemporaryDirectory() as reg_dir:
        # create dummy lambdas
        for i in range(lambda_count):
            with open(os.path.join(reg_dir, "L%d.py"%i), "w") as f:
                f.write("def handler(event):\n")
                f.write("    global s\n")
                f.write("    s = '*' * %d * 1024**2\n" % alloc_mb)
                f.write("    return %d\n" % i)

        with TestConf(registry=reg_dir):
            call_each_once_exec(lambda_count=lambda_count, alloc_mb=alloc_mb)


def tests():
    startup_pkgs = ["parso", "jedi", "urllib3", "idna", "chardet", "certifi", "requests", "simplejson"]
    test_reg = os.path.abspath("test-registry")

    with TestConf(registry=test_reg, startup_pkgs=startup_pkgs):
        # do smoke tests under various configs
        with TestConf(handler_cache_mb=0, import_cache_mb=0):
            smoke_tests()
        with TestConf(handler_cache_mb=256, import_cache_mb=0):
            smoke_tests()
        with TestConf(handler_cache_mb=0, import_cache_mb=256):
            smoke_tests()
        with TestConf(handler_cache_mb=256, import_cache_mb=256):
            smoke_tests()
        with TestConf(sandbox="docker", handler_cache_mb=0, import_cache_mb=0):
            smoke_tests()
        with TestConf(sandbox="docker", handler_cache_mb=256, import_cache_mb=0):
            smoke_tests()

        # test resource limits
        fork_bomb()
        
    # test heavy load
    with TestConf(sandbox="sock", handler_cache_mb=256, import_cache_mb=256, registry=test_reg):
        stress_one_lambda(procs=1, seconds=15)
        stress_one_lambda(procs=2, seconds=15)
        stress_one_lambda(procs=8, seconds=15)

    with TestConf(sandbox="sock", handler_cache_mb=256, import_cache_mb=256):
        call_each_once(lambda_count=100, alloc_mb=1)
        call_each_once(lambda_count=1000, alloc_mb=10)


def main():
    t0 = time.time()
    
    # general setup
    if os.path.exists(OLDIR):
        try:
            run(['./ol', 'kill', '-p='+OLDIR])
        except:
            print('could not kill cluster')
        run(['rm', '-rf', OLDIR])
    run(['./ol', 'new', '-p='+OLDIR])

    # run tests with various configs
    tests()

    # save test results
    passed = len([t for t in results["runs"] if t["pass"]])
    failed = len([t for t in results["runs"] if not t["pass"]])
    results["passed"] = passed
    results["failed"] = failed
    results["seconds"] = time.time() - t0
    print("PASSED: %d, FAILED: %d" % (passed, failed))

    with open("test.json", "w") as f:
        json.dump(results, f, indent=2)

    sys.exit(failed)


if __name__ == '__main__':
    main()
