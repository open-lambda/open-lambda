# See ipc_call
# This bench does the following:
#       1) Open IPC channel
#       2) Wait for message from call side
#       3) Send Ack
#       4) Exit

from posix_ipc import *
import importlib
import random
import string
from sys import getsizeof

def random_string(n):
  return 'X' + ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(n-1))

# Set protobuf fields to random stuff
def setfields(obj, value_len):
  for fd in obj.DESCRIPTOR.fields:
    if fd.type == fd.TYPE_MESSAGE:
      setfields(getattr(obj, fd.name), value_len)
    else:
      setattr(obj, fd.name, random_string(value_len))

def handler(conn, event):
    try:
      # Responder also must open pickle file to set mq bufsize appropriately
      num_keys = int(event['num_keys'])
      depth = int(event['depth'])
      value_len = int(event['value_len'])
    
      name = "rand_" + str(num_keys) + "_" + str(depth) + "_" + str(value_len)
      i = importlib.import_module(name + "_pb2")
      d = i.Rand()
      setfields(d, value_len)

      # Setup mq
      tmp = d.SerializeToString()
      mq = MessageQueue("/mytest", flags=(O_CREAT | O_EXCL), mode=0600, max_messages = 8, max_message_size=len(tmp))

      # Recv and close
      mq.receive() # Get message from ipc_call
      mq.send("Ack")   # Ack
      mq.close()
      return str(len(tmp))
    except Exception as e:
        return {'error': str(e)} 
