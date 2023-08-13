# ol-install: grpcio
# ol-install: redis
# ol-install: protobuf

import grpc
import json
import wsmanager_pb2_grpc
import wsmanager_pb2

channel = grpc.insecure_channel('127.0.0.1:50051')
client = wsmanager_pb2_grpc.WsManagerStub(channel)

# this function is used to send messages to designated connections
def f(event):
    for id in event["body"]["receiver_ids"]:
        request = wsmanager_pb2.PostToConnectionRequest()
        body = {
            "sender_id": event["context"]["id"],
            "body": event["body"]["msg"]
        }
        print(json.dumps(body))
        request.msg = json.dumps(body)
        request.connection_id = id  # indicate the connection id to send message to

        response = client.PostToConnection(request)
        if response.success:
            print('PostToConnection successful')
        else:
            print('PostToConnection failed: {}'.format(response.error))
    return {"body": "messages sent"}
