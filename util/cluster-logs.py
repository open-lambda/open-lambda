#!/usr/bin/env python
import os, sys, subprocess, json, argparse
from common import *

SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
CLUSTER_DIR = os.path.join(SCRIPT_DIR, 'cluster')

def main():
    parser = argparse.ArgumentParser(description='Compile the logs of all workers in the cluster')
    parser.add_argument('--destination', '-d', default=False)
    parser.add_argument('--tail', '-t', default=False)
    args = parser.parse_args()

    if not os.path.exists(CLUSTER_DIR):
        print 'Cluster not running!'
        sys.exit(1)

    if args.destination:
	out = open(args.destination, 'w+')
    else:
	out = sys.stdout

    for c in run('docker ps').strip().split('\n')[1:]:
	if c == '':
		continue

	attr = c.split()
   	cid = attr[0]
	image = attr[1].split(':')[0]	 

	out.write("\n<--------------- Start '%s' container [%s] logs: --------------->\n\n" % (image, cid))
	out.flush()

	cmd = ['docker', 'logs']
	if args.tail:
	    cmd.append('--tail="%s"' % args.tail)

	cmd.append(cid)

	if subprocess.call(cmd, stdout=out, stderr=out) != 0:
            print 'Docker logs command execution failed.'
	    sys.exit(1)

        out.write("\n<--------------- End '%s' container [%s] logs --------------->\n" % (image, cid))

    out.close()
if __name__ == '__main__':
    main()
