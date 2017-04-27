#!/usr/bin/python

import sys
import os
from time import sleep
from subprocess import call, Popen
from posix_ipc import *
import resource

# For test 1 and 2
maxsize = 20 # log2 of max size desired
#maxsize = 0 # log2 of max size desired
sizes = map(lambda x: 2**x, range(maxsize+1))

# For test 3
maxdepth = 3
depthsize = 10 # log2 of size of data struct for depth tests (4096)

benches = ["pkl", "json", "proto"]
#benches = ["proto"]

def delete_queue():
  # Make sure mq is dead
  try:
    mq = MessageQueue("/mytest")
    mq.unlink()
    mq.close()
    call(["rm", "-f", "/dev/mqueue/mytest"])
  except:
    pass

def call_lambda(lam, log, num_keys, depth, value_len):
  delete_queue()
  call(["./bin/admin", "workers", "-cluster=bench_resp", "-p=8081"])
  call(["./bin/admin", "workers", "-cluster=bench_call"])
  sleep(1)
  args = "{\"num_keys\" : \"" + str(num_keys) + "\", \"depth\" : \"" + str(depth) + "\", \"value_len\" : \"" + str(value_len) + "\" }"
  Popen(["curl", "-X", "POST", "localhost:8081/runLambda/" + lam + "_resp", "-d", args])
  sleep(1)
  call(["curl", "-X", "POST", "localhost:8080/runLambda/" + lam + "_call", "-d", args], stdout=log)
  call(["./clean_bench.sh"])

def t1():
  for b in benches:
    f = open("serial_results/" + b + ".t1.log", "a")
    for s in sizes:
      # Test 1: vary value_len
      call_lambda(b, f, 1, 1, s)
      f.write("\n")
      f.flush()
    f.close()

def t2():
  for b in benches:
    g = open("serial_results/" + b + ".t2.log", "a")
    for s in sizes:
      # Test 2: vary num_keys with 1B values
      call_lambda(b, g, s, 1, 1)
      g.write("\n")
      g.flush()
    g.close()

def t3():
  depths = map(lambda x: 2**x, range(maxdepth+1))
  for b in benches:
    g = open("serial_results/" + b + ".t3.log", "a")
    for d in depths:
      call_lambda(b, g, depthsize/d, d, 1)
      g.write("\n")
      g.flush()
    g.close()

def main():
  #t1()
  #t2()
  t3()

if __name__ == "__main__":
  main()
