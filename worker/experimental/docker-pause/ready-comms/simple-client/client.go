package main

import (
	"fmt"
	"log"
	"os"

	pb "github.com/open-lambda/open-lambda/lambdaManager/experimental/docker-pause/ready-comms/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var address string

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("usage: %s <hostname> <port>\n", os.Args[0])
	}

	address = fmt.Sprintf("%s:%s", os.Args[1], os.Args[2])

	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewReadyClient(conn)

	// Contact the server and print out its response.
	r, err := c.SayReady(context.Background(), &pb.ReadyRequest{Id: "Hello!"})
	if err != nil {
		log.Fatalf("could not say ready: %v", err)
	}
	// Assume response good
	if r != nil {
		log.Println("good response\n")
	}
}
