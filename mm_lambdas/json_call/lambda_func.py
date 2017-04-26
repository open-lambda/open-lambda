import json
import cPickle as pickle
import time
from posix_ipc import *
from sys import getsizeof

def handler(conn, event):
    try:
        num_keys = int(event['num_keys'])
        depth = int(event['depth'])
        value_len = int(event['value_len'])
      
        name = "rand_" + str(num_keys) + "_" + str(depth) + "_" + str(value_len)
        f = open("../" + name + ".pkl",'rb')
        d = pickle.load(f)
        f.close()

        # Setup mq
        tmp = pickle.dumps(d)
        mq = MessageQueue("/mytest", flags=O_CREAT, mode=0600, max_messages = 8, max_message_size=getsizeof(tmp))
        
        # Timed send
        start = time.time()
        payload = json.dumps(d)
        mq.send(payload)
        # Wait for ack
        mq.receive() 

        stop = time.time()
        mq.close()
        mq.unlink()
        return "t: " + str(stop - start) + " s: " + str(getsizeof(payload))
    except Exception as e:
        return {'error': str(e)} 
