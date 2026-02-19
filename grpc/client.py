import grpc
import echo_pb2
import echo_pb2_grpc

class EchoClient:
    def __init__(self, channel):
        self.stub = echo_pb2_grpc.EchoServiceStub(channel)

    def send_request(self, json_data):
        response = self.stub.SendRequest(echo_pb2.EchoRequest(json_data=json_data))
        return response.json_result

def run():
    with grpc.insecure_channel('localhost:50051') as channel:
        client = EchoClient(channel)
        json_data = '{"hello": "world"}'
        response = client.send_request(json_data)
        print(f"Received from Lambda: {response}")

if __name__ == '__main__':
    run()
