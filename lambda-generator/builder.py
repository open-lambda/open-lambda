#!/usr/bin/python
import subprocess
import argparse

parser = argparse.ArgumentParser(description="Build and run a lambda from given"                                 " lambda file")

parser.add_argument('--lambdafile', '-l', default='hello.py')
parser.add_argument('--name', '-n', default='pyserver')
args = parser.parse_args()

subprocess.call(['cp', args.lambdafile, './pyserver/lambda_func.py'])
subprocess.call(['docker', 'build', '-t', args.name, './pyserver'])
subprocess.call(['rm', '-f', './pyserver/lambda_func.py'])
