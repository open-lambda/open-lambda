package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "google.golang.org/grpc"
    pb "example.com/open-lambda/grpc-lambda/lambda"
)

const (
    addr = "localhost:50051"
)

func main() {
    conn, err := grpc.Dial(addr, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("dial failed: %v", err)
    }
    defer conn.Close()

    client := pb.NewLambdaServiceClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
   
    req := &pb.LambdaRequest{
        Name:    "foo",
        Payload: `{"x":1}`,
    }
    resp, err := client.Invoke(ctx, req)
    if err != nil {
        log.Fatalf("invoke foo failed: %v", err)
    }
    fmt.Println("Foo response:", resp.GetResult())
}
