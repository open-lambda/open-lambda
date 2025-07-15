import os, sys, time

# return True on success
def attempt(mb):
    print('attempt %dMB' % mb)
    pid = os.fork()
    assert(pid >= 0)

    if pid == 0:
        # allocate i MB of memory, may crash
        buf = 'M' * (mb * 1024**2)
        os._exit(0)

    _, status = os.waitpid(pid, 0)
    return status == 0


def f(event):
    max_attempt = 512

    for i in range(max_attempt):
        if not attempt(i):
            return i-1 # we must have succeeded with i-1 MB

    return max_attempt


if __name__ == "__main__":
    rv = attempt(int(sys.argv[1]))
    print(rv)
