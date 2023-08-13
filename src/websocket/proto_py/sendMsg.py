# ol-install: grpcio
# ol-install: redis
import os
# delete
import grpc
import redis

print(os.popen('pwd').readlines())

import wsmanager_pb2 as wsmanager_pb2
import wsmanager_pb2_grpc as wsmanager_pb2_grpc

redis_host = "redis-10580.c14.us-east-1-3.ec2.cloud.redislabs.com"
redis_port = 10580
redis_username = "default"
redis_password = "openws"

channel = grpc.insecure_channel('192.168.60.128:50051')
client = wsmanager_pb2_grpc.WsManagerStub(channel)
def f(event):
    r = redis.Redis(host=redis_host, port=redis_port, db=0, username=redis_username, password=redis_password)
    members = r.smembers('clients')
    print(len(members))
    for member in members:
        print(member.decode('utf-8'))
        request = wsmanager_pb2.PostToConnectionRequest()
        request.msg = 'Hello'
        request.connection_id = member.decode('utf-8')

        response = client.PostToConnection(request)

        if response.success:
            print('PostToConnection successful')
        else:
            print('PostToConnection failed: {}'.format(response.error))