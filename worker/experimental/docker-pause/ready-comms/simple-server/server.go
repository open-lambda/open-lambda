package main

import (
	"fmt"
	"log"
	"net"
	"os"

	pb "github.com/open-lambda/open-lambda/lambdaManager/experimental/docker-pause/ready-comms/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var port string

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayReady(ctx context.Context, in *pb.ReadyRequest) (*pb.ReadyReply, error) {
	log.Printf("acking ready request Id: %s\n", in.Id)
	return &pb.ReadyReply{}, nil
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <port>", os.Args[0])
	}
	port = fmt.Sprintf(":%s", os.Args[1])
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterReadyServer(s, &server{})
	s.Serve(lis)
}
