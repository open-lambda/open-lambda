import importlib
import time
from posix_ipc import *
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
        num_keys = int(event['num_keys'])
        depth = int(event['depth'])
        value_len = int(event['value_len'])
      
        name = "rand_" + str(num_keys) + "_" + str(depth) + "_" + str(value_len)
        i = importlib.import_module(name + "_pb2")
        d = i.Rand()
        setfields(d, value_len)
        obj = d.SerializeToString()

        # Setup mq
        tmp = d.SerializeToString()
        mq = MessageQueue("/mytest", flags=O_CREAT, mode=0600, max_messages = 8, max_message_size=getsizeof(tmp))
        
        # Timed send
        start = time.time()
        payload = d.SerializeToString()
        mq.send(payload)
        # Wait for ack
        mq.receive() 

        stop = time.time()
        mq.close()
        mq.unlink()
        return "t: " + str(stop - start) + " s: " + str(getsizeof(payload))
    except Exception as e:
        return {'error': str(e)} 
