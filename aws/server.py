import boto3
import string
import random
import json
import time

client = boto3.client('lambda')

def random_string(n):
    return ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(n))


# keys^depth values
def random_dict(num_keys, depth, value_len):
    d = {}
    for _ in range(num_keys):
        if depth < 2:
            d[random_string(8)] = random_string(value_len)
        else:
            d[random_string(8)] = random_dict(num_keys, depth-1, value_len)


    return d


def lambda_handler(event, context):
    k = event.get('num_keys',1)
    d = event.get('depth', 1)
    l = event.get('value_len', 0)
    i = event.get('iterations', 1)
    
    data = random_dict(num_keys=k, depth=d, value_len=l)
    p = json.dumps(data)
    
    #print(time.clock())
    start = time.time()

    for _ in range(i):
        resp = client.invoke(
            FunctionName='client',
            InvocationType='RequestResponse',
            Payload = p
        )
    
    #print(time.clock())
    end = time.time()

    elapsed = end - start
    print(elapsed)
    return {
        'num_keys':   k,
        'depth':      d,
        'value_len':  l,
        'iterations': i,
        'duration':   elapsed
    }

