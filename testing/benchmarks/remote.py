import os, sys
from subprocess import check_output

SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
global CONF_DIR
global RESET

BENCH_DIR = '/ol/open-lambda/testing/benchmarks'
WORKER_HOST = 'node-0.sosp.openlambda-pg0.wisc.cloudlab.us'

def fetch_log(conf):
    try:
        prefix = conf.split('.json')[0]
        local = os.path.join(CONF_DIR, '%s.worker-log' % prefix)
        print(str(check_output(['scp', '%s:%s/test/logs/worker-0.out' % (WORKER_HOST, BENCH_DIR), local])))
    except Exception as e:
        print('fetching log for %s failed: %s' % (cmd, e))

def ssh(cmd):
    try:
        print(str(check_output(['ssh', WORKER_HOST, cmd])))
    except Exception as e:
        print('executing "%s" failed: %s' % (cmd, e))

def get_confs():
    confs = {}
    for f in os.listdir(CONF_DIR):
        if f.endswith('.json'):
            try:
                cmd = f.split('.json')[0] + '.cmd'
                with open(os.path.join(CONF_DIR, cmd)) as fd:
                    confs[f] = str(fd.read().strip())
            except:
                print('missing worker command for %s' % f)

    return confs.items()

def run_bench(conf):
    print('run_bench for: %s' % conf)

def main():
    global RESET
    RESET = 'cd %s;python3 cluster_util.py --stop-all;rm -rf test' % BENCH_DIR

    for conf,setup in get_confs():
        try:
            print('Resetting worker...')
            ssh(RESET)
            print('Starting worker...')
            print('cmd: %s' % setup)
            ssh(setup)
            print('Running test "%s"...' % conf)
            run_bench(conf)
            print('Fetching worker log...')
            fetch_log(conf)
        except Exception as e:
            print('TEST %s FAILED: %s' % (conf, e))

    ssh(RESET)

if __name__ == '__main__':
    global CONF_DIR
    if len(sys.argv) != 2:
        print('Usage: %s <config_dir>' % sys.argv[0])
        sys.exit(1)

    CONF_DIR = os.path.join(SCRIPT_DIR, sys.argv[1])
    main()
