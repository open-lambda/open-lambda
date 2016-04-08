#!/usr/bin/env python
import subprocess
import argparse
import os, sys

SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))

def main():
    parser = argparse.ArgumentParser(description="Build and run a lambda from given lambda file")

    parser.add_argument('--lambdafile', '-l', default='hello.py')
    parser.add_argument('--name', '-n', default='pyserver')
    args = parser.parse_args()

    pyserver_dir = os.path.join(SCRIPT_DIR, 'pyserver')
    lambda_func = os.path.join(pyserver_dir, 'lambda_func.py')
    subprocess.call(['cp', args.lambdafile, lambda_func])
    subprocess.call(['docker', 'build', '-t', args.name, pyserver_dir])
    subprocess.call(['rm', '-f', lambda_func])

if __name__ == '__main__':
    main()



