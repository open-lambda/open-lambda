package registry

import (
	"fmt"
	"testing"
	"time"
)

const (
	SERVER_ADDR = "127.0.0.1:10000"
	SERVER_PORT = 10000
	CHUNK_SIZE  = 1024

	NAME         = "TEST"
	PROTO_PUSH   = "proto.in"
	PROTO_PULL   = "proto.out"
	HANDLER_PUSH = "handler.in"
	HANDLER_PULL = "handler.out"
)

func TestAll(t *testing.T) {
	CLUSTER := []string{"127.0.0.1:28015"}
	pushs := InitPushServer(CLUSTER, SERVER_PORT, CHUNK_SIZE)
	fmt.Println("Running pushserver...")
	go pushs.Run()
	time.Sleep(3 * time.Second)

	pushc := InitPushClient(SERVER_ADDR, CHUNK_SIZE)
	fmt.Println("Pushing from client...")
	pushc.Push(NAME, PROTO_PUSH, HANDLER_PUSH)

	spull := InitServerClient(CLUSTER)
	fmt.Println("Running pullclient as a server...")
	spull.Pull(NAME)

	lbpull := InitLBClient(CLUSTER)
	fmt.Println("Running pullclient as a balancer...")
	lbpull.Pull(NAME)
}
