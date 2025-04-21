package main

import (
    "bytes"
    "context"
    "io"
    "log"
    "net"
    "net/http"

    "google.golang.org/grpc"
    pb "example.com/open-lambda/grpc-lambda/lambda"
)

const (
    port = ":50051"
    openLambdaURL = "http://localhost:5000/run/"
)

// server implements the LambdaService defined in lambda.proto
type server struct {
    pb.UnimplementedLambdaServiceServer
}

// invoke calls the specified lambda by name with the given payload
func (s *server) Invoke(ctx context.Context, req *pb.LambdaRequest) (*pb.LambdaResponse, error) {
    name := req.GetName()
    log.Printf("🔧 Lambda name: %q", name)

    url := openLambdaURL + name
    resp, err := http.Post(url, "application/json", bytes.NewBufferString(req.GetPayload()))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    return &pb.LambdaResponse{Result: string(body)}, nil
}

func main() {
    lis, err := net.Listen("tcp", port)
    if err != nil {
        log.Fatalf("listen failed: %v", err)
    }
    grpcServer := grpc.NewServer()
    pb.RegisterLambdaServiceServer(grpcServer, &server{})

    log.Printf("gRPC server listening on %s", port)
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("serve failed: %v", err)
    }
}
