from os import fork
from os import wait
from time import sleep

def handler(conn, event):
    try:
      pid = fork()
      if pid == 0:
        sleep(1)
        return "Hi"
      else:
        wait()
    except Exception as e:
        return {'error': str(e)} 
