# IPC call
# Paired with IPC_resp
# This bench does the following:
#       1) Open mq (already setup by resp)
#       2) Send msg
#       3) Wait for resp
#       4) Exit

# This timing is unfair wrt the curl bench: 
# Resp sets up the channel and call just opens it, so this bench doesn't include setup time
# This is tricky to coordinate, since benches must be setup using http messages

from subprocess import call
from posix_ipc import *

bufsize = 16

def handler(conn, event):
    try:
      # Start time
      mq = MessageQueue("/mytest", flags=O_CREAT, mode=0600, max_messages = 8, max_message_size=bufsize)
      mq.send("Call msg")
      ret = mq.receive()[0] # (msg, priority)
      # Stop time
      mq.unlink()
      mq.close()
      return ret
    except Exception as e:
        return {'error': str(e)} 
