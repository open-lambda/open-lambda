# ol-install: redis

import redis

def f(event):
    redis_host = "127.0.0.1"
    redis_port = 6379
    r = redis.Redis(host=redis_host, port=redis_port, db=0)

    # redis_host = "redis-15291.c14.us-east-1-3.ec2.cloud.redislabs.com"
    # redis_port = 15291
    # redis_username = "default"
    # redis_password = "openws"
    # r = redis.Redis(host=redis_host, port=redis_port, db=0, username=redis_username, password=redis_password)
    r.sadd('clients', event['context']['id'])
    return {'body': event['context']['id']}
