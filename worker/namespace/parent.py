import os, sys, ns, time
from subprocess import check_output

sys.path.append('/handler') # assume submitted .py file is /handler/lambda_func

def handler(args, path):
    import lambda_func
    try:
        ret = lambda_func(args)
    except:
        ret = json.dumps('"error": "handler execution failed"')
    
    with open(path) as fifo:
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

def main(pid, inpath, outpath):
    while True:
        args = listen(inpath)
        r = forkenter(pid)

        # child escapes
        if r == 0:
            break

    handler(args, outpath)

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print('Usage: parent.py <ns_pid> <input_fifo> <output_fifo>')
        sys.exit(1)
    else:
        main(sys.argv[1], sys.argv[2], sys.argv[3])
