#!/usr/bin/python

import sys
from subprocess import call
from gen import gen
# gen (num_keys, depth, value_len)

#maxsize = 20 # log2 of max size desired
maxsize = 0

sizes = map(lambda x: 2**x, range(maxsize+1))

def make_and_move(nk, d, vl):
    # Make files
    name = gen(nk, d, vl)
    names = "COPY " + name + ".pkl /\nCOPY " + name + "_pb2.py /\n"
    # move to lambda folder
    call(["mv", name + ".pkl", "../lambda"])
    call(["mv", name + "_pb2.py", "../lambda"])
    call(["rm", name + ".proto"])
    return names

def do_gen():
  names = ""
  for s in sizes:
    # Test 1: vary value_len
    names += make_and_move(1, 1, s)
    # Test 2: vary num_keys with 1B values
    names += make_and_move(s, 1, 1)
    # Print list of names for adding to dockerfile
    print names

if __name__ == "__main__":
  do_gen()
