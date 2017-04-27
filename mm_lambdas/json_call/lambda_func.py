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
        tmp = json.dumps(d)
        #mq = MessageQueue("/mytest", flags=O_CREAT | O_EXCL, mode=0600, max_messages = 8, max_message_size=len(tmp.encode('utf-8')))
        mq = MessageQueue("/mytest")
        
        # Timed send
        start = time.time()
        payload = json.dumps(d)
        mq.send(payload)
        # Wait for ack
        mq.receive() 

        stop = time.time()
        mq.close()
        mq.unlink()
        return "t: " + str(stop - start) + " s: " + str(len(payload.encode('utf-8')))
    except Exception as e:
        return {'error': str(e)} 
