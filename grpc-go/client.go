package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "google.golang.org/grpc"
    pb "open-lambda/grpc-go/echo"
)

const serverAddr = "localhost:50051"

func main() {
    conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewEchoServiceClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    jsonData := `{"hello": "world"}`

    resp, err := client.SendRequest(ctx, &pb.EchoRequest{JsonData: jsonData})
    if err != nil {
        log.Fatalf("Error calling EchoService: %v", err)
    }

    fmt.Printf("Received from Lambda: %s\n", resp.JsonResult)
}
