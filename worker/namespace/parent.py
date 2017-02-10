import os, sys, ns, json
from subprocess import check_output

sys.path.append('/handler') # assume submitted .py file is /handler/lambda_func

def handler(args, path):
    import lambda_func
    try:
        ret = lambda_func.handler(None, json.loads(args))
    except Exception as e:
        ret = json.dumps({"error": "handler execution failed: %s" % str(e)})
    
    with open(path, 'wb') as fifo:
        fifo.write(ret+'\n')

def listen(path):
    args = ""
    with open(path) as fifo:
        while True:
            data = fifo.read()
            if len(data) == 0:
                break
            args += data

    return args

def main(pid, inpath, outpath):
    cwd = os.getcwd()
    while True:
        args = listen(inpath)
        r = ns.forkenter(pid)

        # child escapes
        if r == 0:
            break
        os.chdir(cwd)

    handler(args, outpath)

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print('Usage: parent.py <ns_pid> <input_fifo> <output_fifo>')
        sys.exit(1)
    else:
        main(sys.argv[1], sys.argv[2], sys.argv[3])
