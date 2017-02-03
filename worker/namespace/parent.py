#TODO make sure listening on the pipe blocks correctly, better error handling

import os, sys, ns, time
from subprocess import check_output

sys.path.append('/handler') # assume submitted .py file is /handler/lambda_func

def handler(args, path):
    import lambda_func
    try:
        ret = lambda_func(args)
    except:
        ret = json.dumps{'error': 'handler execution failed with args: %s' % args}
    
    with open(path) as pipe:
        pipe.write(ret)

def listen(path):
    args = ""
    with open(path) as pipe:
        while True:
            data = pipe.read()
            if len(data) == 0:
                break
            args += data

    return args

def main(pid, inpath, outpath):
    # parent never exits
    while True:
        args = listen(inpath)

        r = forkenter(pid)
        if r == 0:
            break       # grandchild escapes
        elif r < 0:
            sys.exit(0) # child dies quietly

    handler(args, outpath)

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print('Usage: test.py <ns_pid> <input_pipe> <output_pipe>')
        sys.exit(1)
    else:
        main(sys.argv[1], sys.argv[2] sys.argv[3])
