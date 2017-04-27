# See ipc_call
# This bench does the following:
#       1) Open IPC channel
#       2) Wait for message from call side
#       3) Send Ack
#       4) Exit

from posix_ipc import *
import json
from sys import getsizeof
import cPickle as pickle

def handler(conn, event):
    try:
      # Responder also must open pickle file to set mq bufsize appropriately
      num_keys = int(event['num_keys'])
      depth = int(event['depth'])
      value_len = int(event['value_len'])
    
      name = "rand_" + str(num_keys) + "_" + str(depth) + "_" + str(value_len)
      f = open(name + ".pkl",'rb')
      d = pickle.load(f)
      f.close()

      # Setup mq
      tmp = json.dumps(d)
      mq = MessageQueue("/mytest", flags=O_CREAT | O_EXCL, mode=0600, max_messages = 8, max_message_size=len(tmp.encode('utf-8')))

      # Recv and close
      mq.receive() # Get message from ipc_call
      mq.send("Ack")   # Ack
      mq.close()
      return ""
    except Exception as e:
        return {'error': str(e)} 
