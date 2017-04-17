from subprocess import call
from posix_ipc import *
from os import fork
from os import wait

bufsize = 16

def handler(conn, event):
    try:
      pid = fork()
      if pid == 0:
        mq = MessageQueue("/mytest")
        return mq.receive()[0] # (msg, priority) blocking
      else:
        mq = MessageQueue("/mytest", flags=O_CREAT, mode=0600, max_messages = 8, max_message_size=bufsize)
        mq.send("Boo!")
        wait() # Won't make it past here: handler takes whichever proc returns first
        mq.unlink()
        mq.close()
        return "ERROR: made it to end of parent"
    except Exception as e:
        return {'error': str(e)} 
