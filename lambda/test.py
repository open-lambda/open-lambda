import os, sys, ns
from subprocess import check_output

def main(path):
    ls = ['ls', '/']
    ps = ['ps']

    r = ns.fdlisten(path)

    # parent
    if r > 0:
        print('Parent should not escape')

    # child
    if r == 0:
        print("CHILD LS:\n%s\n" % check_output(ls).replace('\n', ' '))
        print("CHILD PS (pid=%s):\n%s" % (os.getpid(), check_output(ps)))
        check_output(['touch', '/IM_INSIDE'])

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print('Usage: test.py <sockpath>')
    else:
        main(sys.argv[1])
