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


def mean(data):
    """Return the sample arithmetic mean of data."""
    n = len(data)
    if n < 1:
        raise ValueError('mean requires at least one data point')
    return sum(data)/float(n)

def _ss(data):
    """Return sum of square deviations of sequence data."""
    c = mean(data)
    ss = sum((x-c)**2 for x in data)
    return ss

def dev(data):
    """Calculates the population standard deviation."""
    n = len(data)
    if n < 2:
        raise ValueError('variance requires at least two data points')
    ss = _ss(data)
    pvar = ss/float(n-1) # the sample variance
    return pvar**0.5


def lambda_handler(event, context):
    k = event.get('num_keys',0)
    d = event.get('depth', 0)
    l = event.get('value_len', 0)
    i = event.get('iterations', 1)
    
    data = random_dict(num_keys=k, depth=d, value_len=l)
    p = json.dumps(data)

    times = []
    

    for _ in range(i):

        start = time.time()
        resp = client.invoke(
            FunctionName='client',
            InvocationType='RequestResponse',
            Payload = p
        )
        end = time.time()

        times += [end-start]


    elapsed = sum(times)
    minimum = min(times)
    maximum = max(times)
    stddev = 0
    if i >= 2:
        stddev =  dev(times)
        

    return {
        # 'num_keys':   k,
        # 'depth':      d,
        # 'value_len':  l,
        # 'iterations': i,
        'duration':   elapsed,
        'min':        minimum,
        'max':        maximum,
        'stddev':     stddev
    }


if __name__ == "__main__":
    e = {
        'num_keys': 1,
        'depth': 1,
        'value_len': 4194304,
        'iterations': 1
    }

    lambda_handler(e, {})
