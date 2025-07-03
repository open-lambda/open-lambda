package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    "google.golang.org/grpc"
    pb "example.com/open-lambda/grpc-lambda/lambda"
)

var (
    addr    = flag.String("addr", "localhost:50051", "gRPC server address")
    name    = flag.String("name", "echo", "Lambda function name")
    payload = flag.String("data", `{"hello":"world"}`, "JSON payload")
)

func main() {
    flag.Parse()

    conn, err := grpc.Dial(*addr, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("dial failed: %v", err)
    }
    defer conn.Close()
    client := pb.NewLambdaServiceClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
    defer cancel()

    req := &pb.LambdaRequest{
        Name:    *name,
        Payload: *payload,
    }
    resp, err := client.Invoke(ctx, req)
    if err != nil {
        log.Fatalf("invoke %s failed: %v", *name, err)
    }
    fmt.Printf("Response from %s: %s\n", *name, resp.GetResult())
}
