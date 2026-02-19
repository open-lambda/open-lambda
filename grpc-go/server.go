package main

import (
    "context"
    "log"
    "net"
    "io/ioutil"
    "net/http"
    "bytes"

    "google.golang.org/grpc"
    pb "open-lambda/grpc-go/echo"
)

const (
    port          = ":50051"
    openLambdaURL = "http://localhost:5000/run/echo"
)

type server struct {
    pb.UnimplementedEchoServiceServer
}

func (s *server) SendRequest(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
    jsonData := req.JsonData

    resp, err := http.Post(openLambdaURL, "application/json", bytes.NewBuffer([]byte(jsonData)))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    return &pb.EchoResponse{JsonResult: string(body)}, nil
}

func main() {
    lis, err := net.Listen("tcp", port)
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    grpcServer := grpc.NewServer()
    pb.RegisterEchoServiceServer(grpcServer, &server{})

    log.Printf("gRPC Server started on port %s...", port)
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
