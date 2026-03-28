import grpc
import echo_pb2
import echo_pb2_grpc
from concurrent import futures
import requests

OPENLAMBDA_URL = "http://localhost:5000/run/echo"

class EchoService(echo_pb2_grpc.EchoServiceServicer):
    def SendRequest(self, request, context):
        json_data = request.json_data

        response = requests.post(OPENLAMBDA_URL, data=json_data)

        if response.status_code == 200:
            result = response.text
        else:
            result = f"Error: {response.status_code}"

        return echo_pb2.EchoResponse(json_result=result)

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    echo_pb2_grpc.add_EchoServiceServicer_to_server(EchoService(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("gRPC Server started on port 50051...")
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
