# See ipc_call
# This bench does the following:
#       1) Open IPC channel
#       2) Wait for message from call side
#       3) Send Ack
#       4) Exit

from subprocess import call
from posix_ipc import *

bufsize = 16

def handler(conn, event):
    try:
      mq = MessageQueue("/mytest", flags=O_CREAT, mode=0600, max_messages = 8, max_message_size=bufsize)
      mq.receive() # Get message from ipc_call
      mq.send("Ack")   # Ack
      mq.close()
      return ""
    except Exception as e:
        return {'error': str(e)} 
