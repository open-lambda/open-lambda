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
from time import sleep
import os
from posix_ipc import *

bufsize = 16

def child():
  sleep(5)
  mq = MessageQueue("/mytest", flags=O_CREAT, mode=0600, max_messages = 8, max_message_size=bufsize)
  mq.send("Call msg")
  ret = mq.receive()[0] # (msg, priority)
  os._exit(0)	 # no return

def handler(conn, event):
    try:
      pid = os.fork()
      if pid == 0:
         child()
      else:
        # Runs perf record on child PID while child does IPC
        # Perf will terminate when child exits
        # Switch "ext4:*" for your FS as necessary
        call(["perf","record", "-ag", "-F", "99", "--output=perf.data",
              "-e", "syscalls:sys_*", "-e", "net:*", "-e", "skb:*" ,"-e", "sock:*" ,"-e", "cpu-clock",
              "-e", "ext4:*",
              "-p", str(pid) ])
        os.waitpid(pid,0) 

        # Make sure mq is unlinked (lives in host IPC namespace)
        mq = MessageQueue("/mytest")
        mq.close()
        mq.unlink()
        
        # Run result through perf script and return
        f = open("f", 'wb')
        g = open("g", 'wb')
        call(["perf", "script", "--input=perf.data"], stdout=f, stderr=g)
        f.close()
        g.close()
        f = open("f", 'rb')
        g = open("g", 'rb')
        return f.read() + g.read()
    except Exception as e:
        return {'error': str(e)} 
