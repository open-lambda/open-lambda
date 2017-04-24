#!/usr/bin/python

import sys
import string
import random
import json
import cPickle as pickle
from subprocess import call

def random_string(n):
  # Tweak: Start with char for protobuf
  return 'X' + ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(n-1))

# keys^depth values
def random_dict(num_keys, depth, value_len):
  d = {}
  for _ in range(num_keys):
      if depth < 2:
          d[random_string(8)] = random_string(value_len)
      else:
          d[random_string(8)] = random_dict(num_keys, depth-1, value_len)
  
  return d

# Iterate through nested dict and write out as protobuf structure
def proto_dict(d, name):
  i = 1
  ret = "message " + name + " {\n"
  for name,value in d.items():
    if(isinstance(value, dict)):
      ret += proto_dict(value, name)
      ret += "required " + name + "d" + str(i) + " = " + str(i) + ";\n"
    else:
      ret += "required string " + name + " = " + str(i) + ";\n" 
    i += 1
  ret += "}\n"
  return ret

def gen(num_keys, depth, value_len):
  # Create dictionary
  d = random_dict(num_keys, depth, value_len)
  name = "rand_" + str(num_keys) + "_" + str(depth) + "_" + str(value_len)

  # Save python-format dict to file (for JSON and pickle tests)
  f = open(name + ".pkl", "wb") 
  pickle.dump(d, f)
  f.close()

  # Iterate through dictionary and dump to .proto file
  f = open(name + ".proto", "wb") 
  f.write('syntax = "proto2";\n')
  f.write("package " + name + ";\n") 
  f.write(proto_dict(d, "Rand"))
  f.close()
  # Invoke compiler
  call(["protoc", name + ".proto", "--python_out=."]) 

def main():
  if len(sys.argv) != 4:
    print "Usage: " + sys.argv[0] + " [num_keys] [depth] [value_len]"
    exit()
  
  num_keys = int(sys.argv[1])
  depth = int(sys.argv[2])
  value_len = int(sys.argv[3])
  
  gen(num_keys, depth, value_len)

if __name__ == "__main__":
  main()
