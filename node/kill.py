#!/usr/bin/env python
import os, sys, time, json, subprocess

def run(cmd):
    print 'EXEC ' + cmd
    return subprocess.check_output(cmd, shell=True)

def main():
    for cid in run('docker ps -q').strip().split('\n'):
	if cid == '':
		continue

        state = json.loads(run('docker inspect ' + cid))
        assert(len(state) == 1)
        paused = state[0]['State']['Paused']
        if paused:
            run('docker unpause ' + cid)
        run('docker kill ' + cid)

if __name__ == '__main__':
    main()
