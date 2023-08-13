# ol-install: redis

import redis

def f(event):
    redis_host = "127.0.0.1"
    redis_port = 6379
    r = redis.Redis(host=redis_host, port=redis_port, db=0)
    # redis_username = "default"
    # redis_password = "openlambda"
    # r = redis.Redis(host=redis_host, port=redis_port, db=0, username=redis_username, password=redis_password)
    r.srem('clients', event['context']['id'])
    return {"body": "disconnected"}
