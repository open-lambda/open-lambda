import os, sys, ns, json

sys.path.append('/handler') # assume submitted .py file is /handler/lambda_func

def handler(args, path):
    try:
        ret = lambda_func.handler(None, json.loads(args))
    except Exception as e:
        ret = json.dumps({"error": "handler execution failed: %s" % str(e)})
    
    with open(path, 'wb') as fifo:
        fifo.write(ret)

def listen(path):
    args = ""
    with open(path) as fifo:
        while True:
            data = fifo.read()
            if len(data) == 0:
                break
            args += data
    return args

def main(hfifo, cfifo):
    # change to absolute path in case cwd changed during forkenter
    hfifo = os.path.abspath(hfifo)

    # listen to forkenter request from worker
    while True:
        pid = listen(hfifo)
        r = ns.forkenter(pid)

        # child escapes
        if r == 0:
            break

    import lambda_func
    # listen to requests to run handler
    while True:
        args = listen(cfifo)
        handler(args, cfifo)

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print('Usage: parent.py <host_fifo> <container_fifo>')
        sys.exit(1)
    else:
        main(sys.argv[1], sys.argv[2])
