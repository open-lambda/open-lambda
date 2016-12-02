// +build regpush

package main

import (
	"log"
	"os"

	r "github.com/open-lambda/open-lambda/registry/src"
)

const (
	CHUNK_SIZE = 1024
	HANDLER    = "handler"
)

func main() {
	if len(os.Args) != 4 {
		log.Fatal("Usage: pushserver <server_ip> <name> <file>")
	}
	server_ip := os.Args[1]
	name := os.Args[2]
	fname := os.Args[3]

	pushc := r.InitPushClient(server_ip, CHUNK_SIZE)

	handler := r.PushClientFile{Name: fname, Type: HANDLER}
	pushc.Push(name, handler)
}
