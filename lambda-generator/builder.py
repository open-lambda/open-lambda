#!/usr/bin/env python
import subprocess
import argparse
import os, sys
import json

SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))

def main():
    parser = argparse.ArgumentParser(description="Build and run a lambda from given lambda file")

    parser.add_argument('--lambdafile', '-l',
                        default=os.path.join(SCRIPT_DIR, 'hello.py'))
    parser.add_argument('--configfile', '-c',
                        default=os.path.join(SCRIPT_DIR, 'default.json'))
    parser.add_argument('--name', '-n', default='pyserver')
    parser.add_argument('--environmentfile', '-e')
    args = parser.parse_args()

    pyserver_dir = os.path.join(SCRIPT_DIR, 'pyserver')
    tempfiles = []

    if args.environmentfile:
	with open(args.environmentfile, 'r') as fd:
		ENV_DIR = os.path.dirname(fd.name)
        	env = json.load(fd)

	custom_df = os.path.join(pyserver_dir, 'Dockerfile_custom')
    	with open(custom_df, 'w+') as fd:
        	fd.write('FROM '+env['distribution']+'\n')
        	if 'alpine' in env['distribution']:	
			fd.write('RUN apk update\n')
            		fd.write('RUN apk add python-dev py-pip\n')

        	if 'phusion/baseimage' in env['distribution'] or 'ubuntu' in env['distribution']:
            		fd.write('RUN apt-get -y update\n')
            		fd.write('RUN apt-get -y install python-pip\n')

        	if 'copy' in env:
            		for copy in env['copy']:
				name = os.path.basename(copy)
				if copy.startswith('/'):
					path = copy
				else:
					path = os.path.join(ENV_DIR, copy)
    				subprocess.call(['cp', path, os.path.join(pyserver_dir, name)])
                		fd.write('COPY '+name+' /\n')
				tempfiles.append(name)

        	if 'add' in env:
            		for add in env['add']:
				name = os.path.basename(add)
				if add.startswith('/'):
					path = add
				else:
					path = os.path.join(ENV_DIR, add)
    				subprocess.call(['cp', path, os.path.join(pyserver_dir, name)])
                		fd.write('ADD '+name+' /\n')
				tempfiles.append(name)
        	
		if 'apt' in env:
            		for package in env['apt']:
                		fd.write('RUN apt-get -y install '+package+'\n')

        	if 'apk' in env:
            		for package in env['apk']:
                		fd.write('RUN apk add --no-cache '+package+'\n')

        	if 'pip' in env:
            		for package in env['pip']:
                		fd.write('RUN pip install '+package+'\n')

        	if 'run' in env:
            		for fn in env['run']:
                		fd.write('RUN '+fn+'\n')


        	fd.write('COPY server.py /\n')
        	fd.write('COPY lambda_func.py /\n')
        	fd.write('COPY config.json /\n')
        	fd.write('CMD python /server.py /\n')

    tempfiles.append('lambda_func.py')
    lambda_func = os.path.join(pyserver_dir, 'lambda_func.py')
    config_file = os.path.join(pyserver_dir, 'config.json')
    subprocess.call(['cp', args.lambdafile, lambda_func])
    subprocess.call(['cp', args.configfile, config_file])
    if args.environmentfile:
        subprocess.call(['docker', 'build', '-t', args.name, '-f', custom_df, pyserver_dir])
    else:
        subprocess.call(['docker', 'build', '-t', args.name, pyserver_dir])

    for tempfile in tempfiles:
	fp = os.path.join(pyserver_dir, tempfile)
    	subprocess.call(['rm', '-f', fp])

if __name__ == '__main__':
    main()
