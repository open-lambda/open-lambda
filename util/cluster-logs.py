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

    for filename in os.listdir(CLUSTER_DIR):
        path = os.path.join(CLUSTER_DIR, filename)
        if not filename.endswith('.json'):
            continue
            
	info = rdjs(path)
        cid = info['cid']
        if info['type'] == 'worker':
	    out.write("\n<--------------- Start worker [%s] logs: --------------->\n\n" % cid)
	    out.flush()

	    cmd = ['docker', 'logs']
	    if args.tail:
	        cmd.append('--tail="%s"' % args.tail)

	    cmd.append(cid)

	    if subprocess.call(cmd, stdout=out, stderr=out) != 0:
	        print 'Docker logs command execution failed.'
	        sys.exit(1)

	    #if not args.tail:
		#out.write("\n")

	    out.write("\n<--------------- End worker [%s] logs --------------->\n" % cid)

    out.close()
if __name__ == '__main__':
    main()
