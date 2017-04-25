# See ipc_call
# This bench does the following:
#       1) Open IPC channel
#       2) Wait for message from call side
#       3) Send Ack
#       4) Exit

from posix_ipc import *
import importlib
from sys import getsizeof

def handler(conn, event):
    try:
      # Responder also must open pickle file to set mq bufsize appropriately
      num_keys = int(event['num_keys'])
      depth = int(event['depth'])
      value_len = int(event['value_len'])
    
      name = "rand_" + str(num_keys) + "_" + str(depth) + "_" + str(value_len)
      i = importlib.import_module(name + "_pb2")
      d = i.Rand()

      # Setup mq
      tmp = d.SerializeToString()
      mq = MessageQueue("/mytest", flags=O_CREAT, mode=0600, max_messages = 8, max_message_size=getsizeof(tmp))

      # Recv and close
      mq.receive() # Get message from ipc_call
      mq.send("Ack")   # Ack
      mq.close()
      return ""
    except Exception as e:
        return {'error': str(e)} 
