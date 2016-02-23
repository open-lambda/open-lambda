#!/usr/bin/python
import subprocess
import argparse

parser = argparse.ArgumentParser(description="Build and run a lambda from given"                                 " lambda file")

parser.add_argument('--lambdafile', '-l', default='hello.py')
args = parser.parse_args()

subprocess.call(['cp', args.lambdafile, './pyserver/lambda_func.py'])
subprocess.call(['docker', 'build', '-t', 'pyserver:base', './pyserver'])
subprocess.call(['docker', 'run', '-d', '-p', '5000:8080', 'pyserver:base', './server.py'])
