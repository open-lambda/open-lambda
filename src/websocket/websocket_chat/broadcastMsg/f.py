# ol-install: grpcio
# ol-install: redis
# ol-install: protobuf

import grpc
import redis

import wsmanager_pb2_grpc
import wsmanager_pb2

channel = grpc.insecure_channel('127.0.0.1:50051')
client = wsmanager_pb2_grpc.WsManagerStub(channel)

# remote database used to store connection ids, set username and password if needed
redis_host = "127.0.0.1"
redis_port = 6379

# this function is used to send messages to all connections, namely broadcast
def f(event):
    r = redis.Redis(host=redis_host, port=redis_port, db=0)
    members = r.smembers('clients')
    msg = {
        "sender_id": event['context']['id'],
        "body": event['body']['msg']
    }

    for member in members:
        request = wsmanager_pb2.PostToConnectionRequest()
        request.msg = str(msg)
        request.connection_id = member.decode('utf-8')

        response = client.PostToConnection(request)
        if response.success:
            print('PostToConnection successful')
        else:
            print('PostToConnection failed: {}'.format(response.error))
    return {"body": "messages sent"}
