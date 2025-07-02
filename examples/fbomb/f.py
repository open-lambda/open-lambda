import os, sys, time

# attempt to create n new processes; return the number that are succesfully created
def fork_times(n):
    print("fork_times(%d)"%n)
    sys.stdout.flush()
    if n == 0:
        return 0

    try:
        pid = os.fork()
    except OSError:
        return 0

    if pid:
        _, rv = os.waitpid(pid, 0)
        rv = rv // 256 # get high byte
        return rv
    else:
        rv = 1 + fork_times(n-1)
        os._exit(rv)


def f(event):
    return fork_times(event["times"])


if __name__ == "__main__":
    rv = fork_times(int(sys.argv[1]))
    print(rv)
