#!/usr/bin/python

import sys
from time import sleep
from subprocess import call, Popen

#maxsize = 20 # log2 of max size desired
maxsize = 0

sizes = map(lambda x: 2**x, range(maxsize+1))
benches = ["pkl", "json", "proto"]

def call_lambda(lam, log, num_keys, depth, value_len):
  call(["./bin/admin", "workers", "-cluster=bench_resp", "-p=8081"])
  call(["./bin/admin", "workers", "-cluster=bench_call"])
  sleep(1)
  args = "{\"num_keys\" : \"" + str(num_keys) + "\", \"depth\" : \"" + str(depth) + "\", \"value_len\" : \"" + str(value_len) + "\" }"
  Popen(["curl", "-X", "POST", "localhost:8081/runLambda/" + lam + "_resp", "-d", args])
  call(["curl", "-X", "POST", "localhost:8080/runLambda/" + lam + "_call", "-d", args], stdout=log)
  call(["./clean_bench.sh"])

def main():
  for b in benches:
    f = open("serial_results/" + b + ".t1.log", "a")
    g = open("serial_results/" + b + ".t2.log", "a")
    for s in sizes:
      # Test 1: vary value_len
      call_lambda(b, f, 1, 1, s)
      f.write("\n")
      f.flush()
      # Test 2: vary num_keys with 1B values
      call_lambda(b, g, s, 1, 1)
      g.write("\n")
      g.flush()
    f.close()
    g.close()

if __name__ == "__main__":
  main()
